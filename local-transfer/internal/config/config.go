package config

import (
	"github.com/spf13/viper"
	"log"
)

var (
	UploadPath string
)

func InitConfig() {
	viper.SetConfigName("config") // 不带扩展名
	viper.SetConfigType("yaml")
	viper.AddConfigPath("internal/config/")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	UploadPath = viper.GetString("upload.path")
	log.Println("UploadPath:", UploadPath)
}
