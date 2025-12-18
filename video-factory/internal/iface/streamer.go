package iface

import (
	"net/http"
	"time"
)

// StreamInfo 包含通用的流媒体信息
type StreamInfo struct {
	AcceptQns  []int // 可以使用的清晰度
	SelectedQn int
	ActualQn   int               // 实际获得的清晰度编号
	StreamUrls map[string]string // 线路名 -> 完整的 HLS/Flv URL
}

// Streamer 定义了所有直播平台需要实现的方法
type Streamer interface {

	// 获取该平台特有的请求头 (Referer, User-Agent, Cookie 等)
	GetHeaders() http.Header

	// IsLive 检查直播间是否在直播中
	IsLive() (bool, error)

	// FetchStreamInfo 获取直播间的最新状态和流媒体 URL
	// currentQn: 用户请求的清晰度
	FetchStreamInfo(currentQn int, certainQnFlag bool) (*StreamInfo, error)

	// GetStreamInfo 获取内部成员变量副本
	GetStreamInfo() StreamInfo

	// ParseExpiration 解析直播源的过期时间
	ParseExpiration(streamUrl string) (time.Time, error)

	// Close 清理资源（如果需要）
	// Close()
}
