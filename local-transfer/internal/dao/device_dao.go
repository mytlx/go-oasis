package dao

import (
	"local-transfer/internal/db"
	"local-transfer/internal/model"
)

func CreateDevice(device *model.Device) error {
	return db.DB.Create(device).Error
}

func GetDeviceById(id int64) (*model.Device, error) {
	var device model.Device
	err := db.DB.First(&device, id).Error
	return &device, err
}
