package missevan

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"video-factory/internal/common/consts"
	"video-factory/internal/domain/vo"
	"video-factory/internal/iface"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"

	"github.com/rs/zerolog/log"
)

const (
	getLiveBaseUrl = "https://fm.missevan.com/api/v2/live/"
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	refererPrefix  = "https://fm.missevan.com/live/"
	origin         = "https://fm.missevan.com"
)

type Streamer struct {
	RealRoomId string
	Platform   string // 平台
	RoomUrl    string // 直播间 URL
	LiveStatus int    // 直播间状态 0:未开播 1:直播中
	OpenTime   int64  // 开播时间
	Header     http.Header
	StreamInfo *iface.StreamInfo
}

func (s *Streamer) GetHeaders() http.Header {
	// func (HandlerStrategy) GetExtraHeaders() http.Header {
	// 	// 猫耳需要特定的 Host
	// 	header := make(http.Header)
	// 	header.Set("Host", "d1-missevan04.bilivideo.com")
	// 	return header
	// }

	// func (m *Manager) Fetch(ctx context.Context, baseURL string, params url.Values, extraHeader http.Header) (*http.Response, error) {
	// 	executor := func(method, url string, p url.Values) (*http.Response, error) {
	// 		// 猫耳需要补充 host
	// 		baseHeader := m.Manager.Streamer.GetInfo().Header
	// 		requestHeader := baseHeader.Clone()
	// 		for k, vv := range extraHeader {
	// 			requestHeader.Del(k)
	// 			for _, v := range vv {
	// 				requestHeader.Add(k, v)
	// 			}
	// 		}
	// 		return fetcher.Fetch(method, url, p, requestHeader)
	// 	}
	// 	return fetcher.FetchWithRefresh(ctx, m, executor, "GET", baseURL, params)
	// }
	// tlxTODO:
	return nil
}

func NewStreamer(realRoomId string, config *config.AppConfig) *Streamer {
	s := &Streamer{
		RealRoomId: realRoomId,
		Platform:   consts.PlatformMissevan,
		Header:     make(http.Header),
		StreamInfo: &iface.StreamInfo{
			StreamUrls: map[string]string{},
		},
	}
	// 设置 Header
	s.Header.Set("User-Agent", userAgent)
	s.Header.Set("Referer", refererPrefix+realRoomId)
	s.Header.Set("Origin", origin)
	s.Header.Set("Accept-Encoding", "identity")
	cookie := strings.TrimSpace(config.Bili.Cookie)
	if cookie != "" {
		s.Header.Set("Cookie", cookie)
	}

	return s
}

