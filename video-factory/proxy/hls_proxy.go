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
		targetURL := parsedHlsUrl
		// 目标URL的路径部分替换为客户端请求的路径，去掉 /bilibili 前缀
		sourcePath := strings.TrimPrefix(r.URL.Path, "/bilibili")
		if !strings.Contains(sourcePath, "manager_") {
			tempUrl := *parsedHlsUrl
			lastSlash := strings.LastIndex(tempUrl.Path, "/")
			if lastSlash != -1 {
				// 截断路径，只保留目录部分（例如 /live-bvc/.../2500/）
				tempUrl.Path = tempUrl.Path[:lastSlash+1]
			} else {
				// 如果路径中没有斜杠（不太可能），则保留原始路径 或者设置为根目录 "/"
				tempUrl.Path = "/"
			}

			relativeURL, err := url.Parse(strings.TrimPrefix(sourcePath, "/"))
			if err != nil {
				log.Printf("错误: 解析相对路径失败: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			// 自动继承 scheme, host，并正确地将相对路径附加到基准路径上
			targetURL = tempUrl.ResolveReference(relativeURL)
			// 保留原始 token
			targetURL.RawQuery = parsedHlsUrl.RawQuery
		}

		// log.Printf("代理请求: %s -> %s", r.URL.RequestURI(), targetURL.String())

		// 转发请求
		resp, err := manager.Fetch(targetURL.String(), nil, false)
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
