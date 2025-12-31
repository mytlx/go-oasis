package recorder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
	"video-factory/internal/domain/model"
	"video-factory/pkg/config"

	"github.com/rs/zerolog/log"
)

type Recorder struct {
	Config *config.AppConfig

	StreamURLs []string
	// CurrentURL      string
	CurrentURLIndex int

	// StreamURL        string
	// Deprecated
	nextSeq uint64 // 下一个期望下载的序列号

	LastActivityUnix int64 // 最后一次成功写入数据的时间

	File       *os.File
	Username   string
	StreamAt   int64
	Sequence   int
	RoomRealId string
	Duration   float64
	Filesize   int
	Ext        string

	rapidFailCnt int // 连续快速失败的次数
	running      atomic.Bool
	mu           sync.RWMutex
	cmd          *exec.Cmd
}

func NewRecorder(cfg *config.AppConfig, streamURLMap map[string]string, room *model.Room, openTime int64) (*Recorder, error) {
	if len(streamURLMap) == 0 {
		return nil, fmt.Errorf("stream urls is empty")
	}

	urls := make([]string, 0, len(streamURLMap))
	for _, u := range streamURLMap {
		urls = append(urls, u)
	}

	return &Recorder{
		Config:          cfg,
		StreamURLs:      urls,
		CurrentURLIndex: 0,
		Username:        room.AnchorName,
		StreamAt:        openTime,
		Ext:             "ts",
	}, nil
}

const (
	stallTimeout   = 1 * time.Minute // 超时阈值
	readBufferSize = 32 * 1024       // 32kb 读取缓冲
)

// Start 开始录制循环，阻塞直到 context 取消或发生致命错误
func (r *Recorder) Start(ctx context.Context) error {
	if err := r.NextFile(); err != nil {
		return fmt.Errorf("next file: %w", err)
	}
	r.rapidFailCnt = 0
	r.running.Store(true)

	// Ensure file is cleaned up when this function exits in any case
	defer func() {
		r.running.Store(false)
		if err := r.Cleanup(); err != nil {
			log.Err(err).Msgf("cleanup on record stream exit")
		}
	}()
	log.Info().Str("filename", r.File.Name()).Msg("[recorder] 开始录制")

	// -------------------------------------------------------
	// 负责【掉线/切换线路后的重启】
	// -------------------------------------------------------
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("file", r.File.Name()).Msg("[recorder] 录制任务收到停止信号")
			return nil
		default:
			// 向下执行
		}

		currentURL := r.GetCurrentURL()
		log.Info().Str("url", currentURL).Msg("[Recorder] 启动 FFmpeg 录制进程")

		// ========== 构造 FFmpeg 命令 ==========
		args := []string{
			"-y", "-hide_banner",
			"-loglevel", "error", // 减少日志噪音

			// --- 网络重连参数 (必须放在 -i 之前) ---
			"-reconnect", "1", // 当底层 TCP 连接意外断开时，尝试重连
			"-reconnect_at_eof", "1", // 在读取到流的结尾（EOF）时尝试重连，避免直播抖动
			"-reconnect_streamed", "1", // 专门针对流媒体（Infinite Stream）启用重连
			"-reconnect_delay_max", "5", // 重连尝试的最大等待时间5秒
			// --- header 伪装 ---
			"-user_agent", "Mozilla/5.0 (iPod; CPU iPhone OS 14_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.163 Mobile/15E148 Safari/604.1",
			// "-headers", "Referer: https://live.bilibili.com/\r\n"
			// --- 输入 ---
			"-i", currentURL,
			// --- 输出 ---
			"-c", "copy", // 直接复制流，不转码（CPU占用低）
			"-f", "mpegts", // 强制封装格式为 TS
			"pipe:1", // 输出到标准输出
		}

		r.cmd = exec.CommandContext(ctx, "ffmpeg", args...)

		stdout, err := r.cmd.StdoutPipe()
		if err != nil {
			log.Err(err).Str("name", r.Username).Msg("[Recorder] 获取 ffmpeg stdout 失败，等待重试")
			time.Sleep(2 * time.Second)
			continue
		}

		// 获取标准错误管道（用于调试 ffmpeg 报错）
		// stderr, _ := r.cmd.StderrPipe()
		// 可以开个 goroutine 打印 stderr，方便排查 403 等错误

		if err := r.cmd.Start(); err != nil {
			log.Err(err).Str("name", r.Username).Msg("[Recorder] 启动 ffmpeg 失败")
			time.Sleep(2 * time.Second)
			continue
		}

		// 记录开始时间
		startTime := time.Now()

		// 读取管道数据到文件
		err = r.readPipe(ctx, stdout)
		if err != nil {
			log.Err(err).Str("file", r.File.Name()).Msgf("[Recorder] 录制中断")
		}

		// 等待进程彻底结束
		_ = r.cmd.Wait()

		// ========== 故障分析与切换 ==========

		// context 取消，直接退出
		if ctx.Err() != nil {
			log.Info().Str("file", r.File.Name()).Msg("[recorder] 录制任务已停止，原因：收到停止信号")
			return nil
		}

		log.Warn().Err(err).Str("file", r.File.Name()).Msgf("[Recorder] 录制中断，准备重试")

		// 只有在非正常退出时，才考虑切换线路
		// 简单的策略：只要断了，就切下一个线路试试
		// 进阶策略：可以判断错误类型，如果是网络超时才切
		time.Sleep(1 * time.Second) // 避免疯狂重启
		r.SwitchNextStream()
		runDuration := time.Since(startTime)
		if runDuration < 10*time.Second {
			r.rapidFailCnt++
			if r.rapidFailCnt > len(r.StreamURLs) {
				log.Error().Msgf("[Recorder] 所有线路均不可用")
				return err
			}
			log.Warn().Msgf("当前流可能无效，切换线路")
			r.SwitchNextStream()
		}
		return err
	}

	// retryCnt := 0
	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		log.Info().Msg("[recorder] 录制任务收到停止信号")
	// 		return nil
	// 	case <-ticker.C:
	// 		// 看门狗机制
	// 		if time.Since(r.lastActivityUnix) > stallTimeout {
	// 			log.Error().
	// 				Str("filename", r.File.Name()).
	// 				Str("url", r.StreamURL).
	// 				Time("last_active", r.lastActivityUnix).
	// 				Msg("[recorder] 检测到直播流长时间未更新(僵尸流)，自动终止录制任务")
	// 			return fmt.Errorf("stream stalled for %v", stallTimeout)
	// 		}
	//
	// 		// 执行单次处理
	// 		playlist, err := r.ProcessSegments(ctx)
	//
	// 		// 动态调整下一次请求的时间
	// 		interval := 2 * time.Second // 默认兜底间隔
	// 		if err == nil && playlist != nil {
	// 			retryCnt = 0
	// 			// 官方建议：请求间隔 = TargetDuration
	// 			// 如果追求低延迟，可以设置为 TargetDuration / 2，但不要太快
	// 			if playlist.TargetDuration > 0 {
	// 				interval = time.Duration(playlist.TargetDuration) * time.Second
	// 			}
	// 		} else {
	// 			// 如果出错，稍微退避一下，避免死循环刷屏
	// 			log.Err(err).
	// 				Str("hls", r.StreamURL).
	// 				Str("name", r.Username).
	// 				Msgf("[recorder] 获取流信息失败，重试中")
	// 			retryCnt += 1
	// 			if retryCnt > 3 {
	// 				return err
	// 			}
	// 			interval = 1 * time.Second
	// 		}
	//
	// 		// 重置定时器
	// 		ticker.Reset(interval)
	// 	}
	// }
}

