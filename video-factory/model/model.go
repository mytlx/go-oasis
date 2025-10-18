package model

import "time"

type RoomItem struct {
	RoomId          string    `json:"room_id"`
	Url             string    `json:"url"`
	LastRefreshTime time.Time `json:"last_refresh_time"`
}
