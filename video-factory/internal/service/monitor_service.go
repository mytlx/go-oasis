package service

import (
	"context"
	"errors"
	"sync"
	"time"
	"video-factory/internal/common/consts"
	"video-factory/internal/domain/model"
	"video-factory/internal/domain/vo"
	"video-factory/internal/manager"
	"video-factory/internal/repository"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"
	"video-factory/pkg/util"

	"github.com/rs/zerolog/log"
)

type MonitorService struct {
	pool     *pool.ManagerPool
	config   *config.AppConfig
	roomRepo *repository.RoomRepository

	// 控制相关
	refreshCh chan struct{}

	// 上下文控制
	rootCtx context.Context    // 父级 Context
	ctx     context.Context    // 当前 Monitor 运行时的 Context (由 rootCtx 衍生)
	cancel  context.CancelFunc // 用于停止当前运行的 Monitor

	// 状态与锁
	isRunning bool
	mu        sync.Mutex
}

func NewMonitorService(pool *pool.ManagerPool, cfg *config.AppConfig, roomRepo *repository.RoomRepository) *MonitorService {
	return &MonitorService{
		pool:      pool,
		config:    cfg,
		roomRepo:  roomRepo,
		refreshCh: make(chan struct{}, 1),
	}
}

func (m *MonitorService) Start(parentCtx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		log.Warn().Msgf("[Monitor] 已经在运行中，不要重复开启")
		return nil
	}

	if parentCtx == nil && m.rootCtx == nil {
		log.Error().Msgf("[Monitor] 无法开启，未提供根 context")
		return errors.New("[Monitor] 无法开启，未提供根 context")
	}

	m.rootCtx = parentCtx
	m.ctx, m.cancel = context.WithCancel(m.rootCtx)
	m.isRunning = true

	log.Info().Msgf("=============== [Monitor] 全局监控服务启动 ===============")

	// 启动循环
	go m.monitorLoop()

	return nil
}

func (m *MonitorService) StopMonitor() {
	if m.cancel != nil {
		m.cancel()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		log.Warn().Msgf("[Monitor] 已经停止，不要重复操作")
		return
	}

	log.Info().Msg("[Monitor] 正在停止全局监控服务...")

	// 触发取消，这会让 loop 退出，也会让所有 Manager 退出
	if m.cancel != nil {
		m.cancel()
	}

	m.isRunning = false
	log.Info().Msg("=============== [Monitor] 全局监控服务已停止 ===============")
}

func (m *MonitorService) RestartMonitor() error {
	log.Info().Msg("[Monitor] 正在重启监控服务...")

	// 1. 先停止
	m.StopMonitor()

	// 2. 等待一小会儿让资源释放（可选，主要看 Manager 清理速度）
	time.Sleep(100 * time.Millisecond)

	// 3. 重新开启 (使用保存的 rootCtx)
	if m.rootCtx == nil {
		return errors.New("[Monitor] 无法重启：未保存 rootContext")
	}

	// Start 内部有锁，所以这里直接调是安全的
	m.Start(m.rootCtx)

	return nil
}

// TriggerRefresh 发送信号给循环，使其立即执行一次刷新
func (m *MonitorService) TriggerRefresh() {
	select {
	case m.refreshCh <- struct{}{}:
		log.Info().Msg("[Monitor] 手动触发即时刷新信号")
	case <-m.ctx.Done():
		log.Warn().Msg("[Monitor] 已停止，忽略刷新请求")
	default:
		// 如果通道已满，说明循环正在忙或等待，忽略本次触发
		log.Warn().Msg("[Monitor] 即时刷新信号发送失败，循环正忙。")
	}
}

func (m *MonitorService) monitorLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// 首次启动时，立即扫描并开启直播流
	m.scanAndStartRooms()

	for {
		select {
		case <-m.ctx.Done():
			log.Info().Msgf("[Monitor] 全局监控服务已停止")
			return
		case <-m.refreshCh:
			log.Info().Msg("[Monitor] 收到即时刷新信号，立即刷新")
			m.scanAndStartRooms()
		case <-ticker.C:
			// 扫描并开启直播流
			log.Info().Msgf("[Monitor] -------- 定时扫描开始 --------")
			m.scanAndStartRooms()
			log.Info().Msgf("[Monitor] ---------- 扫描完成 ----------")
		}
	}
}

