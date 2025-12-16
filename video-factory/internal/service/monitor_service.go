package service

import (
	"context"
	"errors"
	"fmt"
	"video-factory/internal/iface"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"

	"github.com/rs/zerolog/log"
)

type MonitorService struct {
	pool        *pool.ManagerPool
	config      *config.AppConfig
	roomService *RoomService
}

func NewMonitorService(pool *pool.ManagerPool, cfg *config.AppConfig, roomService *RoomService) *MonitorService {
	return &MonitorService{
		pool:        pool,
		config:      cfg,
		roomService: roomService,
	}
}

func (m *MonitorService) StartManager(roomId int64, platform string) error {
	if _, exist := m.pool.Get(roomId); exist {
		return errors.New("已处于运行中状态")
	}

	room, err := m.roomService.GetRoom(roomId)
	if err != nil {
		return err
	}
	if room == nil {
		return errors.New("房间不存在，请先添加房间")
	}
	if room.Status == 0 {
		return errors.New("房间未启用，请先启用房间")
	}

	var mgr iface.Manager
	switch platform {
	case "bili":
		if mgr, err = bili.NewManager(room, m.config); err != nil {
			return err
		}
	case "missevan":
		if mgr, err = missevan.NewManager(room, m.config); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的平台：%s", platform)
	}

	// 启动 manager
	if err = mgr.Start(context.Background()); err != nil {
		return err
	}

	// 添加到 pool 中
	m.pool.Add(roomId, mgr)
	log.Info().Int64("roomId", roomId).Msg("Manager 新建成功并加入 pool")
	return nil
}
