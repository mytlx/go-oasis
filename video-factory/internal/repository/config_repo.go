package repository

import (
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"video-factory/internal/db"
	"video-factory/internal/domain/model"
)

func AddConfig(config *model.Config) error {
	if config == nil {
		return errors.New("config 为空")
	}

	ctx := context.Background()
	return gorm.G[model.Config](db.DB).Create(ctx, config)
}

func ListConfigs() ([]model.Config, error) {
	var configs []model.Config
	err := db.DB.Find(&configs).Error
	return configs, err
}

func UpdateConfig(config *model.Config) error {
	if config == nil {
		return errors.New("config 为空")
	}

	// 1. 根据主键查找记录 (或者直接使用 Model(config) 设定 WHERE ID = config.ID)
	// 2. 使用 .Updates()
	// 注意：.Updates(config) 会自动忽略 config 中为零值的字段
	result := db.DB.Model(&model.Config{}).Where("id = ?", config.ID).Updates(config)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("更新失败，未找到记录或数据未变化")
	}
	return nil
}

// GetConfigByKey 根据key获取配置，没获取到返回空切片非nil
func GetConfigByKey(key string) (*model.Config, error) {
	// 声明一个结构体值，而不是指针
	var config model.Config

	// GORM 查找数据，并写入到结构体 config 的内存地址 (&config)
	err := db.DB.Where("key = ?", key).First(&config).Error
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func GetConfigById(id int64) (*model.Config, error) {
	var config model.Config
	err := db.DB.Where("id = ?", id).Find(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, err
}

func BatchAddConfigs(configs []model.Config) error {
	result := db.DB.CreateInBatches(&configs, len(configs))
	affected := result.RowsAffected
	log.Info().Msgf("批量插入 %d 条数据，受影响行数: %d", len(configs), affected)
	return result.Error
}
