package iface

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

type Manager interface {
	Start(ctx context.Context) error
	AutoRefresh()
	StopAutoRefresh()
	Refresh(ctx context.Context, retryTimes int) error
	Fetch(ctx context.Context, baseURL string, params url.Values, extraHeader http.Header) (*http.Response, error)

	GetLiveStatus() (bool, error)

	GetId() int64
	GetCurrentURL() string
	GetProxyURL() string
	GetLastRefreshTime() time.Time
}

// RefreshStrategy 定义了刷新核心业务逻辑的策略
type RefreshStrategy interface {
	// ExecuteFetchStreamInfo 负责执行具体的网络请求和数据解析
	ExecuteFetchStreamInfo(certainQnFlag bool) (*StreamInfo, error)
	// ParseExpiration 从 URL 字符串中解析出过期时间
	ParseExpiration(streamUrl string) (time.Time, error)
}
