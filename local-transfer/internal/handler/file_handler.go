package handler

import (
	"github.com/gin-gonic/gin"
	"io"
	"local-transfer/internal/config"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func UploadFile(c *gin.Context) {

	if c.Request.Method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "仅支持 POST 请求"})
		return
	}

	deviceId := c.PostForm("deviceId")
	if deviceId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 deviceId"})
		return
	}

	// DeviceMu.RLock()
	// client, ok := DeviceMap[deviceId]
	// DeviceMu.RUnlock()
	// if !ok || client == nil {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "未知设备"})
	// 	return
	// }

	fileHandler, err := c.FormFile("file")
	if err != nil {
		log.Println("文件读取失败:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件读取失败"})
		return
	}

	log.Printf("收到来自设备 %s 的文件: %s (%d bytes)", deviceId, fileHandler.Filename, fileHandler.Size)

	file, err := fileHandler.Open()
	if err != nil {
		log.Println("无法打开文件:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开文件"})
		return
	}
	defer file.Close()

	// 创建目录和文件
	os.MkdirAll(config.UploadPath, os.ModePerm)
	fullPath := filepath.Join(config.UploadPath, fileHandler.Filename)
	log.Println("fullPath:", fullPath)
	// TODO: 判断文件名称是否已经存在，如果存在，判断文件是否一致，如果一致，跳过，如果不一致，重命名
	dst, err := os.Create(fullPath)
	if err != nil {
		log.Println("保存失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	defer dst.Close()

	// 写入文件数据
	_, err = io.Copy(dst, file)
	if err != nil {
		log.Println("写入失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败"})
		return
	}
	// fmt.Fprintf(w, "上传成功: %s\n", handler.Filename)
	//
	// mimeType := mime.TypeByExtension(filepath.Ext(handler.Filename))
	// isImage := mimeType != "" && (mimeType[:6] == "image/")
	// msgType := "file"
	// if isImage {
	// 	msgType = "image"
	// }
	// BroadcastMsg(client, handler.Filename, msgType)

	c.JSON(http.StatusOK, gin.H{"message": "上传成功", "path": fileHandler.Filename})
}

func DownloadFile(c *gin.Context) {
	filename := c.Param("filename")
	fullPath := filepath.Join(config.UploadPath, filename)

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.String(http.StatusNotFound, "文件不存在")
		return
	}

	c.File(fullPath)
}
