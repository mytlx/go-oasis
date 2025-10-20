package missevan

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"video-factory/config"
	"video-factory/fetcher"
	"video-factory/streamer"
)

const (
	getLiveBaseUrl = "https://fm.missevan.com/api/v2/live/"
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	refererPrefix  = "https://fm.missevan.com/live/"
)

type Streamer struct {
	info *streamer.Info
}

func NewStreamer(rid string, config *config.AppConfig) *Streamer {
	s := &Streamer{
		info: &streamer.Info{
			Header:     make(http.Header),
			Rid:        rid,
			RealRoomId: rid,
			StreamInfo: &streamer.StreamInfo{
				StreamUrls: map[string]string{},
			},
			Platform: baseURLPrefix,
		},
	}
	// 设置 Header
	s.info.Header.Set("User-Agent", userAgent)
	s.info.Header.Set("Referer", refererPrefix+rid)
	s.info.Header.Set("Origin", "https://fm.missevan.com")
	s.info.Header.Set("Accept-Encoding", "identity")
	cookie := strings.TrimSpace(config.Bili.Cookie)
	if cookie != "" {
		s.info.Header.Set("Cookie", cookie)
	}

	return s
}

func (s *Streamer) InitRoom() error {
	rid, err := checkAndGetRid(s.info.Rid)
	if err != nil {
		return err
	}
	s.info.Rid = rid

	room, err := s.fetchRoomInfo()
	if err != nil {
		return err
	}

	if room.Status.Open == 0 {
		s.info.LiveStatus = 0
		log.Error().Msgf("房间[%s]未开播", s.info.Rid)
		return fmt.Errorf("房间[%s]未开播", s.info.Rid)
	}

	s.info.LiveStatus = 1
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
	room, err := s.fetchRoomInfo()
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

func (s *Streamer) FetchStreamInfo(currentQn int) (*streamer.StreamInfo, error) {
	room, err := s.fetchRoomInfo()
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

func (s *Streamer) GetInfo() streamer.Info {
	return *s.info
}

func (s *Streamer) GetStreamInfo() streamer.StreamInfo {
	return *s.info.StreamInfo
}

func (s *Streamer) fetchRoomInfo() (*Room, error) {
	resp, err := fetcher.FetchBody(getLiveBaseUrl+s.info.Rid, nil, s.info.Header)
	if err != nil {
		return nil, err
	}

	var response MissevanResponse
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
