package dao

import (
	"go-web-template/internal/db"
	"go-web-template/internal/model"
)

func GetAllUsers() ([]model.User, error) {
	var users []model.User
	err := db.DB.Find(&users).Error
	return users, err
}

func ListUsers(name string, gender *int, page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	query := db.DB.Model(&model.User{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if gender != nil {
		query = query.Where("gender = ?", gender)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func CreateUser(user *model.User) error {
	return db.DB.Create(user).Error
}

func GetUserByID(id int64) (*model.User, error) {
	var user model.User
	err := db.DB.First(&user, id).Error
	return &user, err
}

func UpdateUser(user *model.User) error {
	return db.DB.Save(user).Error
}

func DeleteUser(id int64) error {
	return db.DB.Delete(&model.User{}, id).Error
}
