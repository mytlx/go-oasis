package service

import (
	"video-factory/dao"
	"video-factory/manager"
	"video-factory/model"
	"video-factory/pool"
	"video-factory/utils"
	"video-factory/vo"
)

func AddOrUpdateRoom(m *manager.Manager) error {

	m.Mutex.RLock()
	info := m.Streamer.GetInfo()
	m.Mutex.RUnlock()

	room := &model.Room{
		ID:       info.Rid,
		Platform: info.Platform,
		RealID:   info.RealRoomId,
		Name:     "",
		URL:      "",
		ProxyURL: m.ProxyURL,
	}

	return dao.AddRoom(room)
}

func ListRooms(pool *pool.ManagerPool) ([]vo.RoomVO, error) {
	rooms, err := dao.ListRooms()
	if err != nil {
		return nil, err
	}

	poolSnapshot := pool.Snapshot()
	respList := make([]vo.RoomVO, len(rooms))
	for i, room := range rooms {
		status := 0
		if poolSnapshot[room.ID] != nil {
			status = 1
		}
		respList[i] = vo.RoomVO{
			ID:         room.ID,
			RealID:     room.RealID,
			Name:       room.Name,
			Status:     status,
			ProxyURL:   room.ProxyURL,
			URL:        room.URL,
			Platform:   room.Platform,
			CreateTime: utils.MillisToTime(room.CreateTime),
			UpdateTime: utils.MillisToTime(room.UpdateTime),
		}
	}
	return respList, nil
}
