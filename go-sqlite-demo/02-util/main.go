package main

import (
	"02-util/db"
	"fmt"
)

func main() {
	db.InitDB("../database/demo.db")

	// 插入
	id, err := db.InsertUser("Alice02", 22)
	if err != nil {
		panic(err)
	}
	fmt.Println("插入成功，ID =", id)

	// 查询
	users, err := db.GetAllUsers()
	if err != nil {
		panic(err)
	}
	fmt.Println("-------------查询所有----------------")
	for _, u := range users {
		fmt.Printf("用户：%3d, %10s, %3d\n", u.ID, u.Name, u.Age)
	}
	fmt.Println("-------------查询所有end----------------")

	// 更新
	err = db.UpdateUser(id, "AliceUpdated", 25)
	if err != nil {
		panic(err)
	}
	fmt.Println("更新成功")

	user, err := db.GetById(id)
	if err != nil {
		panic(err)
	}
	fmt.Println(user)

	// 删除
	err = db.DeleteUser(id)
	if err != nil {
		panic(err)
	}
	fmt.Println("删除成功")

	users, err = db.GetUsersWithPagination(1, 10)
	if err != nil {
		panic(err)
	}
	fmt.Println("-------------分页查询----------------")
	for _, u := range users {
		fmt.Printf("用户：%3d, %10s, %3d\n", u.ID, u.Name, u.Age)
	}
}
