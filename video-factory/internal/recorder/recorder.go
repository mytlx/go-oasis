package recorder

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
	"video-factory/internal/domain/model"
	"video-factory/internal/iface"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"
	"video-factory/pkg/util"

	"github.com/Eyevinn/hls-m3u8/m3u8"
	"github.com/avast/retry-go/v5"
	"github.com/rs/zerolog/log"
)

type Recorder struct {
	Config    *config.AppConfig
	StreamURL string
	nextSeq   uint64 // 下一个期望下载的序列号

	File       *os.File
	Username   string
	StreamAt   int64
	Sequence   int
	RoomRealId string
	Duration   float64
	Filesize   int
}

func NewRecorder(cfg *config.AppConfig, streamURL string, room *model.Room, openTime int64) (*Recorder, error) {
	return &Recorder{
		Config:    cfg,
		StreamURL: streamURL,
		Username:  room.AnchorName,
		StreamAt:  openTime,
	}, nil
}

// Start 开始录制循环，阻塞直到 context 取消或发生致命错误
func (r *Recorder) Start(ctx context.Context) error {
	// 初始轮询间隔，给一个小值以便立即开始
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	if err := r.NextFile(); err != nil {
		return fmt.Errorf("next file: %w", err)
	}

	log.Info().Str("filename", r.File.Name()).Str("url", r.StreamURL).Msg("开始录制")

	// Ensure file is cleaned up when this function exits in any case
	defer func() {
		if err := r.Cleanup(); err != nil {
			log.Err(err).Msgf("cleanup on record stream exit")
		}
	}()

	retryCnt := 0
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("录制任务收到停止信号")
			return nil
		case <-ticker.C:
			// 执行单次处理
			playlist, err := r.ProcessSegments(ctx)

			// 动态调整下一次请求的时间
			interval := 2 * time.Second // 默认兜底间隔
			if err == nil && playlist != nil {
				retryCnt = 0
				// 官方建议：请求间隔 = TargetDuration
				// 如果追求低延迟，可以设置为 TargetDuration / 2，但不要太快
				if playlist.TargetDuration > 0 {
					interval = time.Duration(playlist.TargetDuration) * time.Second
				}
			} else {
				// 如果出错，稍微退避一下，避免死循环刷屏
				log.Err(err)
				retryCnt += 1
				if retryCnt > 3 {
					return err
				}
				interval = 1 * time.Second
			}

			// 重置定时器
			ticker.Reset(interval)
		}
	}
}

func (r *Recorder) ProcessSegments(ctx context.Context) (*m3u8.MediaPlaylist, error) {
	if r.StreamURL == "" {
		return nil, errors.New("HLS source is empty")
	}

	bytes, err := fetcher.FetchBody(r.StreamURL, nil, nil)
	if err != nil {
		return nil, err
	}
	p, _, err := m3u8.DecodeFrom(strings.NewReader(string(bytes)), true)
	if err != nil {
		return nil, fmt.Errorf("failed to decode m3u8 playlist: %w", err)
	}
	mediaPlaylist, ok := p.(*m3u8.MediaPlaylist)
	if !ok {
		return nil, fmt.Errorf("cannot cast to media playlist")
	}
	defer mediaPlaylist.ReleasePlaylist()
	// fmt.Print(mediaPlaylist)

	for _, segment := range mediaPlaylist.Segments {
		if segment == nil {
			continue
		}

		// 核心逻辑：只下载比当前序列号大的
		if segment.SeqId < r.nextSeq {
			continue
		}

		resp, err := retry.NewWithData[[]byte](
			retry.Attempts(3),
			retry.Delay(100),
			retry.OnRetry(
				func(n uint, err error) {
					log.Err(err).Msgf("[Recorder segment] 第%d次重试 start", n+1)
				},
			),
			retry.RetryIf(func(err error) bool {
				if errors.Is(err, iface.ErrRoomOffline) {
					// 不重试
					return false
				}
				return true
			}),
			retry.Context(ctx),
		).Do(func() ([]byte, error) {
			return r.downloadTS(segment.URI)
		})
		if err != nil {
			return nil, retry.Unrecoverable(err)
		}

		// 写入文件
		n, err := r.File.Write(resp)
		if err != nil {
			return nil, retry.Unrecoverable(fmt.Errorf("write file: %w", err))
		}

		r.Filesize += n
		r.Duration += segment.Duration
		log.Info().Msgf("filename: %s, duration: %s, filesize: %s", r.File.Name(), util.FormatDuration(r.Duration), util.FormatFilesize(r.Filesize))

		if r.ShouldSwitchFile() {
			if err := r.NextFile(); err != nil {
				return nil, fmt.Errorf("next file: %w", err)
			}
			log.Info().Msgf("max filesize or duration exceeded, new file created: %s", r.File.Name())
			return mediaPlaylist, nil
		}

		// 更新序列号
		r.nextSeq = segment.SeqId + 1
	}

	return mediaPlaylist, nil
}

func (r *Recorder) downloadTS(tsURL string) ([]byte, error) {
	// http://d1-missevan104.bilivideo.com/live-bvc/489331/maoer_30165838_869032634-1765470203.ts?txspiseq=108217735705553382345
	parsedURL, _ := url.Parse(r.StreamURL)
	parsedTsURL, _ := url.Parse(tsURL)
	finalURL := parsedURL.ResolveReference(parsedTsURL)

	return fetcher.FetchBody(finalURL.String(), nil, nil)
}
