package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"time"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/pool"
	"video-factory/web"
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
	// r.RedirectTrailingSlash = false
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

	// 1. 获取 'dist' 目录的子文件系统
	// staticFp 现在代表 'dist' 目录
	// staticFp, err := fs.Sub(web.FS, "dist")
	// if err != nil {
	// 	panic(err)
	// }
	// 将 http.FS 包装成 http.FileSystem
	// httpFS := http.FS(staticFp)
	// r.StaticFS("/assets", httpFS)
	//
	// http.FileSystem.Open(httpFS, "index.html")

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
	// 嵌入的 embed 就是一个文件系统，获取 web 目录下的资源文件
	staticFp, _ := fs.Sub(web.FS, "dist")
	httpFS := http.FS(staticFp)
	// r.StaticFS("/assets", http.FS(staticFs))
	// // 所有请求先匹配 api 路由，如果没匹配到就当作静态资源文件处理
	// r.NoRoute(gin.WrapH(http.FileServer(http.FS(staticFp))))

	// r.NoRoute(func(c *gin.Context) {
	// 	path := c.Request.RequestURI
	// 	if path == "/" || strings.HasSuffix(path, ".svg") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".ico") || strings.HasSuffix(path, ".html") {
	// 		// gin.WrapH(http.FileServer(gin.Dir("static", false)))(c)
	// 		gin.WrapH(http.FileServer(http.FS(staticFp)))(c)
	// 	} else {
	// 		// c.File("static/index.html")
	// 		serveIndexHtml(c, staticFp)
	// 	}
	//
	// })

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
		serveIndexHtml(c, staticFp)
	})

	// =================================================================
	// 3. 静态资源 /assets
	// 假设你的 Vue build 后, JS/CSS 文件位于 'dist/assets/'
	// r.StaticFS("/assets", ...) 会将 /assets/app.js 映射到 staticFp (dist) 下的 'assets/app.js'
	// 你的原始代码是正确的

	// 4. NoRoute - 处理 SPA 路由和根目录下的静态文件 (如 favicon.ico)
	// r.NoRoute(func(c *gin.Context) {
	// 	// 检查是否是 API 请求，如果是，则返回 404
	// 	// (理论上 API 路由应该在前面被匹配，这里是双重保险)
	// 	// if strings.HasPrefix(c.Request.URL.Path, "/api/") {
	// 	// 	c.Status(http.StatusNotFound)
	// 	// 	return
	// 	// }
	//
	// 	// 尝试从 'dist' 目录中找到文件
	// 	// c.Request.URL.Path 是完整的路径，如 /favicon.ico 或 /users/123
	// 	// 我们需要修剪掉开头的 '/'
	// 	filePath := strings.TrimPrefix(c.Request.URL.Path, "/")
	//
	// 	// 尝试打开文件
	// 	f, err := staticFp.Open(filePath)
	// 	if err != nil {
	// 		// 文件不存在 (例如 /users/123)，
	// 		// 这说明它是一个 SPA 路由，我们返回 index.html
	// 		c.FileFromFS("index.html", httpFS)
	// 		return
	// 	}
	// 	defer f.Close()
	//
	// 	// 检查打开的是文件还是目录
	// 	stat, err := f.Stat()
	// 	if err != nil {
	// 		// 获取文件信息失败，同样返回 index.html
	// 		c.FileFromFS("index.html", httpFS)
	// 		return
	// 	}
	//
	// 	if stat.IsDir() {
	// 		// 如果请求的是一个目录 (例如 / 或 /some-dir/)
	// 		// 我们也返回 index.html
	// 		// http.FileServer 会自动处理 / -> /index.html，但我们在这里统一处理
	// 		c.FileFromFS("index.html", httpFS)
	// 		return
	// 	}
	//
	// 	// 文件存在且不是目录 (例如 /favicon.ico, /manifest.json)
	// 	// 使用 Gin 的 c.FileFromFS 来提供这个文件
	// 	// 它会正确设置 Content-Type 等
	// 	c.FileFromFS(filePath, httpFS)
	// })

	// r.NoRoute(func(c *gin.Context) {
	// 	// 检查是否是 API 请求 (如果 API 路由组在前面，这里是可选的保险)
	// 	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
	// 		c.Status(http.StatusNotFound)
	// 		return
	// 	}
	//
	// 	// 检查请求的路径是否看起来像一个静态文件 (例如 /favicon.ico)
	// 	filePath := strings.TrimPrefix(c.Request.URL.Path, "/")
	//
	// 	// 尝试打开文件
	// 	f, err := staticFp.Open(filePath)
	// 	if err != nil {
	// 		// 文件不存在 (例如 /users/123)，
	// 		// 这说明它是一个 SPA 路由，我们返回 index.html
	// 		c.FileFromFS("index.html", httpFS)
	// 		return
	// 	}
	// 	defer f.Close()
	//
	// 	// 检查打开的是文件还是目录
	// 	stat, err := f.Stat()
	// 	if err != nil {
	// 		// 获取文件信息失败，同样返回 index.html
	// 		c.FileFromFS("index.html", httpFS)
	// 		return
	// 	}
	//
	// 	if stat.IsDir() {
	// 		// 如果请求的是一个目录 (例如 /)
	// 		// 我们也返回 index.html
	// 		c.FileFromFS("index.html", httpFS)
	// 		return
	// 	}
	//
	// 	// 文件存在且不是目录 (例如 /favicon.ico)
	// 	// 提供这个文件
	// 	c.FileFromFS(filePath, httpFS)
	// })

	// r.NoRoute(func(c *gin.Context) {
	// 	// 检查是否是 API 请求 (双重保险)
	// 	// if strings.HasPrefix(c.Request.URL.Path, "/api/") {
	// 	// 	c.Status(http.StatusNotFound)
	// 	// 	return
	// 	// }
	// 	log.Println("--- 调试: 已经进入 NoRoute 处理器 ---", "请求路径:", c.Request.URL.Path)
	// 	// 检查是否是 /assets 请求 (双重保险)
	// 	if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
	// 		c.Status(http.StatusNotFound)
	// 		return
	// 	}
	//
	// 	// 提取请求的文件路径
	// 	// / -> ""
	// 	// /some/page -> "some/page"
	// 	// /favicon.ico -> "favicon.ico"
	// 	filePath := strings.TrimPrefix(c.Request.URL.Path, "/")
	//
	// 	// 尝试打开文件
	// 	f, err := staticFp.Open(filePath)
	// 	if err != nil {
	// 		// 文件不存在 (例如 /some/page)，
	// 		// 这说明它是一个 SPA 路由，我们返回 index.html
	// 		// c.FileFromFS("/index.html", httpFS)
	// 		serveIndexHtml(c, staticFp)
	// 		return
	// 	}
	// 	defer f.Close()
	//
	// 	stat, err := f.Stat()
	// 	if err != nil {
	// 		// 获取文件信息失败，同样返回 index.html
	// 		// c.FileFromFS("/index.html", httpFS)
	// 		serveIndexHtml(c, staticFp)
	// 		return
	// 	}
	//
	// 	// 如果请求的是一个目录 (例如 /)
	// 	// 我们也返回 index.html
	// 	if stat.IsDir() {
	// 		// c.FileFromFS("/index.html", httpFS)
	// 		serveIndexHtml(c, staticFp)
	// 		return
	// 	}
	//
	// 	// 文件存在且不是目录 (例如 /favicon.ico, /manifest.json)
	// 	// 提供这个文件
	// 	c.FileFromFS(filePath, httpFS)
	// })

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