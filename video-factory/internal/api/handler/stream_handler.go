package handler

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"video-factory/internal/api/response"
	"video-factory/internal/service"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type StreamHandler struct {
	pool           *pool.ManagerPool
	config         *config.AppConfig
	roomService    *service.RoomService
	monitorService *service.MonitorService
}

func NewStreamHandler(pool *pool.ManagerPool, config *config.AppConfig,
	roomService *service.RoomService,
	monitorService *service.MonitorService,
) *StreamHandler {
	return &StreamHandler{
		pool:           pool,
		config:         config,
		roomService:    roomService,
		monitorService: monitorService,
	}
}

// ProxyHandler 代理客户端请求到服务器
func (s *StreamHandler) ProxyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取路径参数 Manager ID
		managerIDStr := c.Param("managerId")
		if managerIDStr == "" {
			response.Error(c, "缺少 ManagerId")
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

		managerPtr, ok := s.pool.Get(managerID)
		if !ok {
			response.Error(c, fmt.Sprintf("直播间[%d]未启用", managerID))
			return
		}

		targetURL, err := managerPtr.ResolveTargetURL(filename)
		if err != nil {
			log.Err(err)
			response.Error(c, "Internal server error")
			return
		}

		// log.Printf("代理请求: %s -> %s", c.Request.RequestURI, targetURL)

		// 转发请求
		resp, err := managerPtr.Fetch(c.Request.Context(), targetURL, nil)
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

func (s *StreamHandler) StartHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		roomIdStr := c.Param("roomId")
		if roomIdStr == "" {
			response.Error(c, "roomId 不能为空")
			return
		}
		roomId, err := strconv.ParseInt(roomIdStr, 10, 64)
		if err != nil {
			response.Error(c, "roomId 格式不正确")
			return
		}

		if err = s.monitorService.StartManager(roomId); err != nil {
			response.Error(c, err.Error())
		}

		response.Ok(c)
	}
}

func (s *StreamHandler) RefreshHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		roomIdStr := c.Param("roomId")
		if roomIdStr == "" {
			response.Error(c, "roomId 不能为空")
			return
		}
		roomId, err := strconv.ParseInt(roomIdStr, 10, 64)
		if err != nil {
			response.Error(c, "roomId 格式不正确")
			return
		}
		managerObj, ok := s.pool.Get(roomId)
		if !ok {
			response.Error(c, "房间不存在或状态有误")
			return
		}
		managerObj.TriggerRefresh()
		response.Ok(c)
	}
}

func (s *StreamHandler) StopHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		roomIdStr := c.Param("roomId")
		if roomIdStr == "" {
			response.Error(c, "roomId 不能为空")
			return
		}
		roomId, err := strconv.ParseInt(roomIdStr, 10, 64)
		if err != nil {
			response.Error(c, "roomId 格式不正确")
			return
		}
		managerObj, ok := s.pool.Get(roomId)
		if !ok {
			response.Error(c, "房间不存在或状态有误")
			return
		}
		// 停止自动刷新
		managerObj.StopAutoRefresh()

		response.Ok(c)
	}
}

func (s *StreamHandler) ListManager(c *gin.Context) {
	list, err := s.monitorService.GetManagerList()
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.OkWithList(c, list, int64(len(list)), 0, 0)
}
