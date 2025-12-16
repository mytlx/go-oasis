package repository

import (
	"context"
	"errors"
	"video-factory/internal/domain/model"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type ConfigRepository struct {
	db *gorm.DB
}

func NewConfigRepository(db *gorm.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

func (c *ConfigRepository) AddConfig(config *model.Config) error {
	if config == nil {
		return errors.New("config 为空")
	}

	ctx := context.Background()
	return gorm.G[model.Config](c.db).Create(ctx, config)
}

func (c *ConfigRepository) ListConfigs() ([]model.Config, error) {
	var configs []model.Config
	err := c.db.Find(&configs).Error
	return configs, err
}

func (c *ConfigRepository) ListConfigsMap() (map[string]string, error) {
	configs, err := c.ListConfigs()
	if err != nil {
		return nil, err
	}

	configMap := make(map[string]string)
	for _, cfg := range configs {
		configMap[cfg.Key] = cfg.Value
	}

	return configMap, nil
}

func (c *ConfigRepository) UpdateConfig(config *model.Config) error {
	if config == nil {
		return errors.New("config 为空")
	}

	// 1. 根据主键查找记录 (或者直接使用 Model(config) 设定 WHERE ID = config.ID)
	// 2. 使用 .Updates()
	// 注意：.Updates(config) 会自动忽略 config 中为零值的字段
	result := c.db.Model(&model.Config{}).Where("id = ?", config.ID).Updates(config)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("更新失败，未找到记录或数据未变化")
	}
	return nil
}

// GetConfigByKey 根据key获取配置，没获取到返回空切片非nil
func (c *ConfigRepository) GetConfigByKey(key string) (*model.Config, error) {
	// 声明一个结构体值，而不是指针
	var config model.Config

	// GORM 查找数据，并写入到结构体 config 的内存地址 (&config)
	err := c.db.Where("key = ?", key).First(&config).Error
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (c *ConfigRepository) GetConfigById(id int64) (*model.Config, error) {
	var config model.Config
	err := c.db.Where("id = ?", id).Find(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, err
}

func (c *ConfigRepository) BatchAddConfigs(configs []model.Config) error {
	result := c.db.CreateInBatches(&configs, len(configs))
	affected := result.RowsAffected
	log.Info().Msgf("批量插入 %d 条数据，受影响行数: %d", len(configs), affected)
	return result.Error
}
