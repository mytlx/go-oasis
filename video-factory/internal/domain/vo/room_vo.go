package vo

import "time"

type RoomVO struct {
	ID           string `json:"id"`
	Platform     string `json:"platform"`
	ShortID      string `json:"shortId"`
	RealID       string `json:"realId"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	CoverURL     string `json:"coverUrl"`
	ProxyURL     string `json:"proxyUrl"`
	AnchorName   string `json:"anchorName"`
	AnchorAvatar string `json:"anchorAvatar"`
	LiveStatus   int    `json:"liveStatus"` // 0: 未开播 1: 正在直播 2: 轮播中
	// StreamStatus    int       `json:"streamStatus"` // 0: 未启动 1: 运行中
	Status       int `json:"status"`       // 0: 禁用 1: 启用
	RecordStatus int `json:"recordStatus"` // 0: 禁用 1: 启用
	// LastRefreshTime time.Time `json:"lastRefreshTime"`
	CreateTime time.Time `json:"createTime"`
	UpdateTime time.Time `json:"updateTime"`
}
