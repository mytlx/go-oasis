package iface

import (
	"net/http"
	"net/url"
	"time"
)

type Manager interface {
	AutoRefresh()
	StopAutoRefresh()
	Refresh(retryTimes int) error
	Fetch(baseURL string, params url.Values, extraHeader http.Header) (*http.Response, error)

	GetId() string
	GetCurrentURL() string
	GetProxyURL() string
}

// RefreshStrategy 定义了刷新核心业务逻辑的策略
type RefreshStrategy interface {
	// ExecuteFetchStreamInfo 负责执行具体的网络请求和数据解析
	ExecuteFetchStreamInfo() (*StreamInfo, error)
	// ParseExpiration 从 URL 字符串中解析出过期时间
	ParseExpiration(streamUrl string) (time.Time, error)
}