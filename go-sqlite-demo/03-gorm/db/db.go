package db

import (
	"03-gorm/model"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("../database/demo.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 自动迁移表结构
	if err := DB.AutoMigrate(&model.User{}); err != nil {
		log.Fatal("表迁移失败:", err)
	}

	fmt.Println("数据库连接成功并已迁移！")
}
