package bili

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"video-factory/internal/iface"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"
)

/*
qn: 分辨率
qn=80    流畅
qn=150   高清
qn=250   超清
qn=400   蓝光
qn=10000 原画
qn=15000 2K
qn=20000 4K
qn=30000 杜比
*/
const (
	// 默认分辨率
	defaultQn = 10000
	// 模拟手机浏览器 (H5/App 接口通常比 Web 接口稳定)
	userAgent = "Mozilla/5.0 (iPod; CPU iPhone OS 14_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.163 Mobile/15E148 Safari/604.1"
	// 获取真实房间号和状态 API
	roomStatusApi      = "https://api.live.bilibili.com/room/v1/Room/room_init"
	getRoomPlayInfoApi = "https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo"
)

type Streamer struct {
	info *iface.Info
}

func NewStreamer(rid string, config *config.AppConfig) *Streamer {
	s := &Streamer{
		info: &iface.Info{
			Header: make(http.Header),
			Rid:    rid,
			StreamInfo: &iface.StreamInfo{
				StreamUrls: map[string]string{},
			},
			Platform: baseURLPrefix,
		},
	}
	// 设置 Header
	s.info.Header.Set("User-Agent", userAgent)
	// tlxTODO: bili referer
	s.info.Header.Set("Referer", "https://live.bilibili.com")
	cookie := strings.TrimSpace(config.Bili.Cookie)
	if cookie != "" {
		s.info.Header.Set("Cookie", cookie)
	}

	return s
}

func (s *Streamer) OnConfigUpdate(key string, value string) {
	log.Info().Msgf("[bili] 配置更新: %s=%s", key, value)
	if key == "bili.cookie" {
		s.info.Header.Set("Cookie", value)
	}
}

// InitRoom 初始化房间，获取真实房间号、直播状态
func (s *Streamer) InitRoom() error {
	rid, err := checkAndGetRid(s.info.Rid)
	if err != nil {
		return err
	}
	s.info.Rid = rid
	data, err := s.getRoomInfo()
	if err != nil {
		return err
	}
	if data.LiveStatus != 1 {
		return fmt.Errorf("房间[%s]未开播 (LiveStatus: %d)", s.info.Rid, data.LiveStatus)
	}
	s.info.RealRoomId = strconv.Itoa(data.RoomId)
	s.info.RoomUrl = fmt.Sprintf("https://live.bilibili.com/%s", s.info.RealRoomId)
	log.Info().Msgf("房间[%s]初始化成功，真实房间号: %s", s.info.Rid, s.info.RealRoomId)
	return nil
}

// GetId 返回直播源的唯一标识符
func (s *Streamer) GetId() (string, error) {
	var err error
	if s.info.Rid == "" {
		_, err = s.getRoomInfo()
	}

	return s.info.Rid, err
}

// IsLive 检查直播间是否在直播中
func (s *Streamer) IsLive() (bool, error) {
	data, err := s.getRoomInfo()
	if err != nil {
		return false, err
	}

	if data.LiveStatus != 1 {
		s.info.LiveStatus = 0
		return false, nil
	}

	s.info.LiveStatus = 1
	return true, nil
}

