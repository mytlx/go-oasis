package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"video-factory/bilibili"
	"video-factory/proxy"
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

	pool := bilibili.NewManagerPool()

	http.HandleFunc("POST /bilibili/room", proxy.BiliRoomAddHandler(pool))
	http.HandleFunc("DELETE /bilibili/room", proxy.BiliRoomRemoveHandler(pool))
	http.HandleFunc("GET /bilibili/room", proxy.BiliRoomDetailHandler(pool))

	http.HandleFunc("/bilibili/{managerId}/{file...}", proxy.BiliHandler(pool))

	log.Printf("代理服务启动: http://localhost:%d", port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
