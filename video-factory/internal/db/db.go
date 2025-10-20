package db

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	DB, err = gorm.Open(sqlite.Open("./db/video-factory.db"), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("数据库连接失败")
	}

	// 自动迁移表结构
	// if err := DB.AutoMigrate(&domain.User{}); err != nil {
	// 	log.Fatal("表迁移失败:", err)
	// }

	fmt.Println("数据库连接成功并已迁移！")
}
