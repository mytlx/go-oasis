package main

import (
	"github.com/rs/zerolog/log"
	"video-factory/cmd"
	"video-factory/logger"
)

func main() {
	// 1. 设置日志格式/系统
	logger.InitLogger()

	// 2. 启动 CLI 应用和配置加载 (核心逻辑)
	if err := cmd.Execute(); err != nil {
		// 所有的配置加载、CLI 解析错误都在这里捕获
		log.Fatal().Err(err)
	}
}
