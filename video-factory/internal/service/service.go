package service

import (
	"video-factory/internal/repository"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"
)

type Service struct {
	RoomService    *RoomService
	ConfigService  *ConfigService
	MonitorService *MonitorService
}

func NewService(pool *pool.ManagerPool, config *config.AppConfig, repo *repository.Repository) *Service {
	return &Service{
		RoomService:    NewRoomService(pool, config, repo.Room),
		ConfigService:  NewConfigService(pool, config, repo.Config),
		MonitorService: NewMonitorService(pool, config, repo.Room),
	}
}
