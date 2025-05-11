// @title go-web-template API
// @version 1.0
// @description This is a sample server celler server.
// @host localhost:8080
// @BasePath /

package main

import (
	"go-web-template/internal/config"
	"go-web-template/internal/db"
	"go-web-template/internal/router"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "go-web-template/docs"
)

func main() {
	config.InitConfig()
	db.InitDB()
	r := router.SetupRouter()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run(":8080")
}
