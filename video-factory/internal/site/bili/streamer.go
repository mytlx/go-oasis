package bili

import (
	"errors"
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

	"github.com/rs/zerolog/log"
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
)

type Streamer struct {
	RealRoomId string
	Platform   string // 平台
	RoomUrl    string // 直播间 URL
	LiveStatus int    // 直播间状态 0:未开播 1:直播中
	Header     http.Header
	StreamInfo *iface.StreamInfo
}

func NewStreamer(realId string, config *config.AppConfig) *Streamer {
	s := &Streamer{
		RealRoomId: realId,
		Platform:   consts.PlatformBili,
		Header:     make(http.Header),
		StreamInfo: &iface.StreamInfo{
			StreamUrls: map[string]string{},
			SelectedQn: defaultQn,
		},
	}
	// 设置 Header
	s.Header.Set("User-Agent", userAgent)
	// tlxTODO: bili referer
	s.Header.Set("Referer", "https://live.bilibili.com")
	cookie := strings.TrimSpace(config.Bili.Cookie)
	if cookie != "" {
		s.Header.Set("Cookie", cookie)
	}

	return s
}

func (s *Streamer) OnConfigUpdate(key string, value string) {
	log.Info().Msgf("[bili] 配置更新: %s=%s", key, value)
	if key == "bili.cookie" {
		s.Header.Set("Cookie", value)
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (s *Streamer) GetHeaders() http.Header {
	// TODO implement me
	panic("implement me")
}

func (s *Streamer) IsLive() (bool, error) {
	data, err := s.getRoomInfo()
	if err != nil {
		return false, err
	}

	if data.LiveStatus != 1 {
		s.LiveStatus = 0
		return false, nil
	}

	s.LiveStatus = 1
	return true, nil
}

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
			s.StreamInfo.AcceptQns = acceptQn
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

				s.StreamInfo.SelectedQn = codec.CurrentQn
				s.StreamInfo.ActualQn = codec.CurrentQn
				log.Info().Msgf("请求清晰度：%d, 实际清晰度：%d", currentQn, codec.CurrentQn)

				// 遍历所有 url_info (即线路)
				for i, info := range codec.URLInfo {
					fullURL := fmt.Sprintf("%s%s%s", info.Host, baseHost, info.Extra)
					s.StreamInfo.StreamUrls[fmt.Sprintf("线路%d", i+1)] = fullURL
				}

				// 只获取第一个 ts 格式的流信息 (通常足够)
				return s.StreamInfo, nil
			}
		}
	}

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

// ---------------------------------------------------------------------------------------------------------------------

// getRoomInfo 获取房间状态
func (s *Streamer) getRoomInfo() (*RoomInitData, error) {
	data, err := FetchRoomInitInfo(s.RealRoomId, s.Header)
	if err != nil {
		return nil, err
	}
	s.RealRoomId = strconv.Itoa(data.RoomId)
	return data, nil
}

// getPlayInfo 获取播放信息
func (s *Streamer) getPlayInfo(qn int) (*PlayInfoData, error) {
	data, err := FetchPlayInfo(s.RealRoomId, qn, s.Header)
	if err != nil {
		return nil, err
	}
	if data.LiveStatus != 1 {
		s.LiveStatus = 0
		return nil, iface.ErrRoomOffline
	}
	return data, nil
}

// CheckAndGetRid 检查并获取rid
func CheckAndGetRid(s string) (string, error) {
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
		return CheckAndGetRid(longUrl)
	}

	log.Error().Msgf("格式有误，获取rid失败: %s", s)
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

// ---------------------------------------------------------------------------------------------------------------------

func GetRoomLiveStatus(rid string) (int, error) {
	data, err := FetchRoomInfo(rid)
	if err != nil {
		return 0, err
	}

	if data.LiveStatus != 1 {
		return 0, nil
	}

	return 1, nil
}

func GetRoomAddInfo(roomIdStr string) (*vo.RoomAddVO, error) {
	data, err := FetchRoomInfo(roomIdStr)
	if err != nil {
		return nil, err
	}

	info, err := FetchAnchorInfo(strconv.Itoa(data.Uid))
	if err != nil {
		return nil, err
	}

	return &vo.RoomAddVO{
		Platform:     consts.PlatformBili,
		ShortID:      strconv.Itoa(data.ShortId),
		RealID:       strconv.Itoa(data.RoomId),
		Name:         data.Title,
		URL:          fmt.Sprintf("https://live.bilibili.com/%s", roomIdStr),
		CoverURL:     data.UserCover,
		AnchorID:     strconv.Itoa(info.Uid),
		AnchorName:   info.Uname,
		AnchorAvatar: info.Face,
		// ProxyURL: fmt.Sprintf("http://localhost:%d/api/v1/%s/proxy/%s/index.m3u8", config.Port, baseURLPrefix, rid)
	}, nil
}
