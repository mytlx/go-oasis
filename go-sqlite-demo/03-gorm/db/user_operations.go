package db

import (
	"03-gorm/model"
	"03-gorm/utils"
)

// CreateUser 创建新用户
func CreateUser(user model.User) (*model.User, error) {
	if err := DB.Create(&user).Error; err != nil {
		utils.HandleError(err, "创建用户失败")
		return nil, err
	}
	return &user, nil
}

// GetUserById 根据ID获取用户
func GetUserById(id int64) (*model.User, error) {
	var user model.User
	if err := DB.First(&user, id).Error; err != nil {
		utils.HandleError(err, "未找到用户，id: %d", id)
		return nil, err
	}
	return &user, nil
}

// UpdateUser 更新用户信息
func UpdateUser(id int64, user model.User) (*model.User, error) {
	var existingUser model.User
	if err := DB.First(&existingUser, id).Error; err != nil {
		utils.HandleError(err, "未找到用户，id: %d", id)
		return nil, err
	}

	// 更新用户字段
	existingUser.Name = user.Name
	existingUser.Age = user.Age
	if err := DB.Save(&existingUser).Error; err != nil {
		utils.HandleError(err, "更新用户失败, id: %d", id)
		return nil, err
	}
	return &existingUser, nil
}

// DeleteUser 删除用户
func DeleteUser(id int64) error {
	var user model.User
	if err := DB.First(&user, id).Error; err != nil {
		utils.HandleError(err, "未找到用户，id: %d", id)
		return err
	}

	if err := DB.Delete(&user).Error; err != nil {
		utils.HandleError(err, "删除用户失败, id: %d", id)
		return err
	}
	return nil
}
