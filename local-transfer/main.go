package main

import (
	"encoding/json"
	"local-transfer/lan"
	"local-transfer/middleware"
	"local-transfer/server"
	"local-transfer/utils"
	"log"
	"net/http"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/upload", server.UploadHandler)
	mux.HandleFunc("/download/", server.DownloadHandler)
	mux.HandleFunc("/files", server.ListFilesHandler) // 返回 JSON 文件列表

	mux.HandleFunc("/ws", server.WsHandler)

	mux.HandleFunc("/messages", server.LoadMessagesHandler)

	mux.HandleFunc("/ip", server.GetClientIPHandler)

	mux.HandleFunc("/api/devices/online", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(lan.GetOnlineDevices())
	})
	mux.HandleFunc("/api/devices/discovered", server.DiscoveredHandler)
	// 提供静态资源（网页）
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	// 包一层 CORS 中间件
	handlerWithCORS := middleware.EnableCORS(mux)

	// 初始化 ID 生成器
	err := utils.Init(1)
	if err != nil {
		log.Fatalf("初始化 ID 生成器失败: %v", err)
	}

	log.Println("服务启动，访问地址：http://localhost:8080")
	if err := http.ListenAndServe(":8080", handlerWithCORS); err != nil {
		log.Fatal("服务启动失败:", err)
	}

}
