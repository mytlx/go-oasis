package db

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"local-transfer/internal/model"
	"log"
)

var DB *gorm.DB

func InitDB() {
	var err error
	dbPath := viper.GetString("database.sqlite_path")
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 自动迁移建表
	err = DB.AutoMigrate(
		&model.Message{},
		&model.Device{},
	)
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to migrate tables: %v", err))
	}
}
