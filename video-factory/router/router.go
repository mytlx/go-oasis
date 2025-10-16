package router

import (
	"github.com/gin-gonic/gin"
	"video-factory/pool"
	"video-factory/site/bili"
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
	biliGroup := r.Group("/bili")
	{
		// 房间管理 API (POST, DELETE, GET)
		biliGroup.POST("/room", bili.RoomAddHandler(p))
		biliGroup.DELETE("/room", bili.RoomRemoveHandler(p))
		biliGroup.GET("/room", bili.RoomDetailHandler(p))

		// 代理流服务 (GET)
		// 匹配 /bili/:managerId/*file
		// :managerId 是路径参数
		// *file 是通配符，会匹配后面的所有内容（包含斜杠）
		biliGroup.GET("/:managerId/*file", bili.ProxyHandler(p))
	}

	// m, err := missevan.NewMissevan("109896001", "")
	// if err != nil {
	// 	log.Err(err).Msg("创建 Missevan 失败")
	// } else {
	// 	missevanGroup := r.Group("/missevan")
	// 	{
	// 		missevanGroup.GET("/*file", proxy.MissevanHandler(m))
	// 	}
	// }

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
