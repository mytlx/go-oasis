package repository

import (
	"errors"
	"video-factory/internal/domain/model"

	"gorm.io/gorm"
)

type RoomRepository struct {
	db *gorm.DB
}

func NewRoomRepository(db *gorm.DB) *RoomRepository {
	return &RoomRepository{
		db: db,
	}
}

func (r *RoomRepository) AddRoom(room *model.Room) error {
	return r.db.Create(room).Error
}

func (r *RoomRepository) AddOrUpdateRoom(room *model.Room) error {
	// 有主键就更新，无主键就插入
	return r.db.Save(room).Error
}

func (r *RoomRepository) RemoveRoom(id int64) error {
	return r.db.Delete(&model.Room{}, id).Error
}

// UpdateRoomExceptNil UpdateRoom 安全更新房间信息
func (r *RoomRepository) UpdateRoomExceptNil(room *model.Room) error {
	if room.ID == 0 {
		return errors.New("room ID 不能为空")
	}

	// 先检查记录是否存在
	var existing model.Room
	if err := r.db.First(&existing, "id = ?", room.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("room 不存在")
		}
		return err
	}

	// 只更新非零字段（结构体零值字段不会覆盖数据库已有值）
	if err := r.db.Model(&existing).Updates(room).Error; err != nil {
		return err
	}

	return nil
}

func (r *RoomRepository) UpdateRoomById(id int64, updateMap map[string]any) error {
	if id == 0 {
		return errors.New("room ID 不能为空")
	}

	var existing model.Room
	if err := r.db.First(&existing, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("room 不存在")
		}
		return err
	}

	if err := r.db.Model(&model.Room{}).Where("id = ?", id).Updates(updateMap).Error; err != nil {
		return err
	}
	return nil
}

func (r *RoomRepository) ListRooms() ([]model.Room, error) {
	var rooms []model.Room
	err := r.db.Find(&rooms).Error
	return rooms, err
}

func (r *RoomRepository) GetRoomById(id int64) (*model.Room, error) {
	var room model.Room
	err := r.db.First(&room, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没查到
		}
		return nil, err // 其他数据库错误
	}
	return &room, nil
}

func (r *RoomRepository) GetRoomByRealId(id string) (*model.Room, error) {
	var room model.Room
	result := r.db.Where("real_id = ?", id).First(&room)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil // 查不到返回 nil
		}
		return nil, result.Error // 其他错误
	}

	return &room, nil
}
