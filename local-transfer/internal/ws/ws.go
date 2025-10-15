package ws

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"local-transfer/internal/model"
	"local-transfer/internal/service"
	"local-transfer/internal/vo"
	"local-transfer/pkg/utils"
	"log"
	"net/http"
	"time"
)

// WebSocket 升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域连接
	},
}

// WsHandler WebSocket 连接入口
func WsHandler(c *gin.Context) {
	log.Println("wsHandler")
	w := c.Writer
	r := c.Request

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("升级为 WebSocket 失败:", err)
		return
	}
	defer conn.Close()

	// 初始化client
	clientObj := InitClient(c, conn)
	for {
		_, msgByte, err := conn.ReadMessage()
		if err != nil {
			break
		}
		log.Println(string(msgByte))

		var msgVO vo.MessageVO
		if err := json.Unmarshal(msgByte, &msgVO); err != nil {
			log.Println("解析消息失败:", err)
			continue
		}
		switch msgVO.Type {
		case model.MessageTypeRegister:
			log.Printf("新设备连接：%s（ID: %d）", msgVO.Source.Name, msgVO.Source.ID)
			// 注册设备信息
			clientObj.DeviceName = msgVO.Source.Name
			clientObj.DeviceId = msgVO.Source.ID
			clientObj.IP = msgVO.Source.IP
			AddClient(conn, clientObj)
			AddDeviceClient(clientObj.DeviceId, clientObj)
			broadcastDevices()
		case model.MessageTypeText:
			// 一对一，source -> target
			SendAndSaveMessage(&msgVO)
		case model.MessageTypeImage:
			// 一对一，source -> target
			SendAndSaveMessage(&msgVO)
		case model.MessageTypeFile:
			// 一对一，source -> target
			SendAndSaveMessage(&msgVO)
		default:
			log.Println("未知消息类型:", msgVO.Type)
		}
	}

	RemoveClient(conn)
	RemoveDeviceClient(clientObj.DeviceId)
	broadcastDevices()
}

func SendAndSaveMessage(msgVO *vo.MessageVO) {
	messageId := utils.MustNextID()
	msgVO.ID = messageId
	msgVO.CreateTime = time.Now().Format(time.RFC3339)
	raw, _ := json.Marshal(msgVO)

	// 推送source，否则收不到回调
	sourceClient := GetDeviceClient(msgVO.Source.ID)
	_ = sourceClient.Conn.WriteMessage(websocket.TextMessage, raw)
	// 推送target
	targetClient := GetDeviceClient(msgVO.Target.ID)
	_ = targetClient.Conn.WriteMessage(websocket.TextMessage, raw)

	// 持久化消息
	service.InsertMessage(&model.Message{
		ID:         msgVO.ID,
		CreateTime: msgVO.CreateTime,
		SourceId:   msgVO.Source.ID,
		TargetId:   msgVO.Target.ID,
		Type:       msgVO.Type,
		Content:    msgVO.Content,
		Status:     1,
	})
}

// 广播设备在线列表
func broadcastDevices() {
	DeviceClientMu.Lock()
	defer DeviceClientMu.Unlock()

	log.Printf("当前在线设备数量：%d", len(DeviceClientMap))
	for id, c := range DeviceClientMap {
		log.Printf("设备ID: %d, 名称: %s, IP: %s", id, c.DeviceName, c.IP)
	}

	var list []map[string]any
	for _, c := range DeviceClientMap {
		list = append(list, map[string]any{
			"deviceId":   c.DeviceId,
			"deviceName": c.DeviceName,
			"deviceType": c.DeviceType,
			"ip":         c.IP,
		})
	}

	msg := map[string]any{
		"type": "devices",
		"list": list,
	}
	raw, _ := json.Marshal(msg)

	for _, c := range DeviceClientMap {
		_ = c.Conn.WriteMessage(websocket.TextMessage, raw)
	}
}
