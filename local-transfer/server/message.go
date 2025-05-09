package server

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Source struct {
	DeviceId   string `json:"device_id"`
	IP         string `json:"ip"`
	DeviceName string `json:"device_name"`
	DeviceType string `json:"device_type"`
}

type Message struct {
	ID      int64  `json:"id"`      // 消息ID
	Time    string `json:"time"`    // ISO 时间
	Type    string `json:"type"`    // text | image | file
	Source  Source `json:"source"`  // 消息来源
	Content string `json:"content"` // 文本内容 或 图片文件名
}

var (
	msgMutex sync.Mutex
	msgPath  = "data/message.txt"
	msgDir   = filepath.Dir(msgPath)
)

func SaveMsg(msg Message) {
	msgMutex.Lock()
	defer msgMutex.Unlock()

	// 确保目录存在
	if err := os.MkdirAll(msgDir, 0755); err != nil {
		log.Println("创建目录失败:", err)
		return
	}

	// 打开文件并追加写入
	file, err := os.OpenFile(msgPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("打开文件失败:", err)
		return
	}
	defer file.Close()

	// 序列化并写入一行
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("序列化失败:", err)
		return
	}

	_, _ = file.Write(append(data, '\n'))
}

func LoadMessages() ([]Message, error) {
	var messages []Message

	file, err := os.Open(msgPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var msg Message
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil {
			messages = append(messages, msg)
		}
	}
	return messages, scanner.Err()
}
