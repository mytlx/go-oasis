package vo

import "time"

// ManagerVO 运行时的状态快照
type ManagerVO struct {
	RoomID       int64      `json:"roomId"`
	RealID       string     `json:"realId"`
	Platform     string     `json:"platform"`
	Name         string     `json:"name"`
	CoverURL     string     `json:"cover_url"`
	AnchorName   string     `json:"anchorName"`
	AnchorID     string     `json:"anchorId"`
	AnchorAvatar string     `json:"anchor_avatar"`
	LiveStatus   int        `json:"liveStatus"`  // 0：未开播 1：直播中 2：轮播中
	URL          string     `json:"url"`         // 直播间地址
	ProxyURL     string     `json:"proxyUrl"`    // 代理地址
	CurrentURL   string     `json:"currentUrl"`  // 当前解析到的流地址
	LastRefresh  *time.Time `json:"lastRefresh"` // 最后刷新时间
	ExpireTime   *time.Time `json:"expireTime"`  // URL 过期时间

	RecordStatus      int     `json:"recordStatus"`      // 0：未录制 1：录制中
	RecordFile        string  `json:"recordFile"`        // 当前录制文件名
	RecordSize        int     `json:"recordSize"`        // 当前文件大小 (Byte)
	RecordSizeStr     string  `json:"recordSizeStr"`     // 当前文件大小字符串
	RecordDuration    float64 `json:"recordDuration"`    // 当前分片时长
	RecordDurationStr string  `json:"recordDurationStr"` // 当前分片时长字符串
}
