package service

import (
	"video-factory/internal/domain/model"
	"video-factory/internal/domain/vo"
	"video-factory/internal/repository"
	"video-factory/pkg/pool"
	"video-factory/pkg/util"
)

func AddOrUpdateRoom(m *Manager) error {

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

	return repository.AddRoom(room)
}

func ListRooms(pool *pool.ManagerPool) ([]vo.RoomVO, error) {
	rooms, err := repository.ListRooms()
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
			CreateTime: util.MillisToTime(room.CreateTime),
			UpdateTime: util.MillisToTime(room.UpdateTime),
		}
	}
	return respList, nil
}
