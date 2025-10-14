package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"video-factory/bilibili"
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

	var rid string

	for roomId == "" {
		fmt.Print("请输入 Bilibili 直播房间号或 URL:\n")
		_, err := fmt.Scanln(&roomId)
		if err != nil {
			log.Println("输入错误，请重新输入。")
			continue
		}
		rid, err = bilibili.CheckAndGetRid(roomId)
		if err != nil {
			log.Println(err)
			roomId = ""
			continue
		} else {
			break
		}
	}

	if cookie == "" {
		fmt.Print("请输入 Bilibili Cookie (可选，不用无法使用720P以上画质):\n")
		fmt.Scanln(&cookie)
	}

	// 设置日志格式 2025/10/14 13:20:45 proxy.go:128: 错误: 执行 HTTP 请求失败: 403 Forbidden
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("正在获取直播流信息: %s", roomId)

	streams, err := bilibili.GetRealURL(rid, cookie)
	if err != nil {
		log.Fatalf("获取真实流地址失败: %v", err)
	}

	for quality, steamUrl := range streams {
		log.Printf("[%s] -> %s", quality, steamUrl)
	}

	// 默认选择第一个流
	var hlsURL string
	for _, u := range streams {
		hlsURL = u
		break
	}

	if hlsURL == "" {
		log.Fatal("未找到可用的 HLS 播放地址。")
	}

	// cfg := ProxyConfig{
	// 	HlsSourceURL: hlsURL,
	// 	Cookie:       cookie,
	// }
	//
	// http.HandleFunc("/", cfg.ProxyHandler)

	parsed, _ := url.Parse(hlsURL)
	log.Printf("代理服务启动: http://localhost:%d", port)
	log.Printf("在 PotPlayer 中打开: http://localhost:%d%s", port, parsed.RequestURI())

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
