package proxy

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"video-factory/bilibili"
)

// BiliBiliHandler 处理所有来自客户端的请求，转发给B站
func BiliBiliHandler(manager *bilibili.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		manager.Mutex.RLock()
		currentURL := manager.CurrentURL
		manager.Mutex.RUnlock()

		parsedHlsUrl, err := url.Parse(currentURL)
		if err != nil {
			log.Printf("错误: 解析hls源失败: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		// 目标URL的路径部分替换为客户端请求的路径，去掉 /bilibili 前缀
		targetPath := strings.TrimPrefix(r.URL.Path, "/bilibili")
		targetRequestURL := *parsedHlsUrl
		if !strings.Contains(targetPath, "manager_") {
			// ts请求重新拼接
			targetRequestURL.Path = targetPath                // 使用 客户端 请求的路径
			targetRequestURL.RawQuery = parsedHlsUrl.RawQuery // 保留原始 token
		}

		log.Printf("代理请求: %s -> %s", r.URL.RequestURI(), targetRequestURL.String())

		// 转发请求
		resp, err := manager.Fetch(targetRequestURL.String(), nil, false)
		if err != nil {
			log.Printf("错误: 执行 HTTP 请求失败: %v", err)
			http.Error(w, "Error fetching stream data", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		// 转发response给客户端
		// 复制状态码和 Headers
		// 注意：M3U8 文件的 Content-Type 必须正确转发，通常是 application/vnd.apple.mpegurl
		for header, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(header, value)
			}
		}
		w.WriteHeader(resp.StatusCode) // 最佳实践：先设置 Headers，再写入 Status Code

		// 复制响应体 (M3U8 内容或 TS 片段数据)
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			log.Printf("错误: 转发响应体失败: %v", err)
		}
	}
}
