package model

type Device struct {
	ID   int64      `json:"id,string" gorm:"primaryKey;not null"` // 设备ID
	IP   string     `json:"ip" gorm:"type:text;not null"`         // 设备IP
	Name string     `json:"name" gorm:"type:text;not null"`       // 设备名称
	Type DeviceType `json:"type" gorm:"type:text;not null"`       // 设备类型
}

func (Device) TableName() string {
	return "t_device"
}

type DeviceType string

const (
	DeviceTypeAndroid DeviceType = "Android"
	DeviceTypeIPhone  DeviceType = "iPhone"
	DeviceTypeIPad    DeviceType = "iPad"
	DeviceTypeWindows DeviceType = "Windows"
	DeviceTypeMacOS   DeviceType = "MacOS"
	DeviceTypeUnknown DeviceType = "Unknown"
)

// IsValid 校验合法性
func (dt DeviceType) IsValid() bool {
	switch dt {
	case DeviceTypeAndroid, DeviceTypeIPhone, DeviceTypeIPad, DeviceTypeWindows, DeviceTypeMacOS, DeviceTypeUnknown:
		return true
	default:
		return false
	}
}
