package bilibili

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type Manager struct {
	ManagerId        string
	BiliClient       *BiliBili
	CurrentURL       string
	ActualExpireTime time.Time
	ExpectExpireTime time.Time
	LastRefresh      time.Time
	Mutex            sync.RWMutex
}

func NewManager(rid string, cookie string) *Manager {

	biliClient, err := NewBiliBili(rid, cookie)
	if err != nil {
		log.Fatalf("创建 Bilibili 客户端失败: %v", err)
	}

	streams, err := biliClient.GetRealURL(biliClient.SelectedQn)
	if err != nil {
		log.Fatalf("获取真实流地址失败: %v", err)
	}

	log.Println("--- 成功获取到的 HLS 流媒体地址 ---")
	for quality, steamUrl := range streams {
		log.Printf("[%s] -> %s", quality, steamUrl)
	}
	log.Println("---------------------------------------------------------------------")

	// 默认选择第一个流
	var selectUrl string
	for line, stream := range streams {
		selectUrl = stream
		log.Printf("已选择：[%s] -> %s", line, stream)
		break
	}

	if selectUrl == "" {
		log.Fatal("未找到可用的 HLS 播放地址。")
	}

	expireTime, err := parseExpire(selectUrl)
	if err != nil {
		log.Fatalf("解析expireTime失败: %v", err)
	}

	manager := &Manager{
		ManagerId:        "manager_" + rid,
		BiliClient:       biliClient,
		CurrentURL:       selectUrl,
		ActualExpireTime: expireTime,
		ExpectExpireTime: expireTime.Add(-5 * time.Minute),
		LastRefresh:      time.Now(),
	}

	// 开启自动刷新
	go manager.AutoRefresh()

	return manager
}

func (manager *Manager) Fetch(baseURL string, params url.Values, isRetry bool) (*http.Response, error) {
	response, err := manager.BiliClient.Fetch(baseURL, params)
	if err != nil {
		log.Printf("[Fetch] HTTP请求失败: %v", err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotModified {
		log.Printf("[Fetch] HTTP请求失败，状态码: %d", response.StatusCode)

		// 如果已经是重试调用，则不再刷新和重试，直接返回错误
		if isRetry {
			log.Printf("[Fetch] 重试调用失败，不再尝试刷新。")
			return nil, fmt.Errorf("http status code: %d after retry", response.StatusCode)
		}

		log.Println("[Fetch] 尝试刷新直播流并重试一次...")
		if refreshErr := manager.Refresh(5); refreshErr != nil {
			log.Printf("[Fetch] 刷新直播流失败: %v", refreshErr)
			return nil, fmt.Errorf("http status code: %d, and refresh failed: %w", response.StatusCode, refreshErr)
		}

		response, err = manager.Fetch(baseURL, params, true)
	}
	return response, err
}

func (manager *Manager) AutoRefresh() {
	ticker := time.NewTicker(4 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		manager.Mutex.RLock()
		expectExpireTime := manager.ExpectExpireTime
		manager.Mutex.RUnlock()
		if time.Now().After(expectExpireTime) {
			log.Printf("[AutoRefresh] 过期时间到，自动刷新直播流...")
			err := manager.Refresh(5)
			if err != nil {
				log.Printf("[AutoRefresh] 刷新直播流失败: %v", err)
			}
		}
	}
}

func (manager *Manager) Refresh(retryTimes int) error {
	log.Println("[Refresh] 正在更新直播流 token...")
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
			log.Printf("[Refresh] 第%d次重试，失败: %v", cnt, err)
		}
		urls, err := manager.BiliClient.GetRealURL(manager.BiliClient.SelectedQn)
		if err != nil {
			log.Println("[Refresh] 刷新直播流失败:", err)
			continue
		}
		for _, streamUrl := range urls {
			expireTime, err := parseExpire(streamUrl)
			if err != nil {
				log.Printf("[Refresh] 解析expireTime失败: %v", err)
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
		log.Printf("[Refresh] 所有重试均失败，上次错误: %v", err)
		return err
	}

	manager.Mutex.Lock()
	manager.CurrentURL = newStreamUrl
	manager.ActualExpireTime = newExpireTime
	manager.ExpectExpireTime = newExpireTime.Add(-5 * time.Minute)
	manager.LastRefresh = time.Now()
	manager.Mutex.Unlock()

	log.Println("[Refresh] 更新成功")
	return err
}

func parseExpire(hlsUrl string) (time.Time, error) {
	parsedUrl, err := url.Parse(hlsUrl)
	if err != nil {
		log.Printf("解析 HLS URL 失败: %v", err)
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
