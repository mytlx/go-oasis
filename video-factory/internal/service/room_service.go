package service

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
	"video-factory/internal/domain/model"
	"video-factory/internal/domain/vo"
	"video-factory/internal/repository"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/config"
	"video-factory/pkg/pool"
	"video-factory/pkg/util"

	"github.com/rs/zerolog/log"
)

type RoomService struct {
	pool     *pool.ManagerPool
	config   *config.AppConfig
	roomRepo *repository.RoomRepository
}

func NewRoomService(pool *pool.ManagerPool, config *config.AppConfig, roomRepo *repository.RoomRepository) *RoomService {
	return &RoomService{
		pool:     pool,
		config:   config,
		roomRepo: roomRepo,
	}
}

func (r *RoomService) AddRoom(roomInput string, platform string, config *config.AppConfig) error {
	if roomInput == "" {
		return errors.New("地址参数为空")
	}

	var roomAddVO *vo.RoomAddVO
	var err error
	switch platform {
	case "bili":
		roomIdStr, err1 := bili.CheckAndGetRid(roomInput)
		if err1 != nil {
			return err1
		}
		room, err1 := r.CheckRoomExist(roomIdStr)
		if err1 != nil {
			return err1
		}
		if room != nil {
			return errors.New("房间已存在")
		}
		roomAddVO, err = bili.GetRoomAddInfo(roomIdStr)
	case "missevan":
		roomIdStr, err1 := missevan.CheckAndGetRid(roomInput)
		if err1 != nil {
			return err1
		}
		room, err1 := r.CheckRoomExist(roomIdStr)
		if err1 != nil {
			return err1
		}
		if room != nil {
			return errors.New("房间已存在")
		}
		roomAddVO, err = missevan.GetRoomAddInfo(roomIdStr)
	default:
		return errors.New("平台参数有误")
	}

	if err != nil {
		return err
	}
	if roomAddVO == nil {
		return errors.New("未获取到房间信息")
	}

	room := &model.Room{
		ID:           util.MustNextID(),
		Platform:     platform,
		ShortID:      roomAddVO.ShortID,
		RealID:       roomAddVO.RealID,
		Name:         roomAddVO.Name,
		URL:          roomAddVO.URL,
		CoverURL:     roomAddVO.CoverURL,
		ProxyURL:     fmt.Sprintf("http://localhost:%d/api/v1/%s/proxy/%s/index.m3u8", config.Port, platform, roomAddVO.RealID),
		AnchorID:     roomAddVO.AnchorID,
		AnchorName:   roomAddVO.AnchorName,
		AnchorAvatar: roomAddVO.AnchorAvatar,
		Status:       0,
		CreateTime:   time.Now().UnixMilli(),
		UpdateTime:   time.Now().UnixMilli(),
	}

	return r.roomRepo.AddRoom(room)
}

func (r *RoomService) CheckRoomExist(realId string) (*model.Room, error) {
	if realId == "" {
		return nil, errors.New("realId 为空")
	}
	room, err := r.roomRepo.GetRoomByRealId(realId)
	if err != nil {
		return nil, err
	}
	return room, nil
}

func (r *RoomService) ListRooms() ([]vo.RoomVO, error) {
	rooms, err := r.roomRepo.ListRooms()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var statusMutex sync.Mutex
	statusMap := make(map[int64]int, len(rooms))

	for _, room := range rooms {
		wg.Add(1) // 增加计数器
		go func(room *model.Room) {
			defer wg.Done()
			// 调用接口获取实时状态
			status, statusErr := r.GetRoomLiveStatus(room)
			if statusErr != nil {
				log.Warn().Msgf("警告: 获取房间[%d]直播状态失败: %v\n", room.ID, statusErr)
				// 失败默认离线
				status = 0
			}
			statusMutex.Lock()
			statusMap[room.ID] = status
			statusMutex.Unlock()
		}(&room)
	}
	wg.Wait()

	// poolSnapshot := pool.Snapshot()
	respList := make([]vo.RoomVO, len(rooms))
	for i, room := range rooms {
		// streamStatus := 0
		// m := poolSnapshot[room.ID]
		// lastRefreshTime := util.MillisToTime(room.UpdateTime)
		// if m != nil {
		// 	streamStatus = 1
		// 	lastRefreshTime = m.GetLastRefreshTime()
		// }
		liveStatus, ok := statusMap[room.ID]
		if !ok {
			liveStatus = 0
		}
		respList[i] = vo.RoomVO{
			ID:           strconv.FormatInt(room.ID, 10),
			Platform:     room.Platform,
			ShortID:      room.ShortID,
			RealID:       room.RealID,
			Name:         room.Name,
			URL:          room.URL,
			CoverURL:     room.CoverURL,
			ProxyURL:     room.ProxyURL,
			AnchorName:   room.AnchorName,
			AnchorAvatar: room.AnchorAvatar,
			LiveStatus:   liveStatus,
			Status:       room.Status,
			CreateTime:   util.MillisToTime(room.CreateTime),
			UpdateTime:   util.MillisToTime(room.UpdateTime),
		}
	}
	return respList, nil
}

