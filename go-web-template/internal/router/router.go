package router

import (
	"github.com/gin-gonic/gin"
	"go-web-template/internal/handler"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	api := r.Group("/api")
	{
		api.GET("/users/all", handler.GetUsers)
		api.GET("/users", handler.ListUsers) // GET /api/users?page=1&page_size=10&name=张三&gender=1
		api.POST("/users", handler.CreateUser)
		api.GET("/users/:id", handler.GetUser)
		api.PUT("/users", handler.UpdateUser)
		api.DELETE("/users/:id", handler.DeleteUser)
	}
	return r
}
