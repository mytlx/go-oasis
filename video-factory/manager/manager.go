package manager

import (
	"net/http"
	"net/url"
	"sync"
	"time"
	"video-factory/streamer"
)

type Manager struct {
	Id               string
	Streamer         streamer.Streamer `json:"-"`
	CurrentURL       string
	ActualExpireTime time.Time
	ExpectExpireTime time.Time
	LastRefresh      time.Time
	Mutex            sync.RWMutex `json:"-"`
}

type IManager interface {
	AutoRefresh()
	Refresh(retryTimes int) error
	Fetch(baseURL string, params url.Values, isRetry bool) (*http.Response, error)

	Get() *Manager
}