func (m *MonitorService) scanAndStartRooms() {
	rooms, err := m.roomRepo.GetEnabledRooms()
	if err != nil {
		log.Err(err).Msg("获取启用房间失败")
		return
	}

	for _, room := range rooms {
		// 已经在 pool 中，直接跳过
		if _, exist := m.pool.Get(room.ID); exist {
			continue
		}

		// 检查房间是否正在直播
		if m.checkRoomLiveStatus(&room) {
			log.Info().Str("anchor", room.AnchorName).Msg("监测到房间开播，正在启动 Manager")
			if err := m.StartManager(room.ID); err != nil {
				log.Err(err).Int64("roomId", room.ID).Msg("启动 Manager 失败")
			}
		}
	}
}

func (m *MonitorService) StartManager(roomId int64) error {
	if roomId == 0 {
		return errors.New("roomId 为空")
	}
	if _, exist := m.pool.Get(roomId); exist {
		return errors.New("已处于运行中状态")
	}

	room, err := m.roomRepo.GetRoomById(roomId)
	if err != nil {
		return err
	}
	if room == nil {
		return errors.New("房间不存在，请先添加房间")
	}
	if room.Status == 0 {
		return errors.New("房间未启用，请先启用房间")
	}

	// 定义回调：Manager 停止时从池中移除
	onStop := func(id int64) {
		log.Info().Int64("id", id).Msg("Manager 已停止，从 Pool 中移除")
		m.pool.Remove(id)
	}
	mgr, err := manager.NewManager(room, m.config, onStop)
	if err != nil {
		return err
	}

	// 添加到 pool 中
	m.pool.Add(roomId, mgr)
	log.Info().Int64("roomId", roomId).Msg("Manager 新建成功并加入 pool")

	// 启动自动刷新，录制功能在 manager 中启动
	go mgr.StartAutoRefresh(m.ctx)

	return nil
}

func (m *MonitorService) checkRoomLiveStatus(room *model.Room) bool {
	if room == nil {
		return false
	}
	switch room.Platform {
	case consts.PlatformBili:
		status, err := bili.GetRoomLiveStatus(room.RealID)
		return err == nil && status == 1
	case consts.PlatformMissevan:
		status, err := missevan.GetRoomLiveStatus(room.RealID)
		return err == nil && status == 1
	default:
		return false
	}
}

func (m *MonitorService) GetManagerList() ([]vo.ManagerVO, error) {
	rooms, err := m.roomRepo.GetEnabledRooms()
	if err != nil {
		log.Err(err).Msg("获取启用房间失败")
		return nil, err
	}

	poolSnapshot := m.pool.Snapshot()

	respList := make([]vo.ManagerVO, len(rooms))
	for i, room := range rooms {
		managerVo := &vo.ManagerVO{
			RoomID:       room.ID,
			RealID:       room.RealID,
			Platform:     room.Platform,
			Name:         room.Name,
			CoverURL:     room.CoverURL,
			AnchorName:   room.AnchorName,
			AnchorID:     room.AnchorID,
			AnchorAvatar: room.AnchorAvatar,
			URL:          room.URL,
			ProxyURL:     room.ProxyURL,
		}

		if managerPtr, ok := poolSnapshot[room.ID]; ok {
			managerVo.LiveStatus = 1
			managerVo.CurrentURL = managerPtr.CurrentURL
			managerVo.LastRefresh = &managerPtr.LastRefreshTime
			managerVo.ExpireTime = &managerPtr.ActualExpireTime
			managerVo.RecordStatus = managerPtr.RecordStatus
			if managerPtr.RecordStatus == 1 {
				managerVo.RecordFile = managerPtr.Recorder.File.Name()
				managerVo.RecordSize = managerPtr.Recorder.Filesize
				managerVo.RecordSizeStr = util.FormatFilesize(managerPtr.Recorder.Filesize)
				managerVo.RecordDuration = managerPtr.Recorder.Duration
				managerVo.RecordDurationStr = util.FormatDuration(managerPtr.Recorder.Duration)
			}
		}
		respList[i] = *managerVo
	}

	return respList, nil
}
