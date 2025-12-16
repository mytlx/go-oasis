package missevan

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"video-factory/internal/domain/model"
	"video-factory/internal/iface"
	"video-factory/internal/manager"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"

	"github.com/rs/zerolog/log"
)

const safetyExpireTimeInterval = 5 * time.Minute

type Manager struct {
	Manager *manager.Manager `json:"Manager"`
}

func NewManager(room *model.Room, config *config.AppConfig) (*Manager, error) {
	s := NewStreamer(strconv.FormatInt(room.ID, 10), config)
	config.AddSubscriber(s)

	m := &Manager{
		&manager.Manager{
			Id:               room.ID,
			Streamer:         s,
			ProxyURL:         room.ProxyURL,
			ActualExpireTime: time.Now(),
			SafetyExpireTime: time.Now(),
			LastRefreshTime:  time.Now(),
		},
	}
	m.Manager.IManager = m
	log.Info().Object("manager", m.Manager).Msg("[Init] Manager")
	return m, nil
}

func (m *Manager) Start(ctx context.Context) error {
	// 初始化房间
	if err := m.Manager.Streamer.InitRoom(); err != nil {
		return fmt.Errorf("初始化房间失败: %w", err)
	}

	// 调用公共刷新接口，获取流地址和过期时间
	if err := manager.CommonRefresh(ctx, m.Manager, m, 3, safetyExpireTimeInterval, true); err != nil {
		return err
	}

	// 启动自动刷新
	m.AutoRefresh()

	// tlxTODO: 录制功能也在此启动

	return nil
}

func (m *Manager) AutoRefresh() {
	m.Manager.StartAutoRefresh(safetyExpireTimeInterval)
}

func (m *Manager) StopAutoRefresh() {
	m.Manager.StopAutoRefresh()
}

func (m *Manager) Refresh(ctx context.Context, retryTimes int) error {
	return manager.CommonRefresh(
		ctx,
		m.Manager, // 假设 Manager 是内嵌的字段或引用
		m,         // 自身作为 RefreshStrategy
		retryTimes,
		safetyExpireTimeInterval,
		true,
	)
}

func (m *Manager) Fetch(ctx context.Context, baseURL string, params url.Values, extraHeader http.Header) (*http.Response, error) {
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
	return fetcher.FetchWithRefresh(ctx, m, executor, "GET", baseURL, params)
}

func (m *Manager) GetId() int64 {
	return m.Manager.GetId()
}

func (m *Manager) GetCurrentURL() string {
	return m.Manager.GetCurrentURL()
}

func (m *Manager) GetProxyURL() string {
	return m.Manager.GetProxyURL()
}

func (m *Manager) GetLastRefreshTime() time.Time {
	return m.Manager.GetLastRefreshTime()
}

func (m *Manager) GetLiveStatus() (bool, error) {
	return m.Manager.Streamer.IsLive()
}

func (m *Manager) ExecuteFetchStreamInfo(certainQnFlag bool) (*iface.StreamInfo, error) {
	s := m.Manager.Streamer
	return s.FetchStreamInfo(s.GetStreamInfo().SelectedQn, certainQnFlag)
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