func (r *RoomService) RemoveRoom(rid int64) error {
	// tlxTODO: clear manager by status
	return r.roomRepo.RemoveRoom(rid)
}

func (r *RoomService) GetRoom(roomId int64) (*model.Room, error) {
	if roomId == 0 {
		return nil, errors.New("roomId 为空")
	}
	return r.roomRepo.GetRoomById(roomId)
}

func (r *RoomService) GetRoomVO(roomId int64) (*vo.RoomVO, error) {
	room, err := r.GetRoom(roomId)
	if err != nil {
		log.Err(err)
		return nil, err
	}
	return &vo.RoomVO{
		ID:           strconv.FormatInt(room.ID, 10),
		Platform:     room.Platform,
		ShortID:      room.ShortID,
		RealID:       room.RealID,
		Name:         room.Name,
		URL:          room.URL,
		CoverURL:     room.CoverURL,
		ProxyURL:     room.ProxyURL,
		AnchorName:   room.AnchorName,
		AnchorAvatar: room.AnchorAvatar,
		// LiveStatus:      liveStatus,
		Status:     room.Status,
		CreateTime: util.MillisToTime(room.CreateTime),
		UpdateTime: util.MillisToTime(room.UpdateTime),
	}, nil
}

func (r *RoomService) GetRoomLiveStatus(room *model.Room) (int, error) {
	if room == nil {
		return 0, nil
	}
	switch room.Platform {
	case "bili":
		return bili.GetRoomLiveStatus(room.RealID)
	case "missevan":
		return missevan.GetRoomLiveStatus(room.RealID)
	default:
		return 0, nil
	}
}

func (r *RoomService) ChangeRoomStatus(roomIdStr string, targetStatus int) error {
	if roomIdStr == "" {
		return errors.New("入参为空")
	}
	if targetStatus != 0 && targetStatus != 1 {
		return errors.New("目标状态有误")
	}
	roomId, err := strconv.ParseInt(roomIdStr, 10, 64)
	if err != nil {
		log.Err(err).Msgf("入参转换类型失败: %s", roomIdStr)
		return errors.New("入参格式有误")
	}
	room, err := r.roomRepo.GetRoomById(roomId)
	if err != nil || room == nil {
		return errors.New("未查询到房间信息")
	}

	if room.Status == targetStatus {
		return errors.New("房间状态与目标状态一致")
	}

	if targetStatus == 1 {
		return r.EnableRoom(room)
	}

	if targetStatus == 0 {
		return r.DisableRoom(room)
	}

	return nil
}

func (r *RoomService) EnableRoom(room *model.Room) error {
	if room == nil {
		return errors.New("room 为空")
	}

	// 更新数据库状态字段为开启
	err := r.roomRepo.UpdateRoomExceptNil(&model.Room{
		ID:     room.ID,
		Status: 1,
	})
	if err != nil {
		return err
	}

	// tlxTODO: 开启 manager、monitor？

	return nil
}

func (r *RoomService) DisableRoom(room *model.Room) error {
	if room == nil {
		return errors.New("room 为空")
	}

	// 更新数据库状态字段为禁用
	err := r.roomRepo.UpdateRoomById(room.ID, map[string]any{
		"status": 0,
	})
	if err != nil {
		return err
	}

	// tlxTODO: 停止 manager、monitor？

	return nil
}
