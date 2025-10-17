package router

import (
	"github.com/gin-gonic/gin"
	"video-factory/handler"
	"video-factory/pool"
	"video-factory/site/bili"
	"video-factory/site/missevan"
)

func NewEngine(p *pool.ManagerPool) *gin.Engine {
	// 1. 初始化 Gin 引擎 (Default 或 New 都可以，这里使用 Default)
	// 详细日志 gin.DebugMode，生产环境 gin.ReleaseMode
	gin.SetMode(gin.DebugMode)
	r := gin.Default()

	// 2. 添加全局中间件 (Gin Default 已经包含了 Logger 和 Recovery)
	// 如果需要其他全局中间件，可以在这里添加，例如 CORS
	// r.Use(cors.New(cors.Config{...}))

	// 3. 设置所有路由和分组
	setupRoutes(r, p)
	return r
}

func setupRoutes(r *gin.Engine, p *pool.ManagerPool) {
	// =================================================================
	// 核心代理流分组 (Group 1: /bili)
	// =================================================================
	biliGroup := r.Group("/" + bili.HandlerStrategySingleton.GetBaseURLPrefix())
	{
		// 房间管理 API (POST, DELETE, GET)
		biliGroup.POST("/room", handler.RoomAddHandler(p, bili.HandlerStrategySingleton))
		biliGroup.DELETE("/room", handler.RoomRemoveHandler(p, bili.HandlerStrategySingleton))
		biliGroup.GET("/room", handler.RoomDetailHandler(p, bili.HandlerStrategySingleton))

		// 代理流服务 (GET)
		// 匹配 /bili/:managerId/*file
		// :managerId 是路径参数
		// *file 是通配符，会匹配后面的所有内容（包含斜杠）
		biliGroup.GET("/:managerId/*file", handler.ProxyHandler(p, bili.HandlerStrategySingleton))
	}

	missevanGroup := r.Group("/" + missevan.HandlerStrategySingleton.GetBaseURLPrefix())
	{
		// 房间管理 API (POST, DELETE, GET)
		missevanGroup.POST("/room", handler.RoomAddHandler(p, missevan.HandlerStrategySingleton))
		missevanGroup.DELETE("/room", handler.RoomRemoveHandler(p, missevan.HandlerStrategySingleton))
		missevanGroup.GET("/room", handler.RoomDetailHandler(p, missevan.HandlerStrategySingleton))

		// 代理流服务 (GET)
		missevanGroup.GET("/:managerId/*file", handler.ProxyHandler(p, missevan.HandlerStrategySingleton))
	}

	// =================================================================
	// 网页后台管理分组 (Group 2: /admin)
	// =================================================================
	adminGroup := r.Group("/admin")
	{
		adminGroup.GET("/", func(c *gin.Context) {
			c.String(200, "这是后台管理首页")
		})
	}
}
