package manager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"video-factory/internal/common/consts"
	"video-factory/internal/domain/model"
	"video-factory/internal/iface"
	"video-factory/internal/recorder"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"

	"github.com/avast/retry-go/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	MaxAttemptTimes       = 10
	RetryWaitDuration     = 2 * time.Second
	refreshSafetyInterval = 1 * time.Minute
)

type Manager struct {
	Config   *config.AppConfig
	Streamer iface.Streamer `json:"-"`
	Room     *model.Room

	Id               int64
	Platform         string
	OpenTime         int64
	CurrentURL       string
	ProxyURL         string
	StreamURLMap     map[string]string
	ActualExpireTime time.Time
	SafetyExpireTime time.Time
	LastRefreshTime  time.Time

	cancel    context.CancelFunc // 用于触发停止信号
	refreshCh chan struct{}      // 用于通知 AutoRefresh 循环立即执行一次刷新（如首次启动或外部命令）
	ctx       context.Context    // manager 的生命周期
	onStop    func(int64)        // 停止回调

	Recorder     *recorder.Recorder // 持有录制器实例
	RecordStatus int                // 是否开启录制（来自 Room 配置）
	recordCancel context.CancelFunc // 用于单独停止录制任务

	mu sync.RWMutex
}

func NewManager(room *model.Room, config *config.AppConfig, onStop func(int64)) (*Manager, error) {
	if room == nil {
		return nil, errors.New("room is nil")
	}
	var s iface.Streamer
	switch room.Platform {
	case consts.PlatformBili:
		s = bili.NewStreamer(room.RealID, config)
	case consts.PlatformMissevan:
		s = missevan.NewStreamer(room.RealID, config)
	default:
		return nil, errors.New("invalid platform")
	}

	// 类型断言，尝试将 s 转为 ConfigSubscriber
	if subscriber, ok := s.(iface.ConfigSubscriber); ok {
		log.Info().Msgf("注册 streamer 为 config 订阅者")
		config.AddSubscriber(subscriber)
	}

	m := &Manager{
		Config:           config,
		Streamer:         s,
		Room:             room,
		OpenTime:         s.GetOpenTime(),
		Id:               room.ID,
		Platform:         room.Platform,
		ProxyURL:         room.ProxyURL,
		ActualExpireTime: time.Now(),
		SafetyExpireTime: time.Now(),
		RecordStatus:     room.RecordStatus,
		onStop:           onStop,
	}

	log.Info().Object("manager", m).Msg("[Manager] Init Manager")
	return m, nil
}

// StartAutoRefresh 启动一个 Goroutine，根据过期时间自动刷新 Manager 状态
func (m *Manager) StartAutoRefresh(parentCtx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 确保只启动一次
	if m.cancel != nil {
		log.Warn().Int64("id", m.Id).Msg("[Manager] 自动刷新服务已在运行。")
		return
	}

	// 初始化 Context 用于控制 Goroutine 生命周期
	childCtx, cancel := context.WithCancel(parentCtx)
	m.cancel = cancel
	m.refreshCh = make(chan struct{}, 1) // 有缓冲，防止发送阻塞
	m.ctx = childCtx

	log.Info().Int64("id", m.Id).Msg("[Manager AutoRefresh] 启动自动刷新服务")

	// 启动 Goroutine
	go m.autoRefreshLoop()
}

// StopAutoRefresh 发送停止信号给自动刷新 Goroutine
func (m *Manager) StopAutoRefresh() {
	if m.cancel != nil {
		m.cancel() // 调用 context 的 CancelFunc 触发停止
		m.cancel = nil
		// refreshCh 在 autoRefreshLoop 退出后应该被关闭，这里不必显式关闭
	}
}

// TriggerRefresh 发送信号给循环，使其立即执行一次刷新
func (m *Manager) TriggerRefresh() {
	select {
	case m.refreshCh <- struct{}{}:
		log.Info().Int64("id", m.Id).Msg("[Manager] 手动触发即时刷新信号。")
	case <-m.ctx.Done():
		log.Warn().Msg("[Manager] Manager 已停止，忽略刷新请求")
	default:
		// 如果通道已满，说明循环正在忙或等待，忽略本次触发
		log.Debug().Int64("id", m.Id).Msg("[Manager] 即时刷新信号发送失败，循环正忙。")
	}
}

