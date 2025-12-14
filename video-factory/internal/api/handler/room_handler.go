package handler

import (
	"errors"
	"fmt"
	"strconv"
	"video-factory/internal/api/response"
	"video-factory/internal/service"
	"video-factory/pkg/pool"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

// RoomAddHandler 添加直播间
func RoomAddHandler(pool *pool.ManagerPool) gin.HandlerFunc {
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
		// var strategy SiteStrategy
		// switch platform {
		// case "bili":
		// 	strategy = bili.HandlerStrategySingleton
		// case "missevan":
		// 	strategy = missevan.HandlerStrategySingleton
		// default:
		// 	response.Error(c, "平台参数有误")
		// }
		// if req.Platform != "bili" && req.Platform != "missevan" {
		// 	response.Error(c, "平台参数有误")
		// 	return
		// }

		// 检查是否已存在
		// if m, ok := pool.Get(rid); ok {
		// 	response.Error(c, fmt.Sprintf("房间[%s]已存在，请访问：%s", rid, m.GetProxyURL()))
		// 	return
		// }

		if err := service.AddRoom(req.RoomInput, req.Platform, pool.Config); err != nil {
			response.Error(c, err.Error())
			return
		}

		response.OkWithMsg(c, "添加房间成功")
	}
}

// RoomRemoveHandler 删除直播间
func RoomRemoveHandler(pool *pool.ManagerPool) gin.HandlerFunc {
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
		if managerObj, ok := pool.Get(roomId); ok {
			// 停止自动刷新
			managerObj.StopAutoRefresh()
			// 从 ManagerPool 移除
			pool.Remove(roomId)
		}
		// 删除数据库
		err = service.RemoveRoom(roomId)
		if err != nil {
			log.Err(err).Msgf("删除房间失败 %d", roomId)
			response.Error(c, fmt.Sprintf("删除房间失败: %v", err))
			return
		}
		response.OkWithMsg(c, "删除成功")
	}
}

// RoomDetailHandler 获取直播间详情
func RoomDetailHandler(pool *pool.ManagerPool) gin.HandlerFunc {
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
		roomVO, err := service.GetRoomVO(roomId)
		if err != nil {
			log.Err(err)
			response.Error(c, "获取详情失败")
			return
		}

		response.OkWithData(c, roomVO)
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

func RoomStatusHandler(pool *pool.ManagerPool) gin.HandlerFunc {
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

		if err := service.ChangeRoomStatus(req.RoomId, req.TargetStatus); err != nil {
			log.Err(err)
			response.Error(c, "修改房间状态失败")
			return
		}

		response.Ok(c)
	}
}
