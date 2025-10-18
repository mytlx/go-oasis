package bili

import (
	"net/http"
	"video-factory/config"
	"video-factory/manager"
)

const baseURLPrefix = "bili"

// HandlerStrategySingleton 供路由使用的单例
var HandlerStrategySingleton = HandlerStrategy{}

// HandlerStrategy 实现了 SiteStrategy 接口
type HandlerStrategy struct{}

func (HandlerStrategy) GetBaseURLPrefix() string {
	return baseURLPrefix
}

func (HandlerStrategy) CreateManager(rid string, config *config.AppConfig) (manager.IManager, error) {
	// 委托给 NewManager
	return NewManager(rid, config)
}

func (HandlerStrategy) GetExtraHeaders() http.Header {
	// B站通常不需要特殊的额外 Header，返回 nil
	return nil
}


