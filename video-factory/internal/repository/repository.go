package repository

import "gorm.io/gorm"

type Repository struct {
	Room   *RoomRepository
	Config *ConfigRepository
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		Room:   NewRoomRepository(db),
		Config: NewConfigRepository(db),
	}
}
