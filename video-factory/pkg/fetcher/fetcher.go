package fetcher

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"time"
	"video-factory/pkg/config"
)

type Refresher interface {
	// Refresh 方法负责执行业务逻辑上的刷新操作（如获取新的鉴权URL）。
	Refresh(retryTimes int) error
}

// RequestExecutor 是一个委托函数，用于执行实际的 HTTP 请求
type RequestExecutor func(method, baseURL string, params url.Values) (*http.Response, error)

// GlobalClient 是一个通用的 HTTP 客户端实例
var GlobalClient *http.Client

func Init(cfg *config.AppConfig) {
	transport := &http.Transport{}

	if cfg.Proxy.Protocol == "" {
		cfg.Proxy.Protocol = "http"
	}

	switch {
	case cfg.Proxy.Enabled && cfg.Proxy.SystemProxy:
		// 使用系统代理
		transport.Proxy = http.ProxyFromEnvironment
		log.Info().Msg("使用系统代理")
	case cfg.Proxy.Enabled && cfg.Proxy.Host != "" && cfg.Proxy.Port >= 1024 && cfg.Proxy.Port <= 65535:
		// 使用 host + port
		proxyAddr := fmt.Sprintf("%s://%s:%d", cfg.Proxy.Protocol, cfg.Proxy.Host, cfg.Proxy.Port)
		user := url.QueryEscape(cfg.Proxy.Username)
		pass := url.QueryEscape(cfg.Proxy.Password)
		if user != "" && pass != "" {
			proxyAddr = fmt.Sprintf("%s://%s:%s@%s:%d", cfg.Proxy.Protocol, user, pass, cfg.Proxy.Host, cfg.Proxy.Port)
		}
		proxyURL, err := url.Parse(proxyAddr)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			log.Info().Msgf("使用代理: %s", proxyAddr)
		}
	default:
		log.Info().Msg("未启用代理")
	}

	GlobalClient = &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}
}

// Fetch 通用请求方法，适用于所有平台的 API 调用
func Fetch(method string, baseURL string, params url.Values, header http.Header) (*http.Response, error) {
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

	request, err := http.NewRequest(method, parsedURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置 Header
	request.Header = header
	// 特殊处理 host
	if host := request.Header.Get("Host"); host != "" {
		request.Host = host
		request.Header.Del("Host")
	}

	return GlobalClient.Do(request)
}

// FetchBody 用于获取并读取 responseBody
func FetchBody(baseURL string, params url.Values, header http.Header) ([]byte, error) {
	response, err := Fetch(http.MethodGet, baseURL, params, header)
	if err != nil {
		return nil, fmt.Errorf("执行请求失败: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotModified {
		// 这里可以打印更详细的错误日志
		return nil, fmt.Errorf("API 返回错误状态码: %d", response.StatusCode)
	}

	bodyBytes, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", readErr)
	}
	return bodyBytes, nil
}

// FetchWithRefresh 用于尝试刷新状态并重试
// 注意 header 可能在刷新时会发生变化，所以传入的executor闭包中应保持header更新
func FetchWithRefresh(refresher Refresher, executor RequestExecutor, method string,
	baseURL string, params url.Values) (*http.Response, error) {

	// 默认不重试
	isRetry := false

	for {
		response, err := executor(method, baseURL, params)

		// 1. 检查网络错误
		if err != nil {
			log.Err(err).Msg("[FetchWithRefresh] HTTP请求失败")
			return nil, err
		}

		// 2. 检查状态码
		if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusNotModified {
			// 成功，直接返回
			return response, nil
		}

		// 3. 状态码不 OK，检查是否已重试
		log.Error().Msgf("[FetchWithRefresh] HTTP请求失败，状态码: %d", response.StatusCode)

		if isRetry {
			log.Error().Msg("[FetchWithRefresh] 重试调用失败，不再尝试刷新。")
			return nil, fmt.Errorf("http status code: %d after retry", response.StatusCode)
		}

		// 4. 尝试刷新并设置重试标记
		log.Info().Msg("[FetchWithRefresh] 尝试刷新状态并重试一次...")
		if refreshErr := refresher.Refresh(5); refreshErr != nil {
			log.Err(refreshErr).Msg("[FetchWithRefresh] 刷新失败")
			return nil, fmt.Errorf("http status code: %d, and refresh failed: %w", response.StatusCode, refreshErr)
		}

		// 第一次失败，设置重试标记，进入下一个循环
		isRetry = true
	}
}
