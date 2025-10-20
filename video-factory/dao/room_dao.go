package dao

import (
	"errors"
	"gorm.io/gorm"
	"video-factory/db"
	"video-factory/model"
)

func AddRoom(room *model.Room) error {
	return db.DB.Create(room).Error
}

func AddOrUpdateRoom(room *model.Room) error {
	// 有主键就更新，无主键就插入
	return db.DB.Save(room).Error
}

func RemoveRoom(id int64) error {
	return db.DB.Delete(&model.Room{}, id).Error
}

// UpdateRoom 安全更新房间信息
func UpdateRoom(room *model.Room) error {
	if room.ID == "" {
		return errors.New("room ID 不能为空")
	}

	// 先检查记录是否存在
	var existing model.Room
	if err := db.DB.First(&existing, "id = ?", room.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("room 不存在")
		}
		return err
	}

	// 只更新非零字段（结构体零值字段不会覆盖数据库已有值）
	if err := db.DB.Model(&existing).Updates(room).Error; err != nil {
		return err
	}

	return nil
}

func ListRooms() ([]model.Room, error) {
	var rooms []model.Room
	err := db.DB.Find(&rooms).Error
	return rooms, err
}

func GetRoomById(id int64) (*model.Room, error) {
	var room model.Room
	err := db.DB.First(&room, id).Error
	return &room, err
}
