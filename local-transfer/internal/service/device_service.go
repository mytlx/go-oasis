package service

import (
	"fmt"
	"local-transfer/internal/dao"
	"local-transfer/internal/model"
	"local-transfer/pkg/utils"
	"strings"
)

func InsertDevice(device *model.Device) error {
	if device == nil {
		return fmt.Errorf("invalid device: %v", device)
	}
	if device.ID == 0 {
		device.ID = utils.MustNextID()
	}
	return dao.CreateDevice(device)
}

func GetDeviceById(id int64) (*model.Device, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid device id: %d", id)
	}
	return dao.GetDeviceById(id)
}

// GetDeviceTypeByUA 设备类型解析
func GetDeviceTypeByUA(ua string) model.DeviceType {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "android"):
		return model.DeviceTypeAndroid
	case strings.Contains(ua, "iphone"):
		return model.DeviceTypeIPhone
	case strings.Contains(ua, "ipad"):
		return model.DeviceTypeIPad
	case strings.Contains(ua, "windows"):
		return model.DeviceTypeWindows
	case strings.Contains(ua, "macintosh"):
		return model.DeviceTypeMacOS
	default:
		return model.DeviceTypeUnknown
	}
}
