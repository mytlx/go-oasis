package service

import (
	"github.com/rs/zerolog/log"
	"sync"
	"video-factory/internal/domain/model"
	"video-factory/internal/domain/vo"
	"video-factory/internal/repository"
	"video-factory/internal/site/bili"
	"video-factory/internal/site/missevan"
	"video-factory/pkg/pool"
	"video-factory/pkg/util"
)

// func AddOrUpdateRoom(m *manager.Manager) error {
//
// 	m.Mutex.RLock()
// 	info := m.Streamer.GetInfo()
// 	m.Mutex.RUnlock()
//
// 	room := &model.Room{
// 		ID:         info.Rid,
// 		Platform:   info.Platform,
// 		RealID:     info.RealRoomId,
// 		Name:       "",
// 		URL:        info.RoomUrl,
// 		ProxyURL:   m.ProxyURL,
// 		CreateTime: time.Now().UnixMilli(),
// 		UpdateTime: time.Now().UnixMilli(),
// 	}
//
// 	return repository.AddOrUpdateRoom(room)
// }

func ListRooms(pool *pool.ManagerPool) ([]vo.RoomVO, error) {
	rooms, err := repository.ListRooms()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var statusMutex sync.Mutex
	statusMap := make(map[string]int, len(rooms))

	for _, room := range rooms {
		wg.Add(1) // 增加计数器
		go func(room *model.Room) {
			defer wg.Done()
			// 调用接口获取实时状态
			status, statusErr := GetRoomLiveStatus(room)
			if statusErr != nil {
				log.Warn().Msgf("警告: 获取房间[%s]直播状态失败: %v\n", room.ID, statusErr)
				// 失败默认离线
				status = 0
			}
			statusMutex.Lock()
			statusMap[room.ID] = status
			statusMutex.Unlock()
		}(&room)
	}
	wg.Wait()

	poolSnapshot := pool.Snapshot()
	respList := make([]vo.RoomVO, len(rooms))
	for i, room := range rooms {
		status := 0
		m := poolSnapshot[room.ID]
		lastRefreshTime := util.MillisToTime(room.UpdateTime)
		if m != nil {
			status = 1
			lastRefreshTime = m.GetLastRefreshTime()
		}
		liveStatus, ok := statusMap[room.ID]
		if !ok {
			liveStatus = 0
		}
		respList[i] = vo.RoomVO{
			ID:              room.ID,
			RealID:          room.RealID,
			Name:            room.Name,
			Status:          status,
			LiveStatus:      liveStatus,
			ProxyURL:        room.ProxyURL,
			URL:             room.URL,
			Platform:        room.Platform,
			LastRefreshTime: lastRefreshTime,
			CreateTime:      util.MillisToTime(room.CreateTime),
			UpdateTime:      util.MillisToTime(room.UpdateTime),
		}
	}
	return respList, nil
}

func RemoveRoom(rid string) error {
	return repository.RemoveRoom(rid)
}

func GetRoom(rid string) (*model.Room, error) {
	return repository.GetRoomById(rid)
}

func GetRoomLiveStatus(room *model.Room) (int, error) {
	if room == nil {
		return 0, nil
	}
	switch room.Platform {
	case "bili":
		return bili.GetRoomLiveStatus(room.ID)
	case "missevan":
		return missevan.GetRoomLiveStatus(room.ID)
	default:
		return 0, nil
	}
}