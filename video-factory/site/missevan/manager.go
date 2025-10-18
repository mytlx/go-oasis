package missevan

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"video-factory/config"
	"video-factory/fetcher"
	"video-factory/manager"
	"video-factory/streamer"
)

const safetyExpireTimeInterval = 5 * time.Minute

type Manager struct {
	Manager *manager.Manager `json:"Manager"`
}

func NewManager(rid string, config *config.AppConfig) (*Manager, error) {
	s := NewStreamer(rid, config)

	// 初始化房间
	err := s.InitRoom()
	if err != nil {
		return nil, fmt.Errorf("初始化房间失败: %w", err)
	}

	streamInfo, err := s.FetchStreamInfo(0)
	if err != nil {
		return nil, fmt.Errorf("获取真实流地址失败: %w", err)
	}

	log.Info().Msg("--- 成功获取到的 HLS 流媒体地址 ---")
	for quality, steamUrl := range streamInfo.StreamUrls {
		log.Info().Msgf("[%s] -> %s", quality, steamUrl)
	}
	log.Info().Msg("---------------------------------------------------------------------")
	// 默认选择第一个流
	var selectUrl string
	for line, stream := range streamInfo.StreamUrls {
		selectUrl = stream
		log.Info().Msgf("已选择：[%s] -> %s", line, stream)
		break
	}

	if selectUrl == "" {
		return nil, fmt.Errorf("未找到可用的 HLS 播放地址。")
	}

	expireTime, err := parseExpire(selectUrl)
	if err != nil {
		return nil, fmt.Errorf("解析 expireTime 失败: %w", err)
	}

	m := &Manager{
		&manager.Manager{
			Id:               rid,
			Streamer:         s,
			CurrentURL:       selectUrl,
			ProxyURL:         fmt.Sprintf("http://localhost:%d/%s/proxy/%s/index.m3u8", config.Port, baseURLPrefix, rid),
			ActualExpireTime: expireTime,
			SafetyExpireTime: expireTime.Add(-safetyExpireTimeInterval),
			LastRefresh:      time.Now(),
		},
	}
	m.Manager.IManager = m

	// jsonBytes, _ := json.MarshalIndent(m.Manager, "", "  ")
	jsonBytes, _ := json.Marshal(m.Manager)
	log.Info().Msgf("[Init] Manager: %s", string(jsonBytes))

	return m, nil
}

func (m *Manager) AutoRefresh() {
	m.Manager.StartAutoRefresh(safetyExpireTimeInterval)
}

func (m *Manager) Refresh(retryTimes int) error {
	return manager.CommonRefresh(
		m.Manager, // 假设 Manager 是内嵌的字段或引用
		m,         // 自身作为 RefreshStrategy
		retryTimes,
		safetyExpireTimeInterval,
	)
}

func (m *Manager) Fetch(baseURL string, params url.Values, extraHeader http.Header) (*http.Response, error) {
	executor := func(method, url string, p url.Values) (*http.Response, error) {
		// 猫耳需要补充 host
		baseHeader := m.Manager.Streamer.GetInfo().Header
		requestHeader := baseHeader.Clone()
		for k, vv := range extraHeader {
			requestHeader.Del(k)
			for _, v := range vv {
				requestHeader.Add(k, v)
			}
		}
		return fetcher.Fetch(method, url, p, requestHeader)
	}
	return fetcher.FetchWithRefresh(m, executor, "GET", baseURL, params)
}

func (m *Manager) Get() *manager.Manager {
	return m.Manager
}

func (m *Manager) ExecuteFetchStreamInfo() (*streamer.StreamInfo, error) {
	s := m.Manager.Streamer
	return s.FetchStreamInfo(s.GetStreamInfo().SelectedQn)
}

func (m *Manager) ParseExpiration(streamUrl string) (time.Time, error) {
	return parseExpire(streamUrl)
}

func parseExpire(hlsUrl string) (time.Time, error) {
	parsedUrl, err := url.Parse(hlsUrl)
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
