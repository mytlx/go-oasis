package router

import (
	"github.com/gin-gonic/gin"
	"local-transfer/internal/handler"
	"local-transfer/internal/ws"
	"time"

	"github.com/gin-contrib/cors"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// 加载 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有来源，生产环境应限制
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		AllowWebSockets:  true,
	}))

	api := r.Group("/api")
	{
		api.GET("/ws", ws.WsHandler)

		api.POST("/upload", handler.UploadFile)
		api.GET("/download/:filename", handler.DownloadFile)
		api.GET("/msg/list", handler.ListMsgHandler)
		api.GET("/device/:id", handler.GetDeviceByIdHandler)

		api.GET("/id/next", handler.GetNextIdHandler)
		api.GET("ip", handler.GetClientIPHandler)

		// api.GET("/users/all", handler.GetUsers)
		// api.GET("/users", handler.ListUsers) // GET /api/users?page=1&page_size=10&name=张三&gender=1
		// api.POST("/users", handler.CreateUser)
		// api.GET("/users/:id", handler.GetUser)
		// api.PUT("/users", handler.UpdateUser)
		// api.DELETE("/users/:id", handler.DeleteUser)
	}

	// r := gin.Default()
	//
	// // 设置静态资源目录
	// r.Static("/uploads", "./static/uploads")
	//
	// // 上传接口
	// r.POST("/upload", func(c *gin.Context) {
	// 	file, _ := c.FormFile("file")
	// 	dst := fmt.Sprintf("./static/uploads/%s", file.Filename)
	// 	c.SaveUploadedFile(file, dst)
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"url": "/uploads/" + file.Filename,
	// 	})
	// })

	return r
}
