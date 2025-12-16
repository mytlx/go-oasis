package bili

import (
	"net/http"
	"video-factory/internal/iface"
	"video-factory/pkg/config"
)

const baseURLPrefix = "bili"

// HandlerStrategySingleton 供路由使用的单例
var HandlerStrategySingleton = HandlerStrategy{}

// HandlerStrategy 实现了 SiteStrategy 接口
type HandlerStrategy struct{}

func (HandlerStrategy) GetBaseURLPrefix() string {
	return baseURLPrefix
}

func (HandlerStrategy) CreateManager(rid int64, config *config.AppConfig) (iface.Manager, error) {
	// 委托给 NewManager
	// return NewManager(rid, config)
	return nil, nil
}

func (HandlerStrategy) GetExtraHeaders() http.Header {
	// B站通常不需要特殊的额外 Header，返回 nil
	return nil
}
