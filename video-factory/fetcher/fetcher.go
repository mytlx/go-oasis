package fetcher

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// GlobalClient 是一个通用的 HTTP 客户端实例
var GlobalClient = &http.Client{Timeout: 15 * time.Second}

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