// autoRefreshLoop 是 AutoRefresh 的核心循环
func (m *Manager) autoRefreshLoop() {
	defer func() {
		// 循环退出时关闭 Channel
		close(m.refreshCh)
		// 循环退出时（下播或异常），触发回调通知 Pool 移除自己
		if m.onStop != nil {
			log.Info().Int64("id", m.Id).Msg("[Manager] Manager 停止，触发 onStop 回调")
			m.onStop(m.Id)
		}
		// 停止录制
		if m.recordCancel != nil {
			m.recordCancel()
			m.Recorder = nil
		}
	}()

	// 立即触发一次初始刷新，确保启动时就有有效的URL
	m.TriggerRefresh()

	for {
		m.mu.RLock()
		// 核心计算：等待时间 = 预期过期时间 - 当前时间 - 安全提前量
		waitTime := m.SafetyExpireTime.Sub(time.Now()) - refreshSafetyInterval
		isFirstRun := m.LastRefreshTime.IsZero()
		m.mu.RUnlock()

		if waitTime < 0 {
			if isFirstRun {
				waitTime = 5 * time.Second
				log.Info().Msg("[Manager AutoRefresh] 首次启动，准备立即刷新")
			} else {
				// 如果计算出负值（已过期或配置的安全间隔太长），则等待一个短的重试时间
				waitTime = 3 * time.Second
				log.Warn().Int64("id", m.Id).Msgf("[Manager AutoRefresh] 链接已过期或即将过期，立即等待 %s 后重试。", waitTime)
			}
		}

		timer := time.NewTimer(waitTime)

		select {
		case <-m.ctx.Done():
			// 收到停止信号，退出循环
			timer.Stop()
			log.Info().Int64("id", m.Id).Msg("[Manager AutoRefresh] 自动刷新服务已优雅停止。")
			return // 退出 Goroutine

		case <-m.refreshCh:
			// 收到立即刷新信号（手动触发或首次启动）
			timer.Stop()
			log.Info().Int64("id", m.Id).Msg("[Manager AutoRefresh] 收到即时刷新信号，立即刷新。")
			// 继续执行刷新逻辑

		case <-timer.C:
			log.Info().Int64("id", m.Id).Msg("[Manager AutoRefresh] 刷新间隔到期，开始刷新。")
			// 定时器到期，执行刷新逻辑
			// 继续执行刷新逻辑
		}

		// --- 核心刷新执行 ---
		err := m.CommonRefresh(nil, MaxAttemptTimes)

		// 检测是否下播
		if errors.Is(err, iface.ErrRoomOffline) {
			log.Info().Int64("id", m.Id).Msg("[Manager AutoRefresh] 检测到直播结束，自动停止 Manager")
			// 这里不需要调用 StopAutoRefresh，直接 return 即可退出循环
			return
		}

		if err != nil {
			log.Err(err).Int64("id", m.Id).Msg("[Manager AutoRefresh] 自动刷新失败，将在下一轮循环中重试。")
			// 如果是其他错误，可以选择继续重试，或者设置一个连续失败阈值来退出
		}
	}
}

// CommonRefresh 通用 Refresh 函数，负责控制流、重试和状态更新
func (m *Manager) CommonRefresh(tempCtx context.Context, attempts int) error {
	log.Info().Msg("[Manager CommonRefresh] 正在刷新直播流 token...")

	currentCtx := m.ctx
	if tempCtx != nil {
		currentCtx = tempCtx
	}

	// 边界检查
	if attempts < 0 {
		attempts = 1
	}
	if attempts > MaxAttemptTimes {
		attempts = MaxAttemptTimes
	}

	var newStreamUrl string
	var newExpireTime time.Time

	r := retry.New(
		retry.Attempts(uint(attempts)),
		retry.Delay(RetryWaitDuration),
		retry.OnRetry(
			func(n uint, err error) {
				log.Err(err).Msgf("[Manager CommonRefresh] 第%d次重试 start", n+1)
			},
		),
		retry.RetryIf(func(err error) bool {
			if errors.Is(err, iface.ErrRoomOffline) {
				// 不重试
				return false
			}
			return true
		}),
		retry.Context(currentCtx),
	)
	err := r.Do(func() error {
		// --- 1. 业务逻辑调用（通过策略接口） ---
		streamInfo, fetchErr := m.Streamer.FetchStreamInfo(m.Streamer.GetStreamInfo().SelectedQn, true)
		if fetchErr != nil {
			log.Err(fetchErr).Msg("[Manager CommonRefresh] 刷新直播流信息失败:")
			return fetchErr
		}

		// --- 2. 业务逻辑调用（通过策略接口） ---
		// 尝试解析 Stream URL
		for _, streamUrl := range streamInfo.StreamUrls {
			expireTime, parseErr := m.Streamer.ParseExpiration(streamUrl)
			if parseErr != nil {
				log.Err(parseErr).Msg("[Manager CommonRefresh] 解析 expireTime 失败")
				continue
			}
			newStreamUrl = streamUrl
			newExpireTime = expireTime
			return nil
		}
		return errors.New("[Manager CommonRefresh] 解析 expireTime 失败")
	})

	// 检查是否所有重试都失败
	if newStreamUrl == "" || err != nil {
		log.Err(err).Msg("[Manager CommonRefresh] 所有重试均失败，上次错误")
		return err
	}

	// --- 3. 通用状态更新和加锁 ---
	m.mu.Lock()
	m.CurrentURL = newStreamUrl
	m.StreamURLMap = m.Streamer.GetStreamInfo().StreamUrls
	m.ActualExpireTime = newExpireTime
	m.SafetyExpireTime = newExpireTime.Add(-1 * time.Minute)
	m.LastRefreshTime = time.Now()
	m.mu.Unlock()

	log.Info().Msg("[Manager CommonRefresh] 更新成功")
	log.Info().Object("manager", m).Msg("[Manager CommonRefresh] Manager")

	// 核心联动逻辑：URL 变了，或者录制没启动，就去处理一下
	if m.RecordStatus == 1 {
		// 异步启动，不要阻塞刷新主流程
		go m.updateRecorder()
	}

	return nil
}

