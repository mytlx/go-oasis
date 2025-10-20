package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strings"
	"video-factory/config"
	"video-factory/manager"
	"video-factory/pool"
	"video-factory/response"
	"video-factory/service"
)

type SiteStrategy interface {
	GetBaseURLPrefix() string
	CreateManager(rid string, appConfig *config.AppConfig) (manager.IManager, error)
	GetExtraHeaders() http.Header
}

// ProxyHandler 代理客户端请求到服务器
func ProxyHandler(pool *pool.ManagerPool, strategy SiteStrategy) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取路径参数 Manager ID
		managerID := c.Param("managerId")
		if managerID == "" {
			response.Error(c, "缺少ManagerId")
			return
		}
		// Gin 的 *file 通配符会包含匹配到的第一个斜杠，例如：/index.m3u8
		filenameWithSlash := c.Param("file")
		filename := strings.TrimPrefix(filenameWithSlash, "/")

		managerObj, ok := pool.Get(managerID)
		if !ok {
			// tlxTODO: 自动配置
			response.Error(c, fmt.Sprintf("直播间[%s]未配置", managerID))
			return
		}

		m := managerObj.Get()
		m.Mutex.RLock()
		currentURL := m.CurrentURL
		m.Mutex.RUnlock()

		parsedHlsUrl, err := url.Parse(currentURL)
		if err != nil {
			log.Err(err).Msg("解析hls源失败")
			response.Error(c, "Internal server error")
			return
		}
		var targetURL *url.URL

		if strings.HasSuffix(filename, ".m3u8") {
			targetURL = parsedHlsUrl
		} else if strings.HasSuffix(filename, ".ts") || strings.HasSuffix(filename, ".m4s") {
			tempUrl := *parsedHlsUrl
			lastSlash := strings.LastIndex(tempUrl.Path, "/")
			if lastSlash != -1 {
				// 截断路径，只保留目录部分（例如 /live-bvc/.../2500/）
				tempUrl.Path = tempUrl.Path[:lastSlash+1]
			} else {
				log.Err(err).Msg("hls源路径解析有误")
				response.Error(c, "Internal server error")
			}

			relativeURL, err := url.Parse(strings.TrimPrefix(filename, "/"))
			if err != nil {
				log.Err(err).Msg("解析相对路径失败")
				response.Error(c, "Internal server error")
				return
			}
			// 自动继承 scheme, host，并正确地将相对路径附加到基准路径上
			targetURL = tempUrl.ResolveReference(relativeURL)
			// 保留原始 token
			targetURL.RawQuery = parsedHlsUrl.RawQuery
		} else {
			log.Error().Msgf("不支持的文件类型或路径: %s", c.Request.URL.RequestURI())
			response.Error(c, "Unsupported file type or path")
		}

		// log.Printf("代理请求: %s -> %s", r.URL.RequestURI(), targetURL.String())

		// 转发请求
		resp, err := managerObj.Fetch(targetURL.String(), nil, strategy.GetExtraHeaders())
		if err != nil {
			log.Err(err).Msg("错误: 执行 HTTP 请求失败")
			response.Error(c, "Error fetching stream data")
			return
		}
		defer resp.Body.Close()

		// 转发response给客户端
		// 复制状态码和 Headers
		// 注意：M3U8 文件的 Content-Type 必须正确转发，通常是 application/vnd.apple.mpegurl
		for header, values := range resp.Header {
			for _, value := range values {
				c.Writer.Header().Add(header, value)
			}
		}
		c.Status(resp.StatusCode) // 最佳实践：先设置 Headers，再写入 Status Code

		// 复制响应体 (M3U8 内容或 TS 片段数据)
		if _, err = io.Copy(c.Writer, resp.Body); err != nil {
			log.Err(err).Msg("转发响应体失败")
		}
	}
}

// RoomAddHandler 添加直播间
func RoomAddHandler(pool *pool.ManagerPool, strategy SiteStrategy) gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.Query("rid")
		if rid == "" {
			response.Error(c, "rid不能为空")
			return
		}

		// 检查是否已存在
		if m, ok := pool.Get(rid); ok {
			response.Error(c, fmt.Sprintf("房间[%s]已存在，请访问：%s", rid, m.Get().ProxyURL))
			return
		}

		// 新建 Manager
		managerObj, err := strategy.CreateManager(rid, pool.Config)
		if err != nil {
			log.Err(err).Msgf("添加房间 %s", rid)
			response.OkWithMsg(c, fmt.Sprintf("添加房间失败: %v", err))
			return
		}

		// 添加到 ManagerPool
		pool.Add(managerObj.Get().Id, managerObj)

		// 启动自动续期
		managerObj.AutoRefresh()

		response.OkWithMsg(c, fmt.Sprintf("添加房间[%s]成功，请访问：%s", rid, managerObj.Get().ProxyURL))
	}
}

// RoomRemoveHandler 删除直播间
func RoomRemoveHandler(pool *pool.ManagerPool, strategy SiteStrategy) gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.Query("rid")
		if rid == "" {
			response.Error(c, "rid不能为空")
			return
		}
		managerObj, ok := pool.Get(rid)
		if !ok {
			response.Error(c, "房间不存在")
			return
		}
		managerObj.Get().StopAutoRefresh()
		pool.Remove(rid)
		response.OkWithMsg(c, fmt.Sprintf("删除房间[%s]成功", rid))
	}
}

// RoomDetailHandler 获取直播间详情
func RoomDetailHandler(pool *pool.ManagerPool, strategy SiteStrategy) gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.Query("rid")
		if rid == "" {
			response.Error(c, "rid不能为空")
			return
		}
		managerObj, ok := pool.Get(rid)
		if !ok {
			response.Error(c, "房间不存在")
			return
		}
		response.OkWithData(c, managerObj.Get().ProxyURL)
	}
}

// RoomListHandler 获取房间列表
func RoomListHandler(pool *pool.ManagerPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		rooms, err := service.ListRooms(pool)
		if err != nil {
			log.Err(err).Msg("获取房间列表失败")
			response.Error(c, "获取房间列表失败")
			return
		}

		response.OkWithList(c, rooms, int64(len(rooms)), 0, 0)
	}
}
