package missevan

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"video-factory/internal/iface"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"
)

const (
	getLiveBaseUrl = "https://fm.missevan.com/api/v2/live/"
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	refererPrefix  = "https://fm.missevan.com/live/"
	origin         = "https://fm.missevan.com"
)

type Streamer struct {
	info *iface.Info
}

func NewStreamer(rid string, config *config.AppConfig) *Streamer {
	s := &Streamer{
		info: &iface.Info{
			Header:     make(http.Header),
			Rid:        rid,
			RealRoomId: rid,
			StreamInfo: &iface.StreamInfo{
				StreamUrls: map[string]string{},
			},
			Platform: baseURLPrefix,
		},
	}
	// 设置 Header
	s.info.Header.Set("User-Agent", userAgent)
	s.info.Header.Set("Referer", refererPrefix+rid)
	s.info.Header.Set("Origin", origin)
	s.info.Header.Set("Accept-Encoding", "identity")
	cookie := strings.TrimSpace(config.Bili.Cookie)
	if cookie != "" {
		s.info.Header.Set("Cookie", cookie)
	}

	return s
}

func (s *Streamer) OnConfigUpdate(key string, value string) {
	if key == "missevan.cookie" {
		s.info.Header.Set("Cookie", value)
	}
}

func (s *Streamer) InitRoom() error {
	rid, err := checkAndGetRid(s.info.Rid)
	if err != nil {
		return err
	}
	s.info.Rid = rid

	room, err := FetchRoomInfo(s.info.Rid, s.info.Header)
	if err != nil {
		return err
	}

	if room.Status.Open == 0 {
		s.info.LiveStatus = 0
		log.Error().Msgf("房间[%s]未开播", s.info.Rid)
		return fmt.Errorf("房间[%s]未开播", s.info.Rid)
	}

	s.info.LiveStatus = 1
	s.info.RoomUrl = fmt.Sprintf("https://fm.missevan.com/live/%s", s.info.Rid)
	// s.info.StreamInfo.StreamUrls["flv"] = room.Channel.FlvPullUrl
	s.info.StreamInfo.StreamUrls["hls"] = room.Channel.HlsPullUrl

	return nil
}

func checkAndGetRid(rid string) (string, error) {
	// tlxTODO: 检查入参，返回正确的 rid
	return rid, nil
}

func (s *Streamer) GetId() (string, error) {
	return s.info.Rid, nil
}

func (s *Streamer) IsLive() (bool, error) {
	room, err := FetchRoomInfo(s.info.Rid, s.info.Header)
	if err != nil {
		return false, err
	}

	if room.Status.Open == 0 {
		s.info.LiveStatus = 0
		log.Error().Msgf("房间[%s]未开播", s.info.Rid)
		return false, nil
	}

	s.info.LiveStatus = 1
	return true, nil
}

func (s *Streamer) FetchStreamInfo(currentQn int, certainQnFlag bool) (*iface.StreamInfo, error) {
	room, err := FetchRoomInfo(s.info.Rid, s.info.Header)
	if err != nil {
		return nil, err
	}

	if room.Status.Open == 0 {
		log.Error().Msgf("房间[%d]未开播", room.RoomId)
		return nil, fmt.Errorf("房间[%d]未开播", room.RoomId)
	}

	// s.info.StreamInfo.StreamUrls["flv"] = room.Channel.FlvPullUrl
	s.info.StreamInfo.StreamUrls["hls"] = room.Channel.HlsPullUrl

	return s.info.StreamInfo, nil
}

func (s *Streamer) GetInfo() iface.Info {
	return *s.info
}

func (s *Streamer) GetStreamInfo() iface.StreamInfo {
	return *s.info.StreamInfo
}

func FetchRoomInfo(rid string, header http.Header) (*Room, error) {
	if header == nil {
		header = make(http.Header)
		header.Set("User-Agent", userAgent)
		header.Set("Referer", refererPrefix+rid)
		header.Set("Origin", origin)
		header.Set("Accept-Encoding", "identity")
	}
	resp, err := fetcher.FetchBody(getLiveBaseUrl+rid, nil, header)
	if err != nil {
		return nil, err
	}

	var response ApiResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, fmt.Errorf("JSON结构解析失败: %v", err)
	}
	if response.Code != 0 {
		return nil, fmt.Errorf("API业务错误 (%d)", response.Code)
	}

	var info Info
	if err := json.Unmarshal(response.Info, &info); err != nil {
		return nil, fmt.Errorf("JSON结构解析失败: %v", err)
	}

	return &info.Room, nil
}

func GetRoomLiveStatus(rid string) (int, error) {
	room, err := FetchRoomInfo(rid, nil)
	if err != nil {
		return 0, err
	}

	if room.Status.Open == 0 {
		return 0, nil
	}

	return 1, nil
}