// ResolveTargetURL 根据请求的文件名（相对路径），计算出上游直播流的完整 URL
func (m *Manager) ResolveTargetURL(filename string) (string, error) {
	// 1. 获取当前的基础流地址
	currentHls := m.CurrentURL
	if currentHls == "" {
		return "", fmt.Errorf("current stream url is empty")
	}

	parsedHlsUrl, err := url.Parse(currentHls)
	if err != nil {
		return "", fmt.Errorf("parse current hls url failed: %w", err)
	}

	// 2. 如果请求的是 m3u8，直接返回当前流地址
	// 注意：这里简单的通过后缀判断，如果文件名为空或者就是 endpoint 本身，通常也返回 m3u8
	if filename == "" || strings.HasSuffix(filename, ".m3u8") {
		return currentHls, nil
	}

	// 3. 如果是分片 (ts/m4s)，需要拼接相对路径
	if strings.HasSuffix(filename, ".ts") || strings.HasSuffix(filename, ".m4s") {
		// 复制一份 parsedHlsUrl 用于修改，避免影响原始对象（虽然 Parse 返回的是指针，但我们修改 Path）
		baseUrl := *parsedHlsUrl

		// HLS 协议中，分片的相对路径是相对于 m3u8 文件所在的目录
		// 例如：http://example.com/live/stream.m3u8 -> Base 是 http://example.com/live/
		lastSlash := strings.LastIndex(baseUrl.Path, "/")
		if lastSlash != -1 {
			baseUrl.Path = baseUrl.Path[:lastSlash+1]
		}

		// 解析请求的文件名（它可能是 "seg-1.ts" 也可能是 "sub_dir/seg-1.ts"）
		relativeURL, err := url.Parse(strings.TrimPrefix(filename, "/"))
		if err != nil {
			return "", fmt.Errorf("parse relative filename failed: %w", err)
		}

		// 使用 ResolveReference 处理路径拼接 (自动处理 ./ ../ 等)
		targetURL := baseUrl.ResolveReference(relativeURL)

		// 4. 关键：保留原始 m3u8 的 Query 参数 (Token/签名)
		// 很多直播流的鉴权 Token 是跟在 m3u8 后面的，分片下载也需要带上
		targetURL.RawQuery = parsedHlsUrl.RawQuery

		return targetURL.String(), nil
	}

	return "", fmt.Errorf("unsupported file type: %s", filename)
}

// MarshalZerologObject 实现 zerolog.LogObjectMarshaler 接口
// 调用 log.Object("manager", m) 时，哪些字段会被打印
func (m *Manager) MarshalZerologObject(e *zerolog.Event) {
	// 只记录关键的业务字段，跳过锁、Context、通道等无关字段
	e.Int64("id", m.Id).
		Str("current_url", m.CurrentURL).
		Str("proxy_url", m.ProxyURL).
		Time("actual_expire_time", m.ActualExpireTime).
		Time("safety_expire_time", m.SafetyExpireTime).
		Time("last_refresh_time", m.LastRefreshTime).
		Int("record_status", m.RecordStatus).
		Int64("open_time", m.OpenTime)
}

// ---------------------------------------------------------------------------------------------------------------------

// Fetch 封装了带有自动刷新 (Refresh) 功能的 HTTP 请求
// 它会自动从 Streamer 获取 Headers，并处理 403/401 触发的 Token 刷新
func (m *Manager) Fetch(ctx context.Context, urlStr string, params url.Values) (*http.Response, error) {
	// 定义执行器：真正发起请求的函数
	executor := func(method, baseURL string, p url.Values) (*http.Response, error) {
		// 1. 从 Streamer 获取特定平台的 Headers (核心改动)
		headers := m.Streamer.GetHeaders()

		// 2. 如果有代理配置，也可以在这里通过 config 获取并设置
		// if m.ProxyURL != "" { ... }

		return fetcher.Fetch(method, baseURL, p, headers)
	}

	return fetcher.FetchWithRefresh(ctx, m, executor, "GET", urlStr, params)
}

// Refresh 实现 fetcher.Refresher 接口，用于 FetchWithRefresh 调用
func (m *Manager) Refresh(ctx context.Context, attempts int) error {
	// 调用自身的 CommonRefresh，传入保存的配置
	return m.CommonRefresh(ctx, attempts)
}
