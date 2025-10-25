package iface

import (
	"net/http"
)

type Info struct {
	Header     http.Header
	Rid        string
	RealRoomId string
	Platform   string // 平台
	RoomUrl    string // 直播间 URL
	LiveStatus int    // 直播间状态 0:未开播 1:直播中
	StreamInfo *StreamInfo
}

// StreamInfo 包含通用的流媒体信息
type StreamInfo struct {
	AcceptQns  []int // 可以使用的清晰度
	SelectedQn int
	ActualQn   int               // 实际获得的清晰度编号
	StreamUrls map[string]string // 线路名 -> 完整的 HLS/Flv URL
}

// Streamer 定义了所有直播平台需要实现的方法
type Streamer interface {

	// InitRoom 初始化房间
	InitRoom() error

	// GetId 返回直播源的唯一标识符
	GetId() (string, error)

	// IsLive 检查直播间是否在直播中
	IsLive() (bool, error)

	// FetchStreamInfo 获取直播间的最新状态和流媒体 URL
	// currentQn: 用户请求的清晰度
	FetchStreamInfo(currentQn int, certainQnFlag bool) (*StreamInfo, error)

	// GetInfo 获取成员变量副本
	GetInfo() Info

	// GetStreamInfo 获取内部成员变量副本
	GetStreamInfo() StreamInfo

	// Close 清理资源（如果需要）
	// Close()
}

