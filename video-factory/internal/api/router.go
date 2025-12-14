package api

import (
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"time"
	"video-factory/internal/api/handler"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/pool"
	"video-factory/web"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	gin.SetMode(p.Config.GinLogMode)
	// gin.SetMode(gin.DebugMode)
	r := gin.New()
	// 2. 添加全局中间件 (Gin Default 包含 Logger 和 Recovery)
	r.Use(gin.Recovery())
	// 日志拦截
	r.Use(LoggerSkipPaths([]string{
		`^(/[^/]+)*/proxy/\d+/.*`, // 拦截代理请求
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
	api := r.Group("/api/v1")
	{
		biliGroup := api.Group("/" + bili.HandlerStrategySingleton.GetBaseURLPrefix())
		{
			// 房间管理 API (POST, DELETE, GET)
			// biliGroup.POST("/room", RoomAddHandler(p, bili.HandlerStrategySingleton))
			// biliGroup.DELETE("/room", RoomRemoveHandler(p, bili.HandlerStrategySingleton))
			// biliGroup.GET("/room", RoomDetailHandler(p, bili.HandlerStrategySingleton))
			// biliGroup.GET("/room/list", handler.RoomListHandler(p))

			// 代理流服务 (GET)
			// 匹配 /bili/:managerId/*file
			// :managerId 是路径参数
			// *file 是通配符，会匹配后面的所有内容（包含斜杠）
			biliGroup.GET("/proxy/:managerId/*file", handler.ProxyHandler(p, bili.HandlerStrategySingleton))
		}

		missevanGroup := api.Group("/" + missevan.HandlerStrategySingleton.GetBaseURLPrefix())
		{
			// 房间管理 API (POST, DELETE, GET)
			// missevanGroup.POST("/room", RoomAddHandler(p, missevan.HandlerStrategySingleton))
			// missevanGroup.DELETE("/room", RoomRemoveHandler(p, missevan.HandlerStrategySingleton))
			// missevanGroup.GET("/room", RoomDetailHandler(p, missevan.HandlerStrategySingleton))

			// 代理流服务 (GET)
			missevanGroup.GET("/proxy/:managerId/*file", handler.ProxyHandler(p, missevan.HandlerStrategySingleton))
		}

		roomGroup := api.Group("/room")
		{
			roomGroup.GET("/list", handler.RoomListHandler(p))
			roomGroup.DELETE("/:roomId", handler.RoomRemoveHandler(p))
			roomGroup.GET("/:roomId", handler.RoomDetailHandler(p))
			roomGroup.POST("/add", handler.RoomAddHandler(p))
			roomGroup.POST("/status", handler.RoomStatusHandler(p))

			roomGroup.POST("/refresh", handler.RefreshHandler(p))
			roomGroup.POST("/stop", handler.StopHandler(p))
			roomGroup.POST("/start", handler.StartHandler(p))
		}

		configGroup := api.Group("/config")
		{
			configGroup.GET("/list", handler.ConfigListHandler(p))
			configGroup.POST("/add", handler.ConfigAddHandler(p))
			configGroup.POST("/update", handler.ConfigUpdateHandler(p))
		}
	}

	// =================================================================
	// 网页后台管理分组 (Group 2: /admin)
	// =================================================================

	// 嵌入的 embed 就是一个文件系统，获取 web 目录下的资源文件
	distFS, err := fs.Sub(web.FS, "dist")
	if err != nil {
		// 如果 web/dist 目录不存在或为空，这里会报错，但 embed 成功的话通常不会
		panic(err)
	}
	httpFS := http.FS(distFS)

	// 手动修正路径，确保 assets 文件能被找到
	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.FileFromFS("assets"+c.Param("filepath"), httpFS)
	})
	r.HEAD("/assets/*filepath", func(c *gin.Context) {
		c.FileFromFS("assets"+c.Param("filepath"), httpFS)
	})

	r.NoRoute(func(c *gin.Context) {
		// 任何没有匹配到 API 或 /assets 的请求，都交给 NoRoute
		// 使用 serveIndexHtml 确保 200 OK 状态
		serveIndexHtml(c, distFS)
	})
}

// 辅助函数：手动读取 index.html 并以 200 OK 状态返回
func serveIndexHtml(c *gin.Context, fsys fs.FS) {
	file, err := fsys.Open("index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to open index.html from embedded FS")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to read index.html content")
		return
	}

	// 强制返回 200 OK 和 HTML Content-Type
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
