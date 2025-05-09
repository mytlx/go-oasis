package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const UploadDir = "./storage"

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "仅支持 POST 方法", http.StatusMethodNotAllowed)
		return
	}

	deviceId := r.FormValue("device_id")
	if deviceId == "" {
		http.Error(w, "缺少 device_id", http.StatusBadRequest)
		return
	}
	DeviceMu.RLock()
	client, ok := DeviceMap[deviceId]
	DeviceMu.RUnlock()
	if !ok {
		http.Error(w, "未知设备", http.StatusUnauthorized)
		return
	}

	// 获取表单中的文件
	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Println(err)
		http.Error(w, "文件读取失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 创建目录和文件
	os.MkdirAll(UploadDir, os.ModePerm)
	dstPath := filepath.Join(UploadDir, handler.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "保存失败", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 写入文件数据
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "写入失败", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "上传成功: %s\n", handler.Filename)

	mimeType := mime.TypeByExtension(filepath.Ext(handler.Filename))
	isImage := mimeType != "" && (mimeType[:6] == "image/")

	if isImage {
		BroadcastMsg(client, handler.Filename, "image")
		// 通知图片类型消息
		// NotifyAll(`img::` + handler.Filename)
		//
		// SaveMsg(Message{
		// 	Time:    time.Now().Format(time.RFC3339),
		// 	Type:    "image",
		// 	Content: handler.Filename,
		// })
	} else {
		BroadcastMsg(client, handler.Filename, "file")

		// 通知所有客户端
		// NotifyAll("new_file_uploaded")
		// NotifyAll(`file::` + handler.Filename)

		// SaveMsg(Message{
		// 	Time:    time.Now().Format(time.RFC3339),
		// 	Type:    "file",
		// 	Content: handler.Filename,
		// })
	}
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/download/")
	fullPath := filepath.Join(UploadDir, filename)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func ListFilesHandler(w http.ResponseWriter, r *http.Request) {
	files, err := os.ReadDir(UploadDir)
	if err != nil {
		http.Error(w, "读取文件失败", http.StatusInternalServerError)
		return
	}

	var names []string
	for _, f := range files {
		if !f.IsDir() {
			names = append(names, f.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(names)
}

func LoadMessagesHandler(w http.ResponseWriter, r *http.Request) {
	messages, _ := LoadMessages()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(messages)
}
