package recorder

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"video-factory/internal/domain/model"
	"video-factory/pkg/config"
	"video-factory/pkg/util"

	"github.com/rs/zerolog/log"
)

type Recorder struct {
	Config *config.AppConfig

	StreamURLs []string
	// CurrentURL      string
	CurrentURLIndex int

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

var timePattern = regexp.MustCompile(`time=\s*-?(\d+):(\d+):(\d+).(\d+)`)

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
			"-loglevel", "info", // 减少日志噪音
			"-stats",

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

		stderr, err := r.cmd.StderrPipe()
		if err != nil {
			log.Err(err).Msg("获取 stderr 失败")
			time.Sleep(2 * time.Second)
			continue
		}

		if err := r.cmd.Start(); err != nil {
			log.Err(err).Str("name", r.Username).Msg("[Recorder] 启动 ffmpeg 失败")
			time.Sleep(2 * time.Second)
			continue
		}

		// 记录开始时间
		startTime := time.Now()

		// 启动日志处理协程 (必须并发读取，否则会阻塞)
		go r.HandleStderr(stderr)

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

		log.Warn().Err(err).Str("file", r.File.Name()).Msgf("[Recorder] 录制中断，进行故障排查")

		runDuration := time.Since(startTime)
		if runDuration < 10*time.Second {
			r.rapidFailCnt++
			if r.rapidFailCnt > len(r.StreamURLs) {
				log.Error().Msg("[Recorder] 所有线路轮询失败，进入冷却模式 (60s)")
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(60 * time.Second): // 冷却 60 秒
					return fmt.Errorf("[Recorder] all streams failed after cooldown")
				}
			}
			log.Warn().Msgf("[Recorder] 当前流可能无效，切换线路")
			r.SwitchNextStream()
			log.Info().Msgf("[Recorder] 切换线路成功，准备重启启动")
			continue
		}

		log.Err(err).Msgf("[Recorder] 当前流异常，抛出错误到 Manager 进行处理")
		return err
	}
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
			if r.cmd != nil && r.cmd.Process != nil {
				_ = r.cmd.Process.Kill()
			}
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
				// 只杀进程，不 return，避免 goroutine 泄漏
				// 杀掉进程后，上面的 stdout.Read 会报错，从而触发 case err := <-errCh
				if r.cmd != nil && r.cmd.Process != nil {
					_ = r.cmd.Process.Kill()
				}
			}
		}
	}
}

func (r *Recorder) HandleStderr(stderr io.ReadCloser) {
	defer stderr.Close()

	scanner := bufio.NewScanner(stderr)
	scanner.Split(splitCRLF)
	for scanner.Scan() {
		line := scanner.Text()
		// log.Debug().Str("raw", line).Msg("ffmpeg_log")

		// 提取时间进度 time=00:01:23.45
		if matches := timePattern.FindStringSubmatch(line); len(matches) == 5 {
			h, _ := strconv.Atoi(matches[1])
			m, _ := strconv.Atoi(matches[2])
			s, _ := strconv.Atoi(matches[3])
			// ms, _ := strconv.Atoi(matches[4])

			// 更新当前录制时长 (秒)
			r.Duration = float64(h*3600 + m*60 + s)

			// 只有变化较大时才打印日志，防止刷屏（例如每10秒打印一次）
			if int(r.Duration)%10 == 0 {
				log.Info().Msgf("filename: %s, duration: %s, filesize: %s",
					r.File.Name(), util.FormatDuration(r.Duration), util.FormatFilesize(r.Filesize))
			}
			continue
		}

		// 2. 捕获错误日志 (FFmpeg 的错误通常包含 Error 或只在开头输出配置信息)
		// 过滤掉普通的 frame=... 进度信息，剩下的通常是关键日志
		if !strings.Contains(line, "frame=") {
			// 将 FFmpeg 的日志输出到你的 log 系统中
			log.Debug().Str("ffmpeg", "stderr").Msg(line)
		}
	}
}

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

func (r *Recorder) UpdateStreamURLs(newURLMap map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(newURLMap) == 0 {
		return
	}

	urls := make([]string, 0, len(newURLMap))
	for _, u := range newURLMap {
		urls = append(urls, u)
	}

	r.StreamURLs = urls
	r.CurrentURLIndex = 0

	log.Info().Str("name", r.Username).Msg("[Recorder] 内部流地址列表已热更新(等待下次重连生效)")
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

// splitCRLF 是一个自定义的 SplitFunc，同时支持 \n 和 \r 作为分隔符
// 这样才能实时读到 FFmpeg 的进度条
func splitCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	// 1. 优先处理 \r\n (标准日志换行) -> 丢弃 \r\n，只返回内容
	if i := bytes.Index(data, []byte{'\r', '\n'}); i >= 0 {
		return i + 2, data[0:i], nil
	}
	// 2. 处理 \n (Linux 换行)
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0:i], nil
	}
	// 3. 处理 \r (FFmpeg 进度条刷新)
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		return i + 1, data[0:i], nil
	}
	// 4. EOF
	if atEOF {
		return len(data), data, nil
	}
	// 5. 数据不够，请求更多
	return 0, nil, nil
}
