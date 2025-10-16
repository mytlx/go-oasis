package bili

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"video-factory/fetcher"
	"video-factory/manager"
)

const (
	refreshInterval          = 4 * time.Minute
	expectExpireTimeInterval = -5 * time.Minute
)

type Manager struct {
	Manager *manager.Manager `json:"Manager"`
}

func NewManager(rid string, cookie string) (*Manager, error) {
	s := NewStreamer(rid, cookie)
	// if err != nil {
	// 	return nil, fmt.Errorf("创建 Bilibili 客户端失败: %w", err)
	// }

	// 初始化房间
	err := s.InitRoom()
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
			ActualExpireTime: expireTime,
			ExpectExpireTime: expireTime.Add(expectExpireTimeInterval),
			LastRefresh:      time.Now(),
		},
	}

	jsonBytes, _ := json.MarshalIndent(m, "", "  ")
	log.Info().Msgf("[Init] Manager: %s", string(jsonBytes))

	return m, nil
}


// tlxTODO: 过后抽出
func (manager *Manager) Fetch(baseURL string, params url.Values, isRetry bool) (*http.Response, error) {

	response, err := fetcher.Fetch("GET", baseURL, params, manager.Manager.Streamer.GetInfo().Header)
	if err != nil {
		log.Err(err).Msg("[Fetch] HTTP请求失败")
		return nil, err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotModified {
		log.Error().Msgf("[Fetch] HTTP请求失败，状态码: %d", response.StatusCode)

		// 如果已经是重试调用，则不再刷新和重试，直接返回错误
		if isRetry {
			log.Error().Msg("[Fetch] 重试调用失败，不再尝试刷新。")
			return nil, fmt.Errorf("http status code: %d after retry", response.StatusCode)
		}

		log.Info().Msg("[Fetch] 尝试刷新直播流并重试一次...")
		if refreshErr := manager.Refresh(5); refreshErr != nil {
			log.Err(refreshErr).Msg("[Fetch] 刷新直播流失败")
			return nil, fmt.Errorf("http status code: %d, and refresh failed: %w", response.StatusCode, refreshErr)
		}

		response, err = manager.Fetch(baseURL, params, true)
	}
	return response, err
}

func (manager *Manager) Get() *manager.Manager {
	return manager.Manager
}

func (manager *Manager) AutoRefresh() {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		manager.Manager.Mutex.RLock()
		expectExpireTime := manager.Manager.ExpectExpireTime
		manager.Manager.Mutex.RUnlock()
		if time.Now().After(expectExpireTime) {
			log.Info().Msg("[AutoRefresh] 过期时间到，自动刷新直播流...")
			err := manager.Refresh(5)
			if err != nil {
				log.Err(err).Msg("[AutoRefresh] 刷新直播流失败")
			}
		}
	}
}

func (manager *Manager) Refresh(retryTimes int) error {
	log.Info().Msg("[Refresh] 正在更新直播流 token...")
	if retryTimes < 0 {
		retryTimes = 0
	}
	if retryTimes > 10 {
		retryTimes = 10
	}

	var err error
	var newStreamUrl string
	var newExpireTime time.Time
	for cnt := 0; cnt <= retryTimes; cnt++ {
		if cnt > 0 {
			time.Sleep(2 * time.Second)
			log.Err(err).Msgf("[Refresh] 第%d次重试", cnt)
		}
		s := manager.Manager.Streamer
		streamInfo, err := s.FetchStreamInfo(s.GetStreamInfo().SelectedQn)
		if err != nil {
			log.Err(err).Msg("[Refresh] 刷新直播流失败:")
			continue
		}
		for _, streamUrl := range streamInfo.StreamUrls {
			expireTime, err := parseExpire(streamUrl)
			if err != nil {
				log.Err(err).Msg("[Refresh] 解析expireTime失败")
				continue
			}
			newStreamUrl = streamUrl
			newExpireTime = expireTime
			err = nil
			break
		}
		break
	}

	// 检查是否所有重试都失败
	if newStreamUrl == "" {
		log.Err(err).Msg("[Refresh] 所有重试均失败，上次错误")
		return err
	}

	manager.Manager.Mutex.Lock()
	manager.Manager.CurrentURL = newStreamUrl
	manager.Manager.ActualExpireTime = newExpireTime
	manager.Manager.ExpectExpireTime = newExpireTime.Add(expectExpireTimeInterval)
	manager.Manager.LastRefresh = time.Now()
	manager.Manager.Mutex.Unlock()

	log.Info().Msg("[Refresh] 更新成功")
	jsonBytes, _ := json.MarshalIndent(manager, "", "  ")
	log.Info().Msgf("[Refresh] Manager: %s", string(jsonBytes))
	return err
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
