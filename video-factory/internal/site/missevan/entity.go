package missevan

import "encoding/json"

type ApiResponse struct {
	Code int             `json:"code"`
	Info json.RawMessage `json:"info"`
}

type Info struct {
	Room    Room    `json:"room"`
	Creator Creator `json:"creator"`
}

type Room struct {
	RoomId   int64  `json:"room_id"`
	Name     string `json:"name"`
	CoverUrl string `json:"cover_url"`
	Channel  struct {
		FlvPullUrl string `json:"flv_pull_url"`
		HlsPullUrl string `json:"hls_pull_url"`
	} `json:"channel"`
	Status struct {
		Open int `json:"open"` // 0: 未开播 1: 正在直播
	} `json:"status"`
}

type Creator struct {
	UserId   int64  `json:"user_id"`
	Username string `json:"username"`
	IconUrl  string `json:"iconurl"`
}
