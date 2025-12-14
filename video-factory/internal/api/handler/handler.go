package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"video-factory/internal/api/response"
	"video-factory/internal/iface"
	"video-factory/internal/service"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type SiteStrategy interface {
	GetBaseURLPrefix() string
	CreateManager(roomId int64, appConfig *config.AppConfig) (iface.Manager, error)
	GetExtraHeaders() http.Header
}

// ProxyHandler 代理客户端请求到服务器
func ProxyHandler(pool *pool.ManagerPool, strategy SiteStrategy) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取路径参数 Manager ID
		managerIDStr := c.Param("managerId")
		if managerIDStr == "" {
			response.Error(c, "缺少ManagerId")
			return
		}
		managerID, err := strconv.ParseInt(managerIDStr, 10, 64)
		if err != nil {
			response.Error(c, "roomId 格式不正确")
			return
		}
		// Gin 的 *file 通配符会包含匹配到的第一个斜杠，例如：/index.m3u8
		filenameWithSlash := c.Param("file")
		filename := strings.TrimPrefix(filenameWithSlash, "/")

		managerObj, ok := pool.Get(managerID)
		if !ok {
			// tlxTODO: 自动配置
			response.Error(c, fmt.Sprintf("直播间[%d]未配置", managerID))
			return
		}

		parsedHlsUrl, err := url.Parse(managerObj.GetCurrentURL())
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
		resp, err := managerObj.Fetch(c.Request.Context(), targetURL.String(), nil, strategy.GetExtraHeaders())
		if err != nil {
			log.Err(err).Msg("错误: 执行 HTTP 请求失败")
			response.Error(c, "Error fetching stream db")
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

func RefreshHandler(pool *pool.ManagerPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomIdStr := c.Query("roomId")
		if roomIdStr == "" {
			response.Error(c, "roomId不能为空")
			return
		}
		roomId, err := strconv.ParseInt(roomIdStr, 10, 64)
		if err != nil {
			response.Error(c, "roomId 格式不正确")
			return
		}
		managerObj, ok := pool.Get(roomId)
		if !ok {
			response.Error(c, "房间不存在或状态有误")
			return
		}
		err = managerObj.Refresh(c.Request.Context(), 0)
		if err != nil {
			response.Error(c, fmt.Sprintf("刷新房间失败: %v", err))
			return
		}
		response.OkWithMsg(c, fmt.Sprintf("刷新房间[%d]成功", roomId))
	}
}

func StopHandler(pool *pool.ManagerPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomIdStr := c.Query("roomId")
		if roomIdStr == "" {
			response.Error(c, "roomId不能为空")
			return
		}
		roomId, err := strconv.ParseInt(roomIdStr, 10, 64)
		if err != nil {
			response.Error(c, "roomId 格式不正确")
			return
		}
		managerObj, ok := pool.Get(roomId)
		if !ok {
			response.Error(c, "房间不存在或状态有误")
			return
		}
		// 停止自动刷新
		managerObj.StopAutoRefresh()
		// 从 ManagerPool 移除
		pool.Remove(roomId)
		response.OkWithMsg(c, fmt.Sprintf("停止自动刷新房间[%d]成功", roomId))
	}
}

func StartHandler(pool *pool.ManagerPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		roomIdStr := c.Query("roomId")
		if roomIdStr == "" {
			response.Error(c, "roomId不能为空")
			return
		}
		roomId, err := strconv.ParseInt(roomIdStr, 10, 64)
		if err != nil {
			response.Error(c, "roomId 格式不正确")
			return
		}

		// 检查是否已存在
		if m, ok := pool.Get(roomId); ok {
			response.Error(c, fmt.Sprintf("房间[%d]已启动，请勿重复启动：%s", roomId, m.GetProxyURL()))
			return
		}

		room, err := service.GetRoom(roomId)
		if err != nil {
			log.Err(err).Msgf("查询房间失败：%s", roomId)
			response.OkWithMsg(c, fmt.Sprintf("未查询到该房间: %v", err))
			return
		}

		// 判断平台
		var strategy SiteStrategy
		if room.Platform == bili.HandlerStrategySingleton.GetBaseURLPrefix() {
			strategy = bili.HandlerStrategySingleton
		} else if room.Platform == missevan.HandlerStrategySingleton.GetBaseURLPrefix() {
			strategy = missevan.HandlerStrategySingleton
		}

		// 新建 Manager
		managerObj, err := strategy.CreateManager(roomId, pool.Config)
		if err != nil {
			log.Err(err).Msgf("添加房间 %s", roomId)
			response.OkWithMsg(c, fmt.Sprintf("启动房间失败: %v", err))
			return
		}

		// 添加到 ManagerPool
		pool.Add(managerObj.GetId(), managerObj)

		// 启动自动续期
		managerObj.AutoRefresh()

		response.OkWithMsg(c, fmt.Sprintf("启动房间[%s]成功，请访问：%s", roomId, managerObj.GetProxyURL()))
	}
}
