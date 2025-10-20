package bili

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"video-factory/internal/manager"
	"video-factory/internal/service"
	"video-factory/internal/streamer"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"
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

	streamInfo, err := s.FetchStreamInfo(defaultQn)
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

	// 保存到数据库
	if err := service.AddOrUpdateRoom(m.Manager); err != nil {
		return nil, err
	}

	// jsonBytes, _ := json.MarshalIndent(m.Manager, "", "  ")
	jsonBytes, _ := json.Marshal(m.Manager)
	log.Info().Msgf("[Init] Manager: %s", string(jsonBytes))

	return m, nil
}

// Fetch 重试机制
func (m *Manager) Fetch(baseURL string, params url.Values, extraHeader http.Header) (*http.Response, error) {
	// 由于 Header 可能在 Refresh 中被 Streamer 更新，我们总是获取最新的 Header
	executor := func(method, url string, p url.Values) (*http.Response, error) {
		return fetcher.Fetch(method, url, p, m.Manager.Streamer.GetInfo().Header)
	}
	return fetcher.FetchWithRefresh(m, executor, "GET", baseURL, params)
}

func (m *Manager) Get() *manager.Manager {
	return m.Manager
}

func (m *Manager) AutoRefresh() {
	m.Manager.StartAutoRefresh(safetyExpireTimeInterval)
}

func (m *Manager) Refresh(retryTimes int) error {
	return manager.CommonRefresh(
		m.Manager, // 假设 Manager 是内嵌的字段或引用
		m,         // 传递 BiliManager 自身作为 RefreshStrategy
		retryTimes,
		safetyExpireTimeInterval,
	)
}

// ExecuteFetchStreamInfo 实现 streamer.RefreshStrategy 接口的方法
func (m *Manager) ExecuteFetchStreamInfo() (*streamer.StreamInfo, error) {
	s := m.Manager.Streamer
	return s.FetchStreamInfo(s.GetStreamInfo().SelectedQn)
}

// ParseExpiration 实现 streamer.RefreshStrategy 接口的方法
func (m *Manager) ParseExpiration(streamUrl string) (time.Time, error) {
	// 这是具体的 B站 URL 解析逻辑
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
