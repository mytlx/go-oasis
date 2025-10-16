package missevan

import "encoding/json"

type MissevanResponse struct {
	Code int             `json:"code"`
	Info json.RawMessage `json:"info"`
}

type Info struct {
	Room Room `json:"room"`
}

type Room struct {
	RoomId  int64 `json:"room_id"`
	Channel struct {
		FlvPullUrl string `json:"flv_pull_url"`
		HlsPullUrl string `json:"hls_pull_url"`
	} `json:"channel"`
	Status struct {
		Open int `json:"open"` // 0: 未开播 1: 正在直播
	} `json:"status"`
}
