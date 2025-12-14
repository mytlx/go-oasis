package vo

import "time"

type RoomAddVO struct {
	// ID         int64  `json:"id"`
	Platform     string    `json:"platform"`
	ShortID      string    `json:"shortId"`
	RealID       string    `json:"realId"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	CoverURL     string    `json:"coverUrl"`
	ProxyURL     string    `json:"proxyUrl"`
	AnchorID     string    `json:"anchorId"`
	AnchorName   string    `json:"anchorName"`
	AnchorAvatar string    `json:"anchorAvatar"`
	CreateTime   time.Time `json:"createTime"`
	UpdateTime   time.Time `json:"updateTime"`
	// LiveStatus      int       `json:"liveStatus"` // 0: 未开播 1: 正在直播
	// Status          int       `json:"status"`     // 0: 未启动 1: 运行中
	// ProxyURL        string    `json:"proxyUrl"`
	// URL             string    `json:"url"`
	// Platform        string    `json:"platform"`
	// LastRefreshTime time.Time `json:"lastRefreshTime"`

}