func (r *Recorder) readPipe(ctx context.Context, stdout io.ReadCloser) error {
	defer stdout.Close()

	errCh := make(chan error, 1)
	r.LastActivityUnix = time.Now().Unix()
	go func() {
		defer close(errCh)
		buf := make([]byte, readBufferSize)

		for {
			// 阻塞读取
			n, readErr := stdout.Read(buf)
			if n > 0 {
				// 喂狗，更新活跃时间
				atomic.StoreInt64(&r.LastActivityUnix, time.Now().Unix())

				// 写入本地文件
				if _, wErr := r.File.Write(buf[:n]); wErr != nil {
					errCh <- fmt.Errorf("[Recorder] write file error: %w", wErr)
					return
				}

				// 更新统计信息
				r.Filesize += n
				// tlxTODO: 时间统计

				if r.ShouldSwitchFile() {
					if err := r.NextFile(); err != nil {
						log.Err(err).Str("file", r.File.Name()).Msg("[Recorder] 切换文件失败")
					}
					log.Info().Msgf("max filesize or duration exceeded, new file created: %s", r.File.Name())
				}
			}
			if readErr != nil {
				if readErr == io.EOF {
					errCh <- fmt.Errorf("ffmpeg stream EOF")
					return
				}
				errCh <- readErr
				return
			}

		}
	}()

	// ========== 看门狗机制 ==========
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = r.cmd.Process.Kill()
			return context.Canceled
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-ticker.C:
			last := atomic.LoadInt64(&r.LastActivityUnix)
			duration := time.Now().Unix() - last
			if duration > int64(stallTimeout.Seconds()) {
				log.Error().
					Str("filename", r.File.Name()).
					Str("url", r.GetCurrentURL()).
					Time("last_active", time.Unix(last, 0)).
					Msg("[recorder] 检测到直播流长时间未更新(僵尸流)，自动终止录制任务")
				_ = r.cmd.Process.Kill()
				return fmt.Errorf("stream stalled for %v", stallTimeout)
			}
		}
	}

}

