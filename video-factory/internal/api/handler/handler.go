package handler

import (
	"video-factory/internal/service"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"
)

type Handler struct {
	RoomHandler   *RoomHandler
	ConfigHandler *ConfigHandler
	StreamHandler *StreamHandler
}

func NewHandler(pool *pool.ManagerPool, config *config.AppConfig, service *service.Service) *Handler {
	return &Handler{
		RoomHandler:   NewRoomHandler(pool, config, service.RoomService),
		ConfigHandler: NewConfigHandler(pool, config, service.ConfigService),
		StreamHandler: NewStreamHandler(pool, config, service.RoomService, service.MonitorService),
	}
}

// type SiteStrategy interface {
// 	GetBaseURLPrefix() string
// 	CreateManager(roomId int64, appConfig *config.AppConfig) (iface.Manager, error)
// 	GetExtraHeaders() http.Header
// }
