package service

import (
	"go-web-template/internal/dao"
	"go-web-template/internal/model"
)

func ListAllUsers() ([]model.User, error) {
	return dao.GetAllUsers()
}

func ListUsers(name string, gender *int, page, pageSize int) ([]model.User, int64, error) {
	return dao.ListUsers(name, gender, page, pageSize)
}

func CreateUser(user *model.User) error {
	return dao.CreateUser(user)
}

func GetUserByID(id int64) (*model.User, error) {
	return dao.GetUserByID(id)
}

func UpdateUser(user *model.User) error {
	return dao.UpdateUser(user)
}

func DeleteUser(id int64) error {
	return dao.DeleteUser(id)
}
