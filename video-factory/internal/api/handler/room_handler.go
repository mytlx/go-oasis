package handler

import (
	"errors"
	"fmt"
	"strconv"
	"video-factory/internal/api/response"
	"video-factory/internal/service"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

type RoomHandler struct {
	pool        *pool.ManagerPool
	config      *config.AppConfig
	roomService *service.RoomService
}

func NewRoomHandler(pool *pool.ManagerPool, config *config.AppConfig, roomService *service.RoomService) *RoomHandler {
	return &RoomHandler{
		pool:        pool,
		config:      config,
		roomService: roomService,
	}
}

// RoomAddHandler 添加直播间
func (r *RoomHandler) RoomAddHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RoomInput string `json:"roomInput" binding:"required"`
			Platform  string `json:"platform" binding:"oneof=bili missevan"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			var ve validator.ValidationErrors
			if errors.As(err, &ve) {
				for _, fe := range ve {
					switch fe.Field() {
					case "RoomInput":
						response.Error(c, "房间标识不能为空")
						return
					case "Platform":
						response.Error(c, "平台参数有误")
						return
					}
				}
			}
			response.Error(c, "参数错误")
			return
		}

		if err := r.roomService.AddRoom(req.RoomInput, req.Platform, r.pool.Config); err != nil {
			response.Error(c, err.Error())
			return
		}

		response.OkWithMsg(c, "添加房间成功")
	}
}

// RoomRemoveHandler 删除直播间
func (r *RoomHandler) RoomRemoveHandler() gin.HandlerFunc {
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
		// tlxTODO: clear
		if managerObj, ok := r.pool.Get(roomId); ok {
			// 停止自动刷新
			managerObj.StopAutoRefresh()
			// 从 ManagerPool 移除
			r.pool.Remove(roomId)
		}
		// 删除数据库
		err = r.roomService.RemoveRoom(roomId)
		if err != nil {
			log.Err(err).Msgf("删除房间失败 %d", roomId)
			response.Error(c, fmt.Sprintf("删除房间失败: %v", err))
			return
		}
		response.OkWithMsg(c, "删除成功")
	}
}

// RoomDetailHandler 获取直播间详情
func (r *RoomHandler) RoomDetailHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		roomIdStr := c.Param("roomId")
		if roomIdStr == "" {
			response.Error(c, "roomId 不能为空")
			return
		}
		roomId, err := strconv.ParseInt(roomIdStr, 10, 64)
		if err != nil {
			response.Error(c, "roomId 格式有误")
			return
		}
		roomVO, err := r.roomService.GetRoomVO(roomId)
		if err != nil {
			log.Err(err).Msgf("获取详情失败")
			response.Error(c, "获取详情失败")
			return
		}

		response.OkWithData(c, roomVO)
	}
}

// RoomListHandler 获取房间列表
func (r *RoomHandler) RoomListHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		rooms, err := r.roomService.ListRooms()
		if err != nil {
			log.Err(err).Msg("获取房间列表失败")
			response.Error(c, "获取房间列表失败")
			return
		}

		response.OkWithList(c, rooms, int64(len(rooms)), 0, 0)
	}
}

func (r *RoomHandler) RoomStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RoomId       string `json:"roomId"`
			TargetStatus int    `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, "请求参数有误")
			return
		}

		if req.RoomId == "" {
			response.Error(c, "房间 id 为空")
			return
		}
		if req.TargetStatus != 0 && req.TargetStatus != 1 {
			response.Error(c, "非法目标状态")
			return
		}

		if err := r.roomService.ChangeRoomStatus(req.RoomId, req.TargetStatus); err != nil {
			log.Err(err).Msgf("修改房间状态失败")
			response.Error(c, "修改房间状态失败")
			return
		}

		response.Ok(c)
	}
}

func (r *RoomHandler) RoomRecordStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RoomId       string `json:"roomId"`
			TargetStatus int    `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, "请求参数有误")
			return
		}

		if req.RoomId == "" {
			response.Error(c, "房间 id 为空")
			return
		}
		if req.TargetStatus != 0 && req.TargetStatus != 1 {
			response.Error(c, "非法目标状态")
			return
		}

		if err := r.roomService.ChangeRecordStatus(req.RoomId, req.TargetStatus); err != nil {
			log.Err(err).Msgf("修改房间录制状态失败")
			response.Error(c, "修改房间录制状态失败")
			return
		}

		response.Ok(c)
	}
}
