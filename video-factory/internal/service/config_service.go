package service

import (
	"errors"
	"fmt"
	"video-factory/internal/domain/model"
	"video-factory/internal/domain/vo"
	"video-factory/internal/repository"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"
	"video-factory/pkg/util"

	"gorm.io/gorm"
)

type ConfigService struct {
	pool       *pool.ManagerPool
	config     *config.AppConfig
	configRepo *repository.ConfigRepository
}

func NewConfigService(pool *pool.ManagerPool, config *config.AppConfig, configRepo *repository.ConfigRepository) *ConfigService {
	return &ConfigService{
		pool:       pool,
		config:     config,
		configRepo: configRepo,
	}
}

func (c *ConfigService) AddConfig(config *model.Config, pool *pool.ManagerPool) error {
	if config == nil {
		return errors.New("config 为空")
	}
	if config.Key == "" {
		return errors.New("key 为空")
	}
	_, err := c.configRepo.GetConfigByKey(config.Key)
	if err == nil {
		// 如果 err 为 nil，说明记录被成功找到了
		return errors.New("key 已存在，请勿重复添加")
	}

	// 检查是否是“未找到记录”的错误
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果是其他数据库或连接错误，返回该错误
		return fmt.Errorf("查询配置失败: %w", err)
	}

	err = c.configRepo.AddConfig(config)
	if err != nil {
		return err
	}

	// 更新全局配置并通知订阅者
	err = pool.Config.OnUpdate(config.Key, config.Value)
	if err != nil {
		return err
	}
	return nil
}

func (c *ConfigService) ListConfigs() ([]vo.ConfigVO, error) {
	configs, err := c.configRepo.ListConfigs()
	if err != nil {
		return nil, err
	}

	var configVOs []vo.ConfigVO
	for _, cfg := range configs {
		configVOs = append(configVOs, vo.ConfigVO{
			ID:          cfg.ID,
			Key:         cfg.Key,
			Value:       cfg.Value,
			Description: cfg.Description,
			CreateTime:  util.MillisToTime(cfg.CreateTime),
			UpdateTime:  util.MillisToTime(cfg.UpdateTime),
		})
	}

	return configVOs, err
}

func (c *ConfigService) UpdateConfig(updateVo *vo.ConfigUpdateVO, pool *pool.ManagerPool) error {
	if updateVo == nil {
		return errors.New("cfg 为空")
	}
	if updateVo.ID == 0 {
		return errors.New("id 为空")
	}

	cfg, err := c.configRepo.GetConfigById(updateVo.ID)
	if err != nil {
		return err
	}
	if cfg == nil {
		return errors.New("配置不存在")
	}

	updateConfig := &model.Config{
		ID:          updateVo.ID,
		Key:         updateVo.Key,
		Value:       updateVo.Value,
		Description: updateVo.Description,
	}
	err = c.configRepo.UpdateConfig(updateConfig)
	if err != nil {
		return err
	}

	// 更新全局配置并通知订阅者
	err = pool.Config.OnUpdate(updateConfig.Key, updateConfig.Value)
	if err != nil {
		return err
	}

	return nil
}

func (c *ConfigService) ListConfigMap() (map[string]string, error) {
	configs, err := c.configRepo.ListConfigs()
	if err != nil {
		return nil, err
	}

	configMap := make(map[string]string)
	for _, cfg := range configs {
		configMap[cfg.Key] = cfg.Value
	}

	return configMap, nil
}
