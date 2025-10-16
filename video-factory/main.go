package main

import (
	"flag"
	"fmt"
	"github.com/rs/zerolog/log"
	"video-factory/logger"
	"video-factory/pool"
	"video-factory/router"
)

func main() {
	var (
		roomId string
		cookie string
		port   int
	)

	flag.StringVar(&roomId, "room", "", "Bilibili房间号 或 URL")
	flag.StringVar(&cookie, "cookie", "", "Bilibili Cookie (可选，不用无法使用720P以上画质)")
	flag.IntVar(&port, "port", 8090, "本地监听端口，默认8090")
	flag.Parse()

	// 设置日志格式
	logger.InitLogger()

	// 初始化 ManagerPool
	p := pool.NewManagerPool()

	// 通过 NewEngine 创建配置好的 Gin 引擎，并将 Pool 注入
	routerEngine := router.NewEngine(p)

	log.Info().Msgf("代理服务启动: http://localhost:%d", port)

	// 启动 Gin 服务器
	log.Fatal().Err(routerEngine.Run(fmt.Sprintf(":%d", port)))
}
