package handler

import (
	"video-factory/internal/api/response"
	"video-factory/internal/service"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"

	"github.com/gin-gonic/gin"
)

type MonitorHandler struct {
	pool           *pool.ManagerPool
	config         *config.AppConfig
	monitorService *service.MonitorService
}

func NewMonitorHandler(pool *pool.ManagerPool, config *config.AppConfig, monitorService *service.MonitorService) *MonitorHandler {
	return &MonitorHandler{
		pool:           pool,
		config:         config,
		monitorService: monitorService,
	}
}

func (m *MonitorHandler) Start(c *gin.Context) {
	if err := m.monitorService.Start(nil); err != nil {
		response.Error(c, "启动失败")
		return
	}
	response.Ok(c)
}

func (m *MonitorHandler) Stop(c *gin.Context) {
	m.monitorService.StopMonitor()
	response.Ok(c)
}

func (m *MonitorHandler) Restart(c *gin.Context) {
	if err := m.monitorService.RestartMonitor(); err != nil {
		response.Error(c, "重启失败")
		return
	}
	response.Ok(c)
}

func (m *MonitorHandler) Refresh(c *gin.Context) {
	m.monitorService.TriggerRefresh()
	response.Ok(c)
}