// FetchStreamInfo 获取直播流信息
func (s *Streamer) FetchStreamInfo(currentQn int, certainQnFlag bool) (*iface.StreamInfo, error) {
	if currentQn < 0 {
		log.Warn().Msgf("清晰度参数错误: %d", currentQn)
		currentQn = defaultQn
		log.Warn().Msgf("使用默认清晰度: %d", currentQn)
	}
	if currentQn == 0 {
		currentQn = defaultQn
	}

	// 第一次尝试获取指定清晰度
	data, err := s.getPlayInfo(currentQn)
	if err != nil {
		return nil, err
	}

	// --- 清晰度协商逻辑 ---
	qnMax, currentFlag := 0, false // 清晰度最大值，当前清晰度是否可用
	// 找到所有流中最大的可接受清晰度
	if len(data.PlayURLInfo.PlayURL.Stream) > 0 {
		// 理论上只需要检查第一个流的第一个格式的第一个编码
		stream := data.PlayURLInfo.PlayURL.Stream[0]
		if len(stream.Format) > 0 && len(stream.Format[0].Codec) > 0 {
			acceptQn := stream.Format[0].Codec[0].AcceptQn
			s.info.StreamInfo.AcceptQns = acceptQn
			for _, qn := range acceptQn {
				if qn > qnMax {
					qnMax = qn
				}
				if qn == currentQn {
					currentFlag = true
				}
			}
		}
	}

	log.Info().Msgf("最大可用清晰度: %d", qnMax)

	// 重新请求最高清晰度: 1) 请求的清晰度不可用 2) 有更高清晰度，并且要求确切清晰度
	if !currentFlag || (!certainQnFlag && qnMax > currentQn) {
		log.Info().Msgf("请求清晰度[%d]不可用或有更高清晰度，重新请求最高清晰度[%d]...", currentQn, qnMax)
		data, err = s.getPlayInfo(qnMax)
		if err != nil {
			return nil, err
		}
	}

	// --- 提取 HLS 地址 ---
	for _, streamData := range data.PlayURLInfo.PlayURL.Stream {
		for _, format := range streamData.Format {
			// 仅处理 HLS 格式
			if format.FormatName == "fmp4" && len(format.Codec) > 0 {
				codec := format.Codec[0]
				baseHost := codec.BaseURL

				s.info.StreamInfo.SelectedQn = codec.CurrentQn
				s.info.StreamInfo.ActualQn = codec.CurrentQn
				log.Info().Msgf("请求清晰度：%d, 实际清晰度：%d", currentQn, codec.CurrentQn)

				// 遍历所有 url_info (即线路)
				for i, info := range codec.URLInfo {
					fullURL := fmt.Sprintf("%s%s%s", info.Host, baseHost, info.Extra)
					s.info.StreamInfo.StreamUrls[fmt.Sprintf("线路%d", i+1)] = fullURL
				}

				// 只获取第一个 ts 格式的流信息 (通常足够)
				return s.info.StreamInfo, nil
			}
		}
	}

	return s.info.StreamInfo, nil
}

// GetInfo 获取成员变量副本
func (s *Streamer) GetInfo() iface.Info {
	return *s.info
}

// GetStreamInfo 获取内部成员变量副本
func (s *Streamer) GetStreamInfo() iface.StreamInfo {
	return *s.info.StreamInfo
}

// getRoomInfo 获取房间状态
func (s *Streamer) getRoomInfo() (*RoomInitData, error) {
	data, err := FetchRoomInfo(s.info.Rid, s.info.Header)
	if err != nil {
		return nil, err
	}
	s.info.RealRoomId = strconv.Itoa(data.RoomId)
	return data, nil
}

// getPlayInfo 获取播放信息
func (s *Streamer) getPlayInfo(qn int) (*PlayInfoData, error) {
	params := url.Values{}
	params.Set("room_id", s.info.RealRoomId)
	params.Set("protocol", "0,1")
	params.Set("format", "0,1,2")
	params.Set("codec", "0,1")
	params.Set("qn", strconv.Itoa(qn))
	params.Set("platform", "html5")
	params.Set("ptype", "8")
	params.Set("dolby", "5")

	body, err := fetcher.FetchBody(getRoomPlayInfoApi, params, s.info.Header)
	if err != nil {
		return nil, err
	}

	var response ApiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("getPlayInfo JSON 解析失败: %v", err)
	}
	if response.Code != 0 {
		return nil, fmt.Errorf("getPlayInfo API 业务错误 (%d): %s", response.Code, response.Msg)
	}

	var data PlayInfoData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		return nil, fmt.Errorf("getPlayInfo Data 解析失败: %v", err)
	}
	if data.LiveStatus != 1 {
		s.info.LiveStatus = 0
		return nil, fmt.Errorf("房间[%s]未开播 (LiveStatus: %d)", s.info.Rid, data.LiveStatus)
	}
	return &data, nil
}

