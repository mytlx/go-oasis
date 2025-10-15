package main

import (
	"flag"
	"fmt"
	"log"
	"video-factory/bilibili"
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

	// 设置日志格式 2025/10/14 13:20:45 proxy.go:128: 错误: 执行 HTTP 请求失败: 403 Forbidden
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 初始化 ManagerPool
	pool := bilibili.NewManagerPool()

	// 通过 NewEngine 创建配置好的 Gin 引擎，并将 Pool 注入
	routerEngine := router.NewEngine(pool)

	log.Printf("代理服务启动: http://localhost:%d", port)

	// 启动 Gin 服务器
	log.Fatal(routerEngine.Run(fmt.Sprintf(":%d", port)))
}
