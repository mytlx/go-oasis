package db

import (
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"video-factory/internal/domain/model"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("./db/video-factory.db"), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("[InitDB] 数据库连接失败")
	}
	log.Info().Msg("[InitDB] 数据库连接成功！")

	// 自动迁移表结构
	if err := DB.AutoMigrate(&model.Room{}); err != nil {
		log.Fatal().Err(err).Msg("[InitDB] 表[t_room]迁移失败")
	}
	if err := DB.AutoMigrate(&model.Config{}); err != nil {
		log.Fatal().Err(err).Msg("[InitDB] 表[t_config]迁移失败")
	}
	log.Info().Msg("[InitDB] 数据库存在或已迁移成功！")

	err = initConfigData()
	if err != nil {
		log.Fatal().Err(err).Msg("[InitConfig] 初始化config数据失败")
	}

	log.Info().Msg("[InitDB] 数据库初始化完成！")
}

func initConfigData() error {
	// 检查是否有数据，如果没有，则批量插入
	var count int64
	DB.Model(&model.Config{}).Count(&count)

	if count == 0 {
		log.Info().Msg("[InitConfig] 初始化config数据..")
		result := DB.CreateInBatches(&InitialConfigs, len(InitialConfigs))
		affected := result.RowsAffected
		log.Info().Msgf("[InitConfig] 批量插入 %d 条数据，受影响行数: %d", len(InitialConfigs), affected)
		return result.Error
	}
	return nil
}

var InitialConfigs = []model.Config{
	{
		ID:          1,
		Key:         "port",
		Value:       "8090",
		Description: "程序端口，下次启动生效，优先级在命令行和配置文件之后",
	},
	{
		ID:          2,
		Key:         "proxy.enabled",
		Value:       "false",
		Description: "是否使用代理",
	},
	{
		ID:          3,
		Key:         "proxy.system_proxy",
		Value:       "false",
		Description: "是否使用系统代理，仅在enabled=true时生效",
	},
	{
		ID:          4,
		Key:         "proxy.host",
		Value:       "127.0.0.1",
		Description: "代理服务器地址，仅在enabled=true且systemProxy=false时生效",
	},
	{
		ID:          5,
		Key:         "proxy.port",
		Value:       "7890",
		Description: "代理服务器端口，仅在enabled=true且systemProxy=false时生效",
	},
	{
		ID:          6,
		Key:         "proxy.username",
		Value:       "",
		Description: "代理服务器验证用户名，仅在enabled=true且systemProxy=false时生效",
	},
	{
		ID:          7,
		Key:         "proxy.password",
		Value:       "",
		Description: "代理服务器验证密码，仅在enabled=true且systemProxy=false时生效",
	},
	{
		ID:          8,
		Key:         "proxy.protocol",
		Value:       "http",
		Description: "代理协议，仅在enabled=true时生效",
	},
	{
		ID:          9,
		Key:         "bili.cookie",
		Value:       "",
		Description: "b站cookie，有些直播间需要cookie才能看高清晰度",
	},
	{
		ID:          10,
		Key:         "missevan.cookie",
		Value:       "",
		Description: "猫耳的cookie，没有也行",
	},
}
