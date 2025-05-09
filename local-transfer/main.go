package main

import (
	"local-transfer/server"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/upload", server.UploadHandler)
	http.HandleFunc("/download/", server.DownloadHandler)
	http.HandleFunc("/files", server.ListFilesHandler) // 返回 JSON 文件列表

	// 提供静态资源（网页）
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/ws", server.WsHandler)

	http.HandleFunc("/messages", server.LoadMessagesHandler)

	log.Println("服务启动，访问地址：http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("服务启动失败:", err)
	}

}
