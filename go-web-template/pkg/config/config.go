package config

import (
	"github.com/spf13/viper"
	"log"
)

func InitConfig() {
	viper.SetConfigName("config") // 不带扩展名
	viper.SetConfigType("yaml")
	viper.AddConfigPath("pkg/config/")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}
}
