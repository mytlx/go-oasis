package main

import (
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"local-transfer/internal/config"
	"local-transfer/internal/db"
	"local-transfer/internal/router"
	"local-transfer/pkg/utils"
)

func main() {
	config.InitConfig()
	db.InitDB()
	utils.InitIDGenerator(1)

	r := router.SetupRouter()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run(":8080")
}
