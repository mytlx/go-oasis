package manager

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
	"sync"
	"time"
	"video-factory/streamer"
)

type Manager struct {
	Id               string
	Streamer         streamer.Streamer `json:"-"`
	CurrentURL       string
	ActualExpireTime time.Time
	SafetyExpireTime time.Time
	LastRefresh      time.Time
	IManager         IManager           `json:"-"` // 持有接口，可以使用外部逻辑
	ctx              context.Context    // 用于控制 Goroutine 的停止信号
	cancel           context.CancelFunc // 用于触发停止信号
	refreshCh        chan struct{}      // 用于通知 AutoRefresh 循环立即执行一次刷新（如首次启动或外部命令）
	Mutex            sync.RWMutex       `json:"-"`
}

type IManager interface {
	AutoRefresh()
	Refresh(retryTimes int) error
	Fetch(baseURL string, params url.Values, extraHeader http.Header) (*http.Response, error)

	Get() *Manager
}

// RefreshStrategy 定义了刷新核心业务逻辑的策略
type RefreshStrategy interface {
	// ExecuteFetchStreamInfo 负责执行具体的网络请求和数据解析
	ExecuteFetchStreamInfo() (*streamer.StreamInfo, error)
	// ParseExpiration 从 URL 字符串中解析出过期时间
	ParseExpiration(streamUrl string) (time.Time, error)
}

// ConditionFunc 定义了检查是否需要执行动作的函数签名
type ConditionFunc func() bool

// ActionFunc 定义了条件满足时需要执行的动作函数签名
type ActionFunc func() error

// StartAutoRefresh 启动一个 Goroutine，根据过期时间自动刷新 Manager 状态
// interval 是一个安全提前量，例如提前 5 秒或 5 分钟
func (m *Manager) StartAutoRefresh(interval time.Duration) {
	// 确保只启动一次
	if m.cancel != nil {
		log.Warn().Str("id", m.Id).Msg("自动刷新服务已在运行。")
		return
	}

	// 初始化 Context 用于控制 Goroutine 生命周期
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.refreshCh = make(chan struct{}, 1) // 有缓冲，防止发送阻塞

	log.Info().Str("id", m.Id).Msg("启动自动刷新服务")

	// 启动 Goroutine
	go m.autoRefreshLoop(interval)
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
		log.Info().Str("id", m.Id).Msg("手动触发即时刷新信号。")
	default:
		// 如果通道已满，说明循环正在忙或等待，忽略本次触发
		log.Debug().Str("id", m.Id).Msg("即时刷新信号发送失败，循环正忙。")
	}
}

// autoRefreshLoop 是 AutoRefresh 的核心循环
func (m *Manager) autoRefreshLoop(refreshSafetyInterval time.Duration) {
	defer close(m.refreshCh) // 循环退出时关闭 Channel

	// 立即触发一次初始刷新，确保启动时就有有效的URL
	// m.TriggerRefresh()

	for {
		m.Mutex.RLock()
		// 核心计算：等待时间 = 预期过期时间 - 当前时间 - 安全提前量
		waitTime := m.SafetyExpireTime.Sub(time.Now()) - refreshSafetyInterval
		m.Mutex.RUnlock()

		if waitTime < 0 {
			// 如果计算出负值（已过期或配置的安全间隔太长），则等待一个短的重试时间
			waitTime = 5 * time.Second
			log.Warn().Str("id", m.Id).Msgf("链接已过期或即将过期，立即等待 %s 后重试。", waitTime)
		}

		timer := time.NewTimer(waitTime)

		select {
		case <-m.ctx.Done():
			// 收到停止信号，退出循环
			timer.Stop()
			log.Info().Str("id", m.Id).Msg("自动刷新服务已优雅停止。")
			return // 退出 Goroutine

		case <-m.refreshCh:
			// 收到立即刷新信号（手动触发或首次启动）
			timer.Stop()
			log.Info().Str("id", m.Id).Msg("收到即时刷新信号，立即刷新。")
			// 继续执行刷新逻辑

		case <-timer.C:
			// 定时器到期，执行刷新逻辑
			// 继续执行刷新逻辑
		}

		// --- 核心刷新执行 ---
		// 注意：此处调用 Refresh 方法，该方法应由 BiliManager 等实现
		if err := m.IManager.Refresh(MaxRetryTimes); err != nil {
			// 刷新失败，日志记录
			log.Err(err).Str("id", m.Id).Msg("自动刷新失败，将在下一轮循环中重试。")
		}
	}
}

const MaxRetryTimes = 10
const RetryWaitDuration = 2 * time.Second

// CommonRefresh 通用 Refresh 函数，负责控制流、重试和状态更新
func CommonRefresh(manager *Manager, strategy RefreshStrategy, retryTimes int, expectExpireTimeInterval time.Duration) error {
	log.Info().Msg("[CommonRefresh] 正在刷新直播流 token...")

	// 边界检查
	if retryTimes < 0 {
		retryTimes = 0
	}
	if retryTimes > MaxRetryTimes {
		retryTimes = MaxRetryTimes
	}

	var err error
	var newStreamUrl string
	var newExpireTime time.Time
	for cnt := 0; cnt <= retryTimes; cnt++ {
		if cnt > 0 {
			time.Sleep(RetryWaitDuration)
			log.Err(err).Msgf("[CommonRefresh] 第%d次重试", cnt)
		}

		// --- 1. 业务逻辑调用（通过策略接口） ---
		streamInfo, fetchErr := strategy.ExecuteFetchStreamInfo()
		if fetchErr != nil {
			err = fetchErr
			log.Err(err).Msg("[CommonRefresh] 刷新直播流信息失败:")
			continue // 重试
		}

		// --- 2. 业务逻辑调用（通过策略接口） ---
		// 尝试解析 Stream URL
		for _, streamUrl := range streamInfo.StreamUrls {
			expireTime, parseErr := strategy.ParseExpiration(streamUrl)
			if parseErr != nil {
				log.Err(parseErr).Msg("[CommonRefresh] 解析 expireTime 失败")
				continue
			}
			newStreamUrl = streamUrl
			newExpireTime = expireTime
			err = nil // 成功解析，清除错误
			break
		}
		break
	}

	// 检查是否所有重试都失败
	if newStreamUrl == "" {
		log.Err(err).Msg("[CommonRefresh] 所有重试均失败，上次错误")
		return err
	}

	// --- 3. 通用状态更新和加锁 ---
	manager.Mutex.Lock()
	manager.CurrentURL = newStreamUrl
	manager.ActualExpireTime = newExpireTime
	manager.SafetyExpireTime = newExpireTime.Add(expectExpireTimeInterval) // 使用传入的参数
	manager.LastRefresh = time.Now()
	manager.Mutex.Unlock()

	log.Info().Msg("[CommonRefresh] 更新成功")

	jsonBytes, _ := json.MarshalIndent(manager, "", "  ")
	log.Info().Msgf("[Refresh] Manager: %s", string(jsonBytes))

	return nil
}
