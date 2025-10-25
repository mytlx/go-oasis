package service

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"video-factory/internal/domain/model"
	"video-factory/internal/domain/vo"
	"video-factory/internal/repository"
	"video-factory/pkg/pool"
	"video-factory/pkg/util"
)

func AddConfig(config *model.Config, pool *pool.ManagerPool) error {
	if config == nil {
		return errors.New("config 为空")
	}
	if config.Key == "" {
		return errors.New("key 为空")
	}
	_, err := repository.GetConfigByKey(config.Key)
	if err == nil {
		// 如果 err 为 nil，说明记录被成功找到了
		return errors.New("key 已存在，请勿重复添加")
	}

	// 检查是否是“未找到记录”的错误
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果是其他数据库或连接错误，返回该错误
		return fmt.Errorf("查询配置失败: %w", err)
	}

	err = repository.AddConfig(config)
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

func ListConfigs() ([]vo.ConfigVO, error) {
	configs, err := repository.ListConfigs()
	if err != nil {
		return nil, err
	}

	var configVOs []vo.ConfigVO
	for _, config := range configs {
		configVOs = append(configVOs, vo.ConfigVO{
			ID:          config.ID,
			Key:         config.Key,
			Value:       config.Value,
			Description: config.Description,
			CreateTime:  util.MillisToTime(config.CreateTime),
			UpdateTime:  util.MillisToTime(config.UpdateTime),
		})
	}

	return configVOs, err
}

func UpdateConfig(updateVo *vo.ConfigUpdateVO, pool *pool.ManagerPool) error {
	if updateVo == nil {
		return errors.New("config 为空")
	}
	if updateVo.ID == 0 {
		return errors.New("id 为空")
	}

	config, err := repository.GetConfigById(updateVo.ID)
	if err != nil {
		return err
	}
	if config == nil {
		return errors.New("配置不存在")
	}

	updateConfig := &model.Config{
		ID:          updateVo.ID,
		Key:         updateVo.Key,
		Value:       updateVo.Value,
		Description: updateVo.Description,
	}
	err = repository.UpdateConfig(updateConfig)
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

func ListConfigMap() (map[string]string, error) {
	configs, err := repository.ListConfigs()
	if err != nil {
		return nil, err
	}

	configMap := make(map[string]string)
	for _, config := range configs {
		configMap[config.Key] = config.Value
	}

	return configMap, nil
}
