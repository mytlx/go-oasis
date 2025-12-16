package handler

import (
	"fmt"
	"video-factory/internal/api/response"
	"video-factory/internal/domain/model"
	"video-factory/internal/domain/vo"
	"video-factory/internal/service"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type ConfigHandler struct {
	configService *service.ConfigService
	pool          *pool.ManagerPool
	config        *config.AppConfig
}

func NewConfigHandler(pool *pool.ManagerPool, config *config.AppConfig, configService *service.ConfigService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
		pool:          pool,
		config:        config,
	}
}

func (ch *ConfigHandler) ConfigAddHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req vo.ConfigAddVO
		err := c.ShouldBindJSON(&req)
		if err != nil {
			response.Error(c, err.Error())
			return
		}
		err = ch.configService.AddConfig(&model.Config{
			Key:         req.Key,
			Value:       req.Value,
			Description: req.Description,
		}, ch.pool)
		if err != nil {
			log.Err(err).Msg("添加配置失败")
			response.Error(c, fmt.Sprintf("添加配置失败: %v", err))
			return
		}
		response.OkWithMsg(c, "添加配置成功")
	}
}

func (ch *ConfigHandler) ConfigUpdateHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req vo.ConfigUpdateVO
		err := c.ShouldBindJSON(&req)
		if err != nil {
			response.Error(c, err.Error())
			return
		}
		err = ch.configService.UpdateConfig(&req, ch.pool)
		if err != nil {
			log.Err(err).Msg("更新配置失败")
			response.Error(c, fmt.Sprintf("更新配置失败: %v", err))
			return
		}
		response.OkWithMsg(c, "更新配置成功")
	}
}

func (ch *ConfigHandler) ConfigListHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		configs, err := ch.configService.ListConfigs()
		if err != nil {
			log.Err(err).Msg("获取配置列表失败")
			response.Error(c, fmt.Sprintf("获取配置列表失败: %v", err))
			return
		}
		response.OkWithList(c, configs, int64(len(configs)), 0, 0)
	}
}
