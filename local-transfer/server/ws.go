package server

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"local-transfer/utils"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client 客户端结构体
type Client struct {
	Conn       *websocket.Conn
	DeviceId   string
	DeviceName string
	DeviceType string
	IP         string
}

// 管理连接的 map
var (
	clients   = make(map[*websocket.Conn]Client)
	clientsMu sync.Mutex
)

// 设备信息map
var (
	DeviceMap = make(map[string]Client)
	DeviceMu  sync.RWMutex
)

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域连接
	},
}

// 是否已启动 UDP 广播（避免重复）
var broadcastStarted atomic.Bool

// WsHandler WebSocket 连接入口
func WsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("升级为 WebSocket 失败:", err)
		return
	}
	defer conn.Close()

	// 获取客户端 IP 和 User-Agent
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ua := r.Header.Get("User-Agent")
	deviceType := getDeviceType(ua)

	// 初始注册为匿名
	clientsMu.Lock()
	clients[conn] = Client{
		Conn:       conn,
		DeviceId:   "",
		DeviceName: "unknown",
		DeviceType: deviceType,
		IP:         host,
	}
	clientsMu.Unlock()
	log.Println("有客户端连接 WebSocket")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		log.Println(string(msg))
		// todo: 消息样例
		var m map[string]string
		_ = json.Unmarshal(msg, &m)

		switch m["type"] {
		case "register":
			// 注册设备信息
			deviceName := m["deviceName"]
			deviceId := m["deviceId"]
			registerIp := m["ip"]
			clientsMu.Lock()
			if client, ok := clients[conn]; ok {
				client.DeviceName = deviceName
				client.DeviceId = deviceId
				client.IP = registerIp
				clients[conn] = client
				// 保存设备信息map
				DeviceMu.Lock()
				DeviceMap[deviceId] = client
				DeviceMu.Unlock()

				// 注册成功后广播设备信息（只启动一次）
				// if broadcastStarted.CompareAndSwap(false, true) {
				// 	go lan.StartBroadcaster(lan.DeviceInfo{
				// 		DeviceID:   deviceId,
				// 		DeviceName: deviceName,
				// 		DeviceType: client.DeviceType,
				// 		IP:         client.IP,
				// 	}, 3*time.Second)
				//
				// 	go lan.StartListener(func(d lan.DeviceInfo) {
				// 		log.Printf("发现设备: %s (%s)", d.DeviceName, d.IP)
				// 	})
				//
				// 	go lan.StartCleaner(15 * time.Second)
				// }
			}
			clientsMu.Unlock()

			broadcastDevices()

		case "text":
			BroadcastMsg(getClientInfo(conn), m["content"], "text")

		default:
			log.Println("未知消息类型:", m["type"])
		}
	}

	clientsMu.Lock()
	delete(clients, conn)
	clientsMu.Unlock()
	broadcastDevices()
	log.Println("客户端断开 WebSocket")
}

// 获取客户端信息
func getClientInfo(conn *websocket.Conn) Client {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	return clients[conn]
}

// 设备类型解析
func getDeviceType(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone"):
		return "iPhone"
	case strings.Contains(ua, "ipad"):
		return "iPad"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "macintosh"):
		return "macOS"
	default:
		return "Unknown"
	}
}

// BroadcastMsg 广播文本消息
func BroadcastMsg(c Client, content string, msgType string) {
	messageId, _ := utils.NextID()
	msg := map[string]string{
		"time":       time.Now().Format(time.RFC3339),
		"type":       msgType,
		"content":    content,
		"deviceId":   c.DeviceId,
		"deviceName": c.DeviceName,
		"deviceType": c.DeviceType,
		"ip":         c.IP,
		"messageId":  strconv.FormatInt(messageId, 10),
	}
	raw, _ := json.Marshal(msg)

	clientsMu.Lock()
	defer clientsMu.Unlock()
	for _, c := range clients {
		_ = c.Conn.WriteMessage(websocket.TextMessage, raw)
	}

	// 持久化消息
	SaveMsg(Message{
		ID:      messageId,
		Time:    msg["time"],
		Type:    msg["type"],
		Content: msg["content"],
		Source: Source{
			DeviceId:   msg["deviceId"],
			DeviceType: msg["deviceType"],
			DeviceName: msg["deviceName"],
			IP:         msg["ip"],
		},
	})
}

// 广播设备在线列表
func broadcastDevices() {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	var list []map[string]string
	for _, c := range clients {
		list = append(list, map[string]string{
			"deviceId":   c.DeviceId,
			"deviceName": c.DeviceName,
			"deviceType": c.DeviceType,
			"ip":         c.IP,
		})
	}

	msg := map[string]interface{}{
		"type": "devices",
		"list": list,
	}
	raw, _ := json.Marshal(msg)

	for _, c := range clients {
		_ = c.Conn.WriteMessage(websocket.TextMessage, raw)
	}
}
