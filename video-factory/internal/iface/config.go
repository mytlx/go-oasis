package iface

// type ConfigAccessor interface {
// 	GetBiliCookie() string
// 	GetMissevanCookie() string
// }

// ConfigSubscriber 定义了需要接收配置更新通知的接口
type ConfigSubscriber interface {
	OnConfigUpdate(key string, value string)
}
