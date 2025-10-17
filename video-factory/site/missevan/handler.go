package missevan

import (
	"net/http"
	"video-factory/config"
	"video-factory/manager"
)

const baseURLPrefix = "missevan"

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
	// 猫耳需要特定的 Host
	header := make(http.Header)
	header.Set("Host", "d1-missevan04.bilivideo.com")
	return header
}

// HandlerStrategySingleton 供路由使用的单例
var HandlerStrategySingleton = HandlerStrategy{}
