package bilibili

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// BiliBili 结构体，用于存储直播间状态和客户端
type BiliBili struct {
	Client     *http.Client
	Header     http.Header
	Rid        string
	RealRoomID int
	StreamUrls map[string]string
	SelectedQn int
	ActualQn   int
}

// NewBiliBili 初始化 BiliBili 结构体并执行 room_init 检查
func NewBiliBili(rid string, cookie string) (*BiliBili, error) {
	bili := &BiliBili{
		Client:     &http.Client{Timeout: 15 * time.Second},
		Header:     make(http.Header),
		Rid:        rid,
		StreamUrls: make(map[string]string),
		SelectedQn: defaultQn,
	}

	// 设置 Header
	bili.Header.Set("User-Agent", userAgent)
	bili.Header.Set("Referer", "https://live.bilibili.com")
	if cookie != "" {
		bili.Header.Set("Cookie", cookie)
	}

	// 构造参数
	params := url.Values{}
	params.Set("id", rid)

	// 发送请求并获取 JSON 响应
	body, err := bili.fetchAPI(roomStatusApi, params)
	if err != nil {
		log.Printf("room_init 请求失败: %v", err)
		return nil, err
	}

	// 解析 room_init 响应
	var response BiliAPIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("room_init 响应 JSON 解析失败: %v", err)
		return nil, fmt.Errorf("room_init JSON 解析失败: %v", err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("bilibili API 错误 (%s): %s", rid, response.Msg)
	}

	// 解析 Data 部分
	var data RoomInitData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		return nil, fmt.Errorf("room_init Data 解析失败: %v", err)
	}

	if data.LiveStatus != 1 {
		return nil, fmt.Errorf("bilibili %s 未开播 (LiveStatus: %d)", rid, data.LiveStatus)
	}

	bili.RealRoomID = data.RoomId
	log.Printf("房间[%s]初始化成功，真实房间号: %d", rid, data.RoomId)

	return bili, nil
}

// Fetch 用于统一处理 API 请求、Header设置
func (bili *BiliBili) Fetch(baseURL string, params url.Values) (*http.Response, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("解析 baseURL 失败: %v", err)
	}

	// 获取现有查询参数（如果存在）
	query := parsedURL.Query()
	// 将新参数合并到现有查询参数中
	if len(params) > 0 {
		for key, values := range params {
			// 使用 Add 而非 Set，确保参数不会覆盖已有的同名参数
			for _, value := range values {
				query.Add(key, value)
			}
		}
	}
	// 将编码后的查询参数重新设置回 URL
	parsedURL.RawQuery = query.Encode()

	request, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置 Header
	request.Header = bili.Header

	return bili.Client.Do(request)
}

// fetchAPI 用于统一处理 API 请求、Header 设置和基本错误检查
func (bili *BiliBili) fetchAPI(baseURL string, params url.Values) ([]byte, error) {
	response, err := bili.Fetch(baseURL, params)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %v", err)
	}
	defer response.Body.Close()

	// 200 304 认为成功
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotModified {
		return nil, fmt.Errorf("API 返回错误状态码: %d", response.StatusCode)
	}

	bodyBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", readErr)
	}

	return bodyBytes, nil
}

// GetRealURL 获取真实 HLS 流媒体地址，并处理清晰度协商
func (bili *BiliBili) GetRealURL(currentQn int) (map[string]string, error) {
	if currentQn < 0 {
		log.Printf("清晰度参数错误: %d", currentQn)
		currentQn = defaultQn
		log.Printf("使用默认清晰度: %d", currentQn)
	}
	if currentQn == 0 {
		currentQn = defaultQn
	}

	// 内部函数，执行核心的 API 请求和解析逻辑
	getPlayInfo := func(qn int) (*PlayInfoData, error) {
		params := url.Values{}
		params.Set("room_id", strconv.Itoa(bili.RealRoomID))
		params.Set("protocol", "0,1")
		params.Set("format", "0,1,2")
		params.Set("codec", "0,1")
		params.Set("qn", strconv.Itoa(qn))
		params.Set("platform", "html5")
		params.Set("ptype", "8")
		params.Set("dolby", "5")

		body, err := bili.fetchAPI(getRoomPlayInfoApi, params)
		if err != nil {
			return nil, err
		}

		var response BiliAPIResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("playInfo JSON 结构解析失败: %v", err)
		}
		if response.Code != 0 {
			return nil, fmt.Errorf("playInfo API 业务错误 (%d): %s", response.Code, response.Msg)
		}

		var data PlayInfoData
		if err := json.Unmarshal(response.Data, &data); err != nil {
			return nil, fmt.Errorf("playInfo Data 解析失败: %v", err)
		}
		return &data, nil
	}

	// 第一次尝试获取指定清晰度
	data, err := getPlayInfo(currentQn)
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

	// 如果请求的 qn 不是最大可用 qn，则重新请求最大 qn 的数据
	if !currentFlag || (qnMax < currentQn && qnMax > 0) {
		log.Printf("请求清晰度 %d 不可用，重新请求最高清晰度 %d...", currentQn, qnMax)
		data, err = getPlayInfo(qnMax)
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

				bili.SelectedQn = currentQn
				bili.ActualQn = codec.CurrentQn
				log.Printf("请求清晰度：%d, 实际清晰度：%d", currentQn, codec.CurrentQn)

				// 遍历所有 url_info (即线路)
				for i, info := range codec.URLInfo {
					fullURL := fmt.Sprintf("%s%s%s", info.Host, baseHost, info.Extra)
					bili.StreamUrls[fmt.Sprintf("线路%d", i+1)] = fullURL
				}

				// 只获取第一个 ts 格式的流信息 (通常足够)
				return bili.StreamUrls, nil
			}
		}
	}

	return bili.StreamUrls, fmt.Errorf("未找到 ts (HLS) 格式的流媒体地址")
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
	reLong := regexp.MustCompile(`(?:https?://)?live\.bilibili\.com/(?:h5/)?(\d+)`)
	if matches := reLong.FindStringSubmatch(s); len(matches) >= 2 {
		return matches[1], nil
	}

	// 短链接匹配
	reShort := regexp.MustCompile(`b23\.tv/[A-Za-z0-9]+`)
	if matches := reShort.FindStringSubmatch(s); len(matches) >= 1 {
		shortUrl := "https://" + matches[0]
		longUrl, err := ResolveBilibiliShortURL(shortUrl)
		if err != nil {
			return "", err
		}
		return CheckAndGetRid(longUrl)
	}

	return "", fmt.Errorf("格式有误，获取rid失败: %s", s)
}

// ResolveBilibiliShortURL 解析哔哩哔哩短链接，返回最终的长链接。
// 支持多级跳转（例如 b23.tv -> live.bilibili.com/...）
func ResolveBilibiliShortURL(shortURL string) (string, error) {
	if !IsShortURL(shortURL) {
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

// IsShortURL 判断是否为 b23.tv 短链接
func IsShortURL(s string) bool {
	return strings.Contains(s, "b23.tv/")
}
