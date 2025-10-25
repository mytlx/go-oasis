package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"regexp"
	"time"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/pool"
)

// LoggerSkipPaths 是一个自定义中间件，用于跳过特定路径的日志
func LoggerSkipPaths(skipPatterns []string) gin.HandlerFunc {
	// 预编译所有正则表达式，避免每次请求都重新编译
	var regexList []*regexp.Regexp
	for _, pattern := range skipPatterns {
		regexList = append(regexList, regexp.MustCompile(pattern))
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 检查路径是否匹配任何一个正则
		for _, re := range regexList {
			if re.MatchString(path) {
				// 匹配成功则跳过日志打印
				c.Next()
				return
			}
		}

		// 未匹配则使用默认的 Gin Logger
		gin.Logger()(c)
	}
}

func NewEngine(p *pool.ManagerPool) *gin.Engine {
	// 1. 初始化 Gin 引擎 (Default 或 New 都可以，这里使用 Default)
	// 详细日志 gin.DebugMode，生产环境 gin.ReleaseMode
	// gin.SetMode(gin.ReleaseMode)
	gin.SetMode(gin.DebugMode)
	r := gin.New()

	// 2. 添加全局中间件 (Gin Default 包含 Logger 和 Recovery)
	r.Use(gin.Recovery())
	// 日志拦截
	r.Use(LoggerSkipPaths([]string{
		`^/[^/]+/proxy/\d+/.*`, // 拦截代理请求
	}))
	// 跨域
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // 允许所有来源，生产环境应限制
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		AllowWebSockets:  true,
	}))

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
		biliGroup.POST("/room", RoomAddHandler(p, bili.HandlerStrategySingleton))
		// biliGroup.DELETE("/room", RoomRemoveHandler(p, bili.HandlerStrategySingleton))
		// biliGroup.GET("/room", RoomDetailHandler(p, bili.HandlerStrategySingleton))
		// biliGroup.GET("/room/list", handler.RoomListHandler(p))

		// 代理流服务 (GET)
		// 匹配 /bili/:managerId/*file
		// :managerId 是路径参数
		// *file 是通配符，会匹配后面的所有内容（包含斜杠）
		biliGroup.GET("/proxy/:managerId/*file", ProxyHandler(p, bili.HandlerStrategySingleton))
	}

	missevanGroup := r.Group("/" + missevan.HandlerStrategySingleton.GetBaseURLPrefix())
	{
		// 房间管理 API (POST, DELETE, GET)
		missevanGroup.POST("/room", RoomAddHandler(p, missevan.HandlerStrategySingleton))
		// missevanGroup.DELETE("/room", RoomRemoveHandler(p, missevan.HandlerStrategySingleton))
		// missevanGroup.GET("/room", RoomDetailHandler(p, missevan.HandlerStrategySingleton))

		// 代理流服务 (GET)
		missevanGroup.GET("/proxy/:managerId/*file", ProxyHandler(p, missevan.HandlerStrategySingleton))
	}

	r.GET("/room/list", RoomListHandler(p))
	r.DELETE("/room", RoomRemoveHandler(p))
	r.GET("/room", RoomDetailHandler(p))
	r.POST("/room/refresh", RefreshHandler(p))
	r.POST("/room/stop", StopHandler(p))
	r.POST("/room/start", StartHandler(p))

	configGroup := r.Group("/config")
	{
		configGroup.GET("/list", ConfigListHandler(p))
		configGroup.POST("/add", ConfigAddHandler(p))
		configGroup.POST("/update", ConfigUpdateHandler(p))
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

	// r.StaticFS("/", http.Dir("./dist")) // 假设 Vue/Vite 的构建输出在 dist 目录

}
