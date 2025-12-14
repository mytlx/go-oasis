package bili

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"video-factory/pkg/fetcher"

	"github.com/rs/zerolog/log"
)

// FetchRoomInfo 获取直播间信息
func FetchRoomInfo(roomId string) (*RoomInfoData, error) {
	apiURL := "https://api.live.bilibili.com/room/v1/Room/get_info"

	params := url.Values{}
	params.Set("room_id", roomId)

	response, err := Fetch(apiURL, params, nil)
	if err != nil {
		return nil, err
	}

	// 解析 Data 部分
	var data RoomInfoData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		log.Err(err).Msgf("RoomInfoData 解析失败, response.Data: %s", response.Data)
		return nil, fmt.Errorf("RoomInfoData 解析失败: %v", err)
	}

	return &data, nil
}

// FetchRoomInitInfo 获取房间页初始化信息
func FetchRoomInitInfo(rid string, header http.Header) (*RoomInitData, error) {
	apiURL := "https://api.live.bilibili.com/room/v1/Room/room_init"

	params := url.Values{}
	params.Set("id", rid)

	response, err := Fetch(apiURL, params, header)
	if err != nil {
		return nil, err
	}

	// 解析 Data 部分
	var data RoomInitData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		log.Err(err).Msgf("room_init Data 解析失败, response.Data: %s", response.Data)
		return nil, fmt.Errorf("room_init Data 解析失败: %v", err)
	}

	return &data, nil
}

func FetchAnchorInfo(uid string) (*AnchorInfo, error) {
	apiURL := "https://api.live.bilibili.com/live_user/v1/Master/info"

	params := url.Values{}
	params.Set("uid", uid)

	response, err := Fetch(apiURL, params, nil)
	if err != nil {
		return nil, err
	}

	// 解析 Data 部分
	var data AnchorInfoData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		log.Err(err).Msgf("AnchorInfoData 解析失败, response.Data: %s", response.Data)
		return nil, fmt.Errorf("AnchorInfoData 解析失败: %v", err)
	}

	return &data.Info, nil
}

// FetchPlayInfo 获取直播间播放信息
func FetchPlayInfo(roomId string, qn int, header http.Header) (*PlayInfoData, error) {
	apiURL := "https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo"

	params := url.Values{}
	params.Set("room_id", roomId)
	params.Set("protocol", "0,1")      // 0：http_stream; 1：http_hls; 可多选, 使用英文逗号分隔
	params.Set("format", "0,1,2")      // 0：flv; 1：ts; 2：fmp4; 可多选, 使用英文逗号分隔
	params.Set("codec", "0,1")         // 0：AVC; 1：HEVC; 可多选, 使用英文逗号分隔
	params.Set("qn", strconv.Itoa(qn)) // 默认150
	params.Set("platform", "html5")
	params.Set("ptype", "8")
	params.Set("dolby", "5")

	response, err := Fetch(apiURL, params, header)
	if err != nil {
		return nil, err
	}

	var data PlayInfoData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		return nil, fmt.Errorf("PlayInfoData 解析失败: %v", err)
	}

	return &data, nil
}

// =====================================================================================================================

func Fetch(baseURL string, params url.Values, header http.Header) (*ApiResponse, error) {
	if header == nil {
		header = make(http.Header)
		header.Set("User-Agent", userAgent)
	}

	// 发送请求并获取 JSON 响应
	body, err := fetcher.FetchBody(baseURL, params, header)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %v", err)
	}

	// 解析响应
	var response ApiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Err(err).Msg("API 响应 JSON 解析失败")
		return nil, fmt.Errorf("JSON 解析失败: %v", err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("bili API 错误 (%d): %s", response.Code, response.Msg)
	}

	return &response, nil
}
