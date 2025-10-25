package vo

import "time"

type RoomVO struct {
	ID              string    `json:"id"`
	RealID          string    `json:"realId"`
	Name            string    `json:"name"`
	LiveStatus      int       `json:"liveStatus"` // 0: 未开播 1: 正在直播
	Status          int       `json:"status"`     // 0: 未启动 1: 运行中
	ProxyURL        string    `json:"proxyUrl"`
	URL             string    `json:"url"`
	Platform        string    `json:"platform"`
	LastRefreshTime time.Time `json:"lastRefreshTime"`
	CreateTime      time.Time `json:"createTime"`
	UpdateTime      time.Time `json:"updateTime"`
}
