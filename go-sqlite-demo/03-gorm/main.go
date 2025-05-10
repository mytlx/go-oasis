package main

import (
	"03-gorm/db"
	"03-gorm/model"
	"03-gorm/utils"
	"fmt"
)

func main() {
	// 初始化数据库
	db.InitDB()

	// 插入一个用户
	user := model.User{Name: "Alice", Age: 25}
	createdUser, err := db.CreateUser(user)
	if err != nil {
		utils.HandleError(err, "创建用户失败")
		return
	}
	fmt.Printf("用户创建成功: %+v\n", createdUser)

	// 查询用户
	fetchedUser, err := db.GetUserById(createdUser.ID)
	if err != nil {
		utils.HandleError(err, "查询用户失败")
		return
	}
	fmt.Printf("查询到的用户: %+v\n", fetchedUser)

	// 更新用户
	updatedUser := model.User{Name: "Bob", Age: 30}
	updatedUserInfo, err := db.UpdateUser(fetchedUser.ID, updatedUser)
	if err != nil {
		utils.HandleError(err, "更新用户失败")
		return
	}
	fmt.Printf("更新后的用户: %+v\n", updatedUserInfo)

	// 删除用户
	err = db.DeleteUser(updatedUserInfo.ID)
	if err != nil {
		utils.HandleError(err, "删除用户失败")
		return
	}
	fmt.Println("用户删除成功")
}
