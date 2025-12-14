package recorder

import (
	"context"
	"errors"
	"fmt"
	"github.com/Eyevinn/hls-m3u8/m3u8"
	"github.com/rs/zerolog/log"
	"net/url"
	"os"
	"strings"
	"time"
	"video-factory/pkg/fetcher"
)

type Recorder struct {
	StreamURL string
	nextSeq   uint64 // 下一个期望下载的序列号

	f *os.File
}

func NewRecorder(streamURL string, path string) (*Recorder, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &Recorder{
		StreamURL: streamURL,
		// seenSegments: make(map[uint64]bool),
		f: file,
	}, nil
}

// Start 开始录制循环，阻塞直到 context 取消或发生致命错误
func (r *Recorder) Start(ctx context.Context) error {
	defer r.close()

	// 初始轮询间隔，给一个小值以便立即开始
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	log.Info().Str("url", r.StreamURL).Msg("开始 HLS 录制循环")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("录制任务收到停止信号")
			return nil
		case <-ticker.C:
			// 执行单次处理
			playlist, err := r.GetPlayList()

			// 动态调整下一次请求的时间
			interval := 2 * time.Second // 默认兜底间隔
			if err == nil && playlist != nil {
				// 官方建议：请求间隔 = TargetDuration
				// 如果追求低延迟，可以设置为 TargetDuration / 2，但不要太快
				if playlist.TargetDuration > 0 {
					interval = time.Duration(playlist.TargetDuration) * time.Second
				}
			} else {
				// 如果出错，稍微退避一下，避免死循环刷屏
				log.Warn().Err(err).Msg("获取或解析播放列表失败，稍后重试")
				interval = 5 * time.Second
			}

			// 重置定时器
			ticker.Reset(interval)
		}
	}
}

func (r *Recorder) GetPlayList() (*m3u8.MediaPlaylist, error) {
	if r.StreamURL == "" {
		return nil, errors.New("HLS source is empty")
	}

	// parsedURL, _ := url.Parse(r.StreamURL)

	// header := make(http.Header)
	// header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36")
	// header.Set("Referer", "https://fm.missevan.com/live/869032634")
	// header.Set("Origin", "https://fm.missevan.com")
	// header.Set("Accept-Encoding", "identity")
	// header.Set("Host", "d1-missevan04.bilivideo.com")

	bytes, err := fetcher.FetchBody(r.StreamURL, nil, nil)
	if err != nil {
		return nil, err
	}
	p, _, err := m3u8.DecodeFrom(strings.NewReader(string(bytes)), true)
	if err != nil {
		return nil, fmt.Errorf("failed to decode m3u8 playlist: %w", err)
	}

	// #EXTM3U
	// #EXT-X-VERSION:3
	// #EXT-X-ALLOW-CACHE:NO
	// #EXT-X-MEDIA-SEQUENCE:1765467796
	// #EXT-X-TARGETDURATION:2
	// #EXTINF:0.982,
	// maoer_30165838_869032634-1765467796.ts?txspiseq=108217735705553382345
	// #EXTINF:1.002,
	// maoer_30165838_869032634-1765467797.ts?txspiseq=108217735705553382345
	// #EXTINF:1.003,
	// maoer_30165838_869032634-1765467798.ts?txspiseq=108217735705553382345

	mediaPlaylist := p.(*m3u8.MediaPlaylist)
	defer mediaPlaylist.ReleasePlaylist()
	fmt.Print(mediaPlaylist)

	for _, segment := range mediaPlaylist.Segments {
		if segment == nil {
			continue
		}

		// 核心逻辑：只下载比当前序列号大的
		if segment.SeqId < r.nextSeq {
			continue
		}

		// 下载 TS (这里可以使用 retry-go)
		tsData, err := r.downloadTS(segment.URI)
		if err != nil {
			log.Error().Err(err).Msg("下载 TS 失败")
			// TS 下载失败是否跳过？通常建议重试几次，不行就跳过，避免阻塞
			continue
		}

		// 5. 写入文件 (包含分文件逻辑)
		if err := r.write(tsData); err != nil {
			return nil, err
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

func (r *Recorder) write(data []byte) error {
	if r.f == nil {
		return fmt.Errorf("file handle is nil")
	}
	_, err := r.f.Write(data)
	return err
}

func (r *Recorder) close() error {
	if r.f != nil {
		return r.f.Close()
	}
	return nil
}
