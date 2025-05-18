package ws

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"local-transfer/internal/model"
	"local-transfer/internal/service"
	"log"
	"sync"
)

// Client 表示一个连接客户端
type Client struct {
	Conn       *websocket.Conn
	DeviceId   int64
	DeviceName string
	DeviceType model.DeviceType
	IP         string
}

var (
	clients   = make(map[*websocket.Conn]*Client)
	clientsMu sync.Mutex
)

// deviceId -> client
var (
	DeviceClientMap = make(map[int64]*Client)
	DeviceClientMu  sync.RWMutex
)

func InitClient(c *gin.Context, conn *websocket.Conn) *Client {
	// 初始注册为匿名
	clientObj := &Client{
		Conn:       conn,
		DeviceId:   0,
		DeviceName: "unknown",
		DeviceType: service.GetDeviceTypeByUA(c.Request.UserAgent()),
		IP:         c.ClientIP(),
	}
	AddClient(conn, clientObj)
	return clientObj
}

func AddClient(conn *websocket.Conn, c *Client) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	clients[conn] = c
	log.Println("有客户端连接 WebSocket")
}

func RemoveClient(conn *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	delete(clients, conn)
	log.Println("客户端断开 WebSocket")
}

func GetClient(conn *websocket.Conn) *Client {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	c, _ := clients[conn]
	return c
}

func ListClients() []*Client {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	var list []*Client
	for _, c := range clients {
		list = append(list, c)
	}
	return list
}

func AddDeviceClient(deviceId int64, c *Client) {
	// TODO: 持久化设备信息，启动时加载设备信息
	DeviceClientMu.Lock()
	defer DeviceClientMu.Unlock()
	DeviceClientMap[deviceId] = c
}

func GetDeviceClient(deviceId int64) *Client {
	DeviceClientMu.RLock()
	defer DeviceClientMu.RUnlock()
	return DeviceClientMap[deviceId]
}

func RemoveDeviceClient(deviceId int64) {
	DeviceClientMu.Lock()
	defer DeviceClientMu.Unlock()
	delete(DeviceClientMap, deviceId)
}