// checkAndGetRid 检查并获取rid
func checkAndGetRid(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("入参为空")
	}

	// 纯数字
	if ok, _ := regexp.MatchString(`^\d+$`, s); ok {
		return s, nil
	}

	// 长链接匹配
	reLong := regexp.MustCompile(`(?:https?://)?live\.bili\.com/(?:h5/)?(\d+)`)
	if matches := reLong.FindStringSubmatch(s); len(matches) >= 2 {
		return matches[1], nil
	}

	// 短链接匹配
	reShort := regexp.MustCompile(`b23\.tv/[A-Za-z0-9]+`)
	if matches := reShort.FindStringSubmatch(s); len(matches) >= 1 {
		shortUrl := "https://" + matches[0]
		longUrl, err := resolveBilibiliShortURL(shortUrl)
		if err != nil {
			return "", err
		}
		return checkAndGetRid(longUrl)
	}

	return "", fmt.Errorf("格式有误，获取rid失败: %s", s)
}

// resolveBilibiliShortURL 解析哔哩哔哩短链接，返回最终的长链接。
// 支持多级跳转（例如 b23.tv -> live.bili.com/...）
func resolveBilibiliShortURL(shortURL string) (string, error) {
	if !isShortURL(shortURL) {
		return "", fmt.Errorf("%s 不是短链接", shortURL)
	}

	client := &http.Client{
		// 禁止自动跳转，保留 302 响应
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	current := shortURL
	for i := 0; i < 5; i++ { // 最多允许 5 次跳转，防止死循环
		req, err := http.NewRequest("GET", current, nil)
		if err != nil {
			return "", err
		}

		// 模拟移动端 UA，有些短链在 PC UA 下不会返回正确跳转
		req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)")

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		resp.Body.Close()

		// 如果不是 3xx，就说明已经到达最终地址
		if resp.StatusCode < 300 || resp.StatusCode >= 400 {
			return current, nil
		}

		// 取出跳转地址
		loc := resp.Header.Get("Location")
		if loc == "" {
			return "", errors.New("未找到 Location 头")
		}

		// 更新当前 URL，继续下一轮
		current = loc
	}

	return "", errors.New("跳转次数过多")
}

// isShortURL 判断是否为 b23.tv 短链接
func isShortURL(s string) bool {
	return strings.Contains(s, "b23.tv/")
}

func FetchRoomInfo(rid string, header http.Header) (*RoomInitData, error) {
	// 构造参数
	params := url.Values{}
	params.Set("id", rid)

	if header == nil {
		header = make(http.Header)
		header.Set("Referer", "https://live.bilibili.com/"+rid)
		header.Set("User-Agent", userAgent)
	}

	// 发送请求并获取 JSON 响应
	body, err := fetcher.FetchBody(roomStatusApi, params, header)
	if err != nil {
		log.Err(err).Msg("room_init 请求失败")
		return nil, err
	}

	// 解析 room_init 响应
	var response ApiResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Err(err).Msg("room_init 响应 JSON 解析失败")
		return nil, fmt.Errorf("room_init JSON 解析失败: %v", err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("bili API 错误 (%s): %s", rid, response.Msg)
	}

	// 解析 Data 部分
	var data RoomInitData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		log.Err(err).Msgf("room_init Data 解析失败, response.Data: %s", response.Data)
		return nil, fmt.Errorf("room_init Data 解析失败: %v", err)
	}

	log.Info().Msgf("获取房间[%s]信息成功，真实房间号: %d", rid, data.RoomId)

	return &data, nil
}

func GetRoomLiveStatus(rid string) (int, error) {
	data, err := FetchRoomInfo(rid, nil)
	if err != nil {
		return 0, err
	}

	if data.LiveStatus != 1 {
		return 0, nil
	}

	return 1, nil
}
