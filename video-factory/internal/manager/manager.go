package manager

import (
	"context"
	"errors"
	"sync"
	"time"
	"video-factory/internal/iface"

	"github.com/avast/retry-go/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Manager struct {
	Id               int64
	Streamer         iface.Streamer `json:"-"`
	CurrentURL       string
	ProxyURL         string
	ActualExpireTime time.Time
	SafetyExpireTime time.Time
	LastRefreshTime  time.Time
	IManager         iface.Manager      `json:"-"` // 持有接口，可以使用外部逻辑
	refreshCancel    context.CancelFunc // 用于触发停止信号
	refreshCh        chan struct{}      // 用于通知 AutoRefresh 循环立即执行一次刷新（如首次启动或外部命令）
	Mutex            sync.RWMutex       `json:"-"`
}

// func (m *Manager) AddOrUpdateRoom() error {
// 	m.Mutex.RLock()
// 	info := m.Streamer.GetInfo()
// 	m.Mutex.RUnlock()
//
// 	room := &model.Room{
// 		ID:         info.Rid,
// 		Platform:   info.Platform,
// 		RealID:     info.RealRoomId,
// 		Name:       "",
// 		URL:        info.RoomUrl,
// 		ProxyURL:   m.ProxyURL,
// 		CreateTime: time.Now().UnixMilli(),
// 		UpdateTime: time.Now().UnixMilli(),
// 	}
//
// 	return repository.AddOrUpdateRoom(room)
// }

// StartAutoRefresh 启动一个 Goroutine，根据过期时间自动刷新 Manager 状态
// interval 是一个安全提前量，例如提前 5 秒或 5 分钟
func (m *Manager) StartAutoRefresh(interval time.Duration) {
	// 确保只启动一次
	if m.refreshCancel != nil {
		log.Warn().Int64("id", m.Id).Msg("自动刷新服务已在运行。")
		return
	}

	// 初始化 Context 用于控制 Goroutine 生命周期
	ctx, cancel := context.WithCancel(context.Background())
	m.refreshCancel = cancel
	m.refreshCh = make(chan struct{}, 1) // 有缓冲，防止发送阻塞

	log.Info().Int64("id", m.Id).Msg("[AutoRefresh] 启动自动刷新服务")

	// 启动 Goroutine
	go m.autoRefreshLoop(ctx, interval)
}

// StopAutoRefresh 发送停止信号给自动刷新 Goroutine
func (m *Manager) StopAutoRefresh() {
	if m.refreshCancel != nil {
		m.refreshCancel() // 调用 context 的 CancelFunc 触发停止
		m.refreshCancel = nil
		// refreshCh 在 autoRefreshLoop 退出后应该被关闭，这里不必显式关闭
	}
}

// TriggerRefresh 发送信号给循环，使其立即执行一次刷新
func (m *Manager) TriggerRefresh() {
	select {
	case m.refreshCh <- struct{}{}:
		log.Info().Int64("id", m.Id).Msg("手动触发即时刷新信号。")
	default:
		// 如果通道已满，说明循环正在忙或等待，忽略本次触发
		log.Debug().Int64("id", m.Id).Msg("即时刷新信号发送失败，循环正忙。")
	}
}

func (m *Manager) GetId() int64 {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()
	return m.Id
}

func (m *Manager) GetCurrentURL() string {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()
	return m.CurrentURL
}

func (m *Manager) GetProxyURL() string {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()
	return m.ProxyURL
}

func (m *Manager) GetLastRefreshTime() time.Time {
	m.Mutex.RLock()
	defer m.Mutex.RUnlock()
	return m.LastRefreshTime
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
		Time("last_refresh_time", m.LastRefreshTime)
}

// autoRefreshLoop 是 AutoRefresh 的核心循环
func (m *Manager) autoRefreshLoop(ctx context.Context, refreshSafetyInterval time.Duration) {
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
			waitTime = 3 * time.Second
			log.Warn().Int64("id", m.Id).Msgf("[AutoRefresh] 链接已过期或即将过期，立即等待 %s 后重试。", waitTime)
		}

		timer := time.NewTimer(waitTime)

		select {
		case <-ctx.Done():
			// 收到停止信号，退出循环
			timer.Stop()
			log.Info().Int64("id", m.Id).Msg("[AutoRefresh] 自动刷新服务已优雅停止。")
			return // 退出 Goroutine

		case <-m.refreshCh:
			// 收到立即刷新信号（手动触发或首次启动）
			timer.Stop()
			log.Info().Int64("id", m.Id).Msg("[AutoRefresh] 收到即时刷新信号，立即刷新。")
			// 继续执行刷新逻辑

		case <-timer.C:
			log.Info().Int64("id", m.Id).Msg("[AutoRefresh] 刷新间隔到期，开始刷新。")
			// 定时器到期，执行刷新逻辑
			// 继续执行刷新逻辑
		}

		// --- 核心刷新执行 ---
		// 注意：此处调用 Refresh 方法，该方法应由 BiliManager 等实现
		if err := m.IManager.Refresh(ctx, MaxAttemptTimes); err != nil {
			// 刷新失败，日志记录
			log.Err(err).Int64("id", m.Id).Msg("[AutoRefresh] 自动刷新失败，将在下一轮循环中重试。")
		}
	}
}

const MaxAttemptTimes = 10
const RetryWaitDuration = 2 * time.Second

// CommonRefresh 通用 Refresh 函数，负责控制流、重试和状态更新
func CommonRefresh(ctx context.Context, manager *Manager, strategy iface.RefreshStrategy,
	attempts int, expectExpireTimeInterval time.Duration, certainQnFlag bool) error {
	log.Info().Msg("[CommonRefresh] 正在刷新直播流 token...")

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
				log.Err(err).Msgf("[CommonRefresh] 第%d次重试 start", n)
			},
		),
		retry.Context(ctx),
	)
	err := r.Do(func() error {
		// --- 1. 业务逻辑调用（通过策略接口） ---
		streamInfo, fetchErr := strategy.ExecuteFetchStreamInfo(certainQnFlag)
		if fetchErr != nil {
			log.Err(fetchErr).Msg("[CommonRefresh] 刷新直播流信息失败:")
			return fetchErr
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
			return nil
		}
		return errors.New("[CommonRefresh] 解析 expireTime 失败")
	})

	// 检查是否所有重试都失败
	if newStreamUrl == "" || err != nil {
		log.Err(err).Msg("[CommonRefresh] 所有重试均失败，上次错误")
		return err
	}

	// --- 3. 通用状态更新和加锁 ---
	manager.Mutex.Lock()
	manager.CurrentURL = newStreamUrl
	manager.ActualExpireTime = newExpireTime
	manager.SafetyExpireTime = newExpireTime.Add(-expectExpireTimeInterval)
	manager.LastRefreshTime = time.Now()
	manager.Mutex.Unlock()

	log.Info().Msg("[CommonRefresh] 更新成功")
	log.Info().Object("manager", manager).Msg("[CommonRefresh] Manager")

	return nil
}
