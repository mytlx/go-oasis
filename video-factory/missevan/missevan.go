package missevan

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	getLiveBaseUrl = "https://fm.missevan.com/api/v2/live/"
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	refererPrefix  = "https://fm.missevan.com/live/"
)

type Missevan struct {
	Client     *http.Client
	Header     http.Header
	RoomId     string
	StreamUrls map[string]string
}

func NewMissevan(roomId string, cookie string) (*Missevan, error) {
	// Charles Proxy 默认地址和端口
	proxyURL, _ := url.Parse("http://127.0.0.1:8888")

	// 关键：创建 Transport 并设置 Proxy
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		// 其他设置，如 SSL 配置等...
	}

	missevan := &Missevan{
		Client:     &http.Client{Timeout: 30 * time.Second, Transport: transport},
		Header:     make(http.Header),
		RoomId:     roomId,
		StreamUrls: make(map[string]string),
	}
	missevan.Header.Set("User-Agent", userAgent)
	missevan.Header.Set("Referer", refererPrefix+roomId)
	missevan.Header.Set("Accept-Encoding", "identity")
	if cookie != "" {
		missevan.Header.Set("Cookie", cookie)
	}

	room, err := missevan.GetRoomInfo()
	if err != nil {
		return nil, err
	}

	if room.Status.Open == 0 {
		log.Error().Msgf("房间[%s]未开播", roomId)
		return nil, fmt.Errorf("房间[%s]未开播", roomId)
	}

	missevan.StreamUrls["flv"] = room.Channel.FlvPullUrl
	missevan.StreamUrls["hls"] = room.Channel.HlsPullUrl

	return missevan, nil
}

// Fetch 用于统一处理 API 请求、Header设置
func (m *Missevan) Fetch(baseURL string, params url.Values, header http.Header) (*http.Response, error) {
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

	request.Header = m.Header.Clone()

	// 2. 合并额外的 Header (使用 Set 覆盖/追加)
	if header != nil {
		for key, values := range header {
			if len(values) > 0 {
				// 使用 Set 覆盖 m.Header 中的旧值，或设置新值。
				// 只取 values 数组的第一个值。
				request.Header.Set(key, values[0])
			}
		}
	}

	if host := request.Header.Get("Host"); host != "" {
		request.Host = host
		request.Header.Del("Host")
	}

	log.Info().Msgf("Request: %s %s %s \n%v", request.Method, request.Host, request.URL.RequestURI(), request.Header)

	return m.Client.Do(request)
}

// fetchAPI 用于统一处理 API 请求、Header 设置和基本错误检查
func (m *Missevan) fetchAPI(baseURL string, params url.Values) ([]byte, error) {
	response, err := m.Fetch(baseURL, params, nil)
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

func (m *Missevan) GetRoomInfo() (*Room, error) {
	resp, err := m.fetchAPI(getLiveBaseUrl+m.RoomId, nil)
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