func (s *Streamer) OnConfigUpdate(key string, value string) {
	log.Info().Msgf("[missevan] 配置更新: %s=%s", key, value)
	if key == "missevan.cookie" {
		s.Header.Set("Cookie", value)
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (s *Streamer) IsLive() (bool, error) {
	room, _, err := FetchRoomInfo(s.RealRoomId, s.Header)
	if err != nil {
		return false, err
	}

	if room.Status.Open == 0 {
		s.LiveStatus = 0
		log.Error().Msgf("房间[%s]未开播", s.RealRoomId)
		return false, nil
	}

	s.LiveStatus = 1
	return true, nil
}

func (s *Streamer) FetchStreamInfo(currentQn int, certainQnFlag bool) (*iface.StreamInfo, error) {
	room, _, err := FetchRoomInfo(s.RealRoomId, s.Header)
	if err != nil {
		return nil, err
	}

	if room.Status.Open == 0 {
		log.Error().Msgf("房间[%d]未开播", room.RoomId)
		return nil, iface.ErrRoomOffline
	}

	// s.info.StreamInfo.StreamUrls["flv"] = room.Channel.FlvPullUrl
	s.StreamInfo.StreamUrls["hls"] = room.Channel.HlsPullUrl

	return s.StreamInfo, nil
}

func (s *Streamer) GetStreamInfo() iface.StreamInfo {
	return *s.StreamInfo
}

func (s *Streamer) ParseExpiration(streamUrl string) (time.Time, error) {
	parsedUrl, err := url.Parse(streamUrl)
	if err != nil {
		log.Err(err).Msg("解析 HLS URL 失败")
		return time.Now(), err
	}

	expiresStr := parsedUrl.Query().Get("expires")

	// 1. 将字符串转换为 int64
	expiresInt, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		fmt.Println("转换时间戳字符串为整数失败:", err)
		return time.Now(), err
	}

	// 2. 使用 time.Unix() 转换为 time.Time 类型
	// 第一个参数是秒 (sec)，第二个参数是纳秒 (nsec)，这里设为 0
	return time.Unix(expiresInt, 0), nil
}

func (s *Streamer) GetOpenTime() int64 {
	room, _, err := FetchRoomInfo(s.RealRoomId, nil)
	if err != nil {
		return 0
	}
	s.OpenTime = room.Status.OpenTime
	return s.OpenTime
}

// ---------------------------------------------------------------------------------------------------------------------

func CheckAndGetRid(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("入参为空")
	}

	// 纯数字
	if ok, _ := regexp.MatchString(`^\d+$`, s); ok {
		return s, nil
	}

	// 长链接匹配
	reLong := regexp.MustCompile(`(?:https?://)?fm\.missevan\.com/live/(\d+)`)
	if matches := reLong.FindStringSubmatch(s); len(matches) >= 2 {
		return matches[1], nil
	}

	log.Error().Msgf("格式有误，获取rid失败: %s", s)
	return "", fmt.Errorf("格式有误，获取rid失败: %s", s)
}

func FetchRoomInfo(realId string, header http.Header) (*Room, *Creator, error) {
	if header == nil {
		header = make(http.Header)
		header.Set("User-Agent", userAgent)
		header.Set("Referer", refererPrefix+realId)
		header.Set("Origin", origin)
		header.Set("Accept-Encoding", "identity")
	}
	resp, err := fetcher.FetchBody(getLiveBaseUrl+realId, nil, header)
	if err != nil {
		return nil, nil, err
	}

	var response ApiResponse
	if err := json.Unmarshal(resp, &response); err != nil {
		return nil, nil, fmt.Errorf("JSON结构解析失败: %v", err)
	}
	if response.Code != 0 {
		return nil, nil, fmt.Errorf("API业务错误 (%d)", response.Code)
	}

	var info Info
	if err := json.Unmarshal(response.Info, &info); err != nil {
		return nil, nil, fmt.Errorf("JSON结构解析失败: %v", err)
	}

	return &info.Room, &info.Creator, nil
}

func GetRoomLiveStatus(rid string) (int, error) {
	room, _, err := FetchRoomInfo(rid, nil)
	if err != nil {
		return 0, err
	}

	if room.Status.Open == 0 {
		return 0, nil
	}

	return 1, nil
}

func GetRoomAddInfo(roomIdStr string) (*vo.RoomAddVO, error) {
	info, creator, err := FetchRoomInfo(roomIdStr, nil)
	if err != nil {
		return nil, err
	}

	return &vo.RoomAddVO{
		Platform:     consts.PlatformMissevan,
		ShortID:      "",
		RealID:       strconv.FormatInt(info.RoomId, 10),
		Name:         info.Name,
		URL:          fmt.Sprintf("https://fm.missevan.com/live/%d", info.RoomId),
		CoverURL:     info.CoverUrl,
		AnchorID:     strconv.FormatInt(creator.UserId, 10),
		AnchorName:   creator.Username,
		AnchorAvatar: creator.IconUrl,
		// ProxyURL: fmt.Sprintf("http://localhost:%d/api/v1/%s/proxy/%s/index.m3u8", config.Port, baseURLPrefix, rid)
	}, nil
}
