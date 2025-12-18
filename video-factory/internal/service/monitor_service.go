package service

import (
	"context"
	"errors"
	"time"
	"video-factory/internal/common/consts"
	"video-factory/internal/domain/model"
	"video-factory/internal/manager"
	"video-factory/internal/repository"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"

	"github.com/rs/zerolog/log"
)

type MonitorService struct {
	pool     *pool.ManagerPool
	config   *config.AppConfig
	roomRepo *repository.RoomRepository
}

func NewMonitorService(pool *pool.ManagerPool, cfg *config.AppConfig, roomRepo *repository.RoomRepository) *MonitorService {
	return &MonitorService{
		pool:     pool,
		config:   cfg,
		roomRepo: roomRepo,
	}
}

func (m *MonitorService) StartMonitorLoop(ctx context.Context) {
	log.Info().Msgf("全局监控服务启动")
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// 首次启动时，立即扫描并开启直播流
	m.scanAndStartRooms(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info().Msgf("[Monitor] 全局监控服务已停止")
			return
		case <-ticker.C:
			// 扫描并开启直播流
			log.Info().Msgf("[Monitor] 刷新间隔到期，扫描并开启直播流")
			m.scanAndStartRooms(ctx)
		}
	}
}

func (m *MonitorService) scanAndStartRooms(ctx context.Context) {
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
			if err := m.StartManager(ctx, room.ID, room.Platform); err != nil {
				log.Err(err).Int64("roomId", room.ID).Msg("启动 Manager 失败")
			}
		}
	}
}

func (m *MonitorService) StartManager(ctx context.Context, roomId int64, platform string) error {
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

	// 启动自动刷新
	go mgr.StartAutoRefresh(ctx)

	// tlxTODO: 录制功能也在此启动

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