// Deprecated
// func (r *Recorder) ProcessSegments(ctx context.Context) (*m3u8.MediaPlaylist, error) {
// 	currentURL := r.GetCurrentURL()
// 	if currentURL == "" {
// 		return nil, errors.New("HLS source is empty")
// 	}
//
// 	bytes, err := fetcher.FetchBody(currentURL, nil, nil)
// 	if err != nil {
// 		return nil, err
// 	}
// 	p, _, err := m3u8.DecodeFrom(strings.NewReader(string(bytes)), true)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to decode m3u8 playlist: %w", err)
// 	}
// 	mediaPlaylist, ok := p.(*m3u8.MediaPlaylist)
// 	if !ok {
// 		return nil, fmt.Errorf("cannot cast to media playlist")
// 	}
// 	defer mediaPlaylist.ReleasePlaylist()
// 	// fmt.Print(mediaPlaylist)
//
// 	for _, segment := range mediaPlaylist.Segments {
// 		if segment == nil {
// 			continue
// 		}
//
// 		// 核心逻辑：只下载比当前序列号大的
// 		if segment.SeqId < r.nextSeq {
// 			continue
// 		}
//
// 		resp, err := retry.NewWithData[[]byte](
// 			retry.Attempts(3),
// 			retry.Delay(100),
// 			retry.OnRetry(
// 				func(n uint, err error) {
// 					log.Err(err).Msgf("[Recorder segment] 第%d次重试 start", n+1)
// 				},
// 			),
// 			retry.RetryIf(func(err error) bool {
// 				if errors.Is(err, iface.ErrRoomOffline) {
// 					// 不重试
// 					return false
// 				}
// 				return true
// 			}),
// 			retry.Context(ctx),
// 		).Do(func() ([]byte, error) {
// 			return r.downloadTS(currentURL, segment.URI)
// 		})
// 		if err != nil {
// 			return nil, retry.Unrecoverable(err)
// 		}
//
// 		// 写入文件
// 		n, err := r.File.Write(resp)
// 		if err != nil {
// 			return nil, retry.Unrecoverable(fmt.Errorf("write file: %w", err))
// 		}
//
// 		r.Filesize += n
// 		r.Duration += segment.Duration
// 		log.Info().Msgf("filename: %s, duration: %s, filesize: %s", r.File.Name(), util.FormatDuration(r.Duration), util.FormatFilesize(r.Filesize))
//
// 		if n > 0 {
// 			// 更新活跃时间
// 			r.lastActivityUnix = time.Now()
// 		}
//
// 		if r.ShouldSwitchFile() {
// 			if err := r.NextFile(); err != nil {
// 				return nil, fmt.Errorf("next file: %w", err)
// 			}
// 			log.Info().Msgf("max filesize or duration exceeded, new file created: %s", r.File.Name())
// 			return mediaPlaylist, nil
// 		}
//
// 		// 更新序列号
// 		r.nextSeq = segment.SeqId + 1
// 	}
//
// 	return mediaPlaylist, nil
// }

// Deprecated
// func (r *Recorder) downloadTS(baseURL string, tsURL string) ([]byte, error) {
// 	// http://d1-missevan104.bilivideo.com/live-bvc/489331/maoer_30165838_869032634-1765470203.ts?txspiseq=108217735705553382345
// 	parsedURL, _ := url.Parse(baseURL)
// 	parsedTsURL, _ := url.Parse(tsURL)
// 	finalURL := parsedURL.ResolveReference(parsedTsURL)
//
// 	return fetcher.FetchBody(finalURL.String(), nil, nil)
// }

// func (r *Recorder) UpdateURL(newURL string) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	if r.StreamURL != newURL {
// 		log.Info().Str("old", r.StreamURL).Str("new", newURL).Msg("[Recorder] 热更新流地址")
// 		r.StreamURL = newURL
// 	}
// }
//
// func (r *Recorder) GetSafeURL() string {
// 	r.mu.RLock()
// 	defer r.mu.RUnlock()
// 	return r.StreamURL
// }

func (r *Recorder) GetCurrentURL() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.StreamURLs) == 0 {
		return ""
	}
	if r.CurrentURLIndex > len(r.StreamURLs) {
		r.CurrentURLIndex = 0
	}

	return r.StreamURLs[r.CurrentURLIndex]
}

func (r *Recorder) SwitchNextStream() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.StreamURLs) <= 1 {
		r.CurrentURLIndex = 0
		return r.StreamURLs[0]
	}

	r.CurrentURLIndex = (r.CurrentURLIndex + 1) % len(r.StreamURLs)
	newUrl := r.StreamURLs[r.CurrentURLIndex]
	log.Info().Str("newUrl", newUrl).Msgf("[Recorder] 切换到下一条线路")

	return newUrl
}

func (r *Recorder) IsRunning() bool {
	return r.running.Load()
}
