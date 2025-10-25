package bili

import "encoding/json"

// --- JSON 结构体定义 (对应 B站 API 返回) ---

// ApiResponse API 顶层的 JSON 结构 (通用结构)
type ApiResponse struct {
	Code int             `json:"code"` // 0 表示成功
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"` // 使用 RawMessage 延迟解析
}

// RoomInitData 对应 room_init 接口的数据部分
//
//	{
//	   "code": 0,
//	   "msg": "ok",
//	   "message": "ok",
//	   "data": {
//	       "room_id": 22109408,
//	       "short_id": 0,
//	       "uid": 110854973,
//	       "need_p2p": 0,
//	       "is_hidden": false,
//	       "is_locked": false,
//	       "is_portrait": false,
//	       "live_status": 1,
//	       "hidden_till": 0,
//	       "lock_till": 0,
//	       "encrypted": false,
//	       "pwd_verified": false,
//	       "live_time": 1759667492,
//	       "room_shield": 0,
//	       "is_sp": 0,
//	       "special_type": 0
//	   }
//	}
type RoomInitData struct {
	RoomId     int `json:"room_id"`     // 真实房间号 (Long ID)
	LiveStatus int `json:"live_status"` // 1: 正在直播, 0: 未直播
}

// {
//    "code": 0,
//    "message": "0",
//    "ttl": 1,
//    "data": {
//        "room_id": 22109308,
//        "playurl_info": {
//            "conf_json": "{\"cdn_rate\":10000,\"report_interval_sec\":150}",
//            "playurl": {
//                "stream": [
//                    {
//                        "protocol_name": "http_hls",
//                        "format": [
//                            {
//                                "format_name": "ts",
//                                "codec": [
//                                    {
//                                        "codec_name": "avc",
//                                        "current_qn": 10000,
//                                        "accept_qn": [
//                                            10000
//                                        ],
//                                        "base_url": "/live-bvc/761794/live_110854973_88571715.m3u8?",
//                                        "url_info": [
//                                            {
//                                                "host": "https://d1--cn-gotcha104.bilivideo.com",
//                                                "extra": "expires=1759675460&len=0&oi=610358125&pt=h5&qn=10000&trid=100352c7e3f0c7456fab21f67c481e68e276&bmt=1&sigparams=cdn,expires,len,oi,pt,qn,trid,bmt&cdn=cn-gotcha104&sign=181621af99ea3a7891ce74dded16c626&bili=ee28318951093942aaf011977733059c&free_type=0&mid=0&sche=ban&trace=16&isp=ct&rg=NorthEast&pv=Jilin&sk=349fc313d24b1840418f3121d3d2b58c&info_source=origin&hdr_type=0&codec=0&pp=srt&origin_bitrate=4301&score=1&p2p_type=-1&sl=1&source=puv3_onetier&suffix=origin&deploy_env=prod&hot_cdn=0&media_type=0&vd=bc&src=puv3&order=1",
//                                                "stream_ttl": 0
//                                            },
//                                            {
//                                                "host": "https://d1--cn-gotcha104b.bilivideo.com",
//                                                "extra": "expires=1759675460&len=0&oi=610358125&pt=h5&qn=10000&trid=100352c7e3f0c7456fab21f67c481e68e276&bmt=1&sigparams=cdn,expires,len,oi,pt,qn,trid,bmt&cdn=cn-gotcha104&sign=181621af99ea3a7891ce74dded16c626&bili=ee28318951093942aaf011977733059c&free_type=0&mid=0&sche=ban&trace=16&isp=ct&rg=NorthEast&pv=Jilin&sk=349fc313d24b1840418f3121d3d2b58c&info_source=origin&hdr_type=0&codec=0&pp=srt&origin_bitrate=4301&score=1&p2p_type=-1&sl=1&source=puv3_onetier&suffix=origin&deploy_env=prod&hot_cdn=0&media_type=0&vd=bc&src=puv3&order=2",
//                                                "stream_ttl": 0
//                                            }
//                                        ],
//                                    }
//                                ],
//                                "master_url": ""
//                            },
//                        ]
//                    }
//                ],
//            },
//        }
//    }
// }

// PlayInfoData 对应 getRoomPlayInfo 接口的数据部分
type PlayInfoData struct {
	RoomId      int         `json:"room_id"`     // 真实房间号 (Long ID)
	LiveStatus  int         `json:"live_status"` // 1: 正在直播, 0: 未直播
	PlayURLInfo PlayURLInfo `json:"playurl_info"`
}

// PlayURLInfo 包含 playurl 信息
type PlayURLInfo struct {
	PlayURL PlayURL `json:"playurl"`
}

// PlayURL 对应 playurl_info 中的 playurl
type PlayURL struct {
	Stream []StreamData `json:"stream"`
}

// StreamData 对应 playurl_info 中的 stream
type StreamData struct {
	Format []StreamFormat `json:"format"`
}

// StreamFormat 对应 playurl_info 中的 format
type StreamFormat struct {
	FormatName string        `json:"format_name"` // ts flv fmp4
	Codec      []StreamCodec `json:"codec"`
}

// StreamCodec 对应 playurl_info 中的 codec
type StreamCodec struct {
	BaseURL   string    `json:"base_url"`
	URLInfo   []URLInfo `json:"url_info"`
	AcceptQn  []int     `json:"accept_qn"`  // 可接受的清晰度列表
	CurrentQn int       `json:"current_qn"` // 当前清晰度
}

// URLInfo 对应 playurl_info 中的 url_info
type URLInfo struct {
	Host  string `json:"host"`
	Extra string `json:"extra"`
}
