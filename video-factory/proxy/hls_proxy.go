package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"video-factory/bilibili"
)

// BiliHandler 处理所有来自客户端的请求，转发给B站
func BiliHandler(pool *bilibili.ManagerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取路径参数 Manager ID
		managerID := r.PathValue("managerId")
		if managerID == "" {
			http.Error(w, "缺少ManagerId", http.StatusBadRequest)
			return
		}
		filename := r.PathValue("file")

		manager, ok := pool.Get(managerID)
		if !ok {
			http.Error(w, fmt.Sprintf("直播间[%s]未配置", managerID), http.StatusNotFound)
			return
		}

		manager.Mutex.RLock()
		currentURL := manager.CurrentURL
		manager.Mutex.RUnlock()

		parsedHlsUrl, err := url.Parse(currentURL)
		if err != nil {
			log.Printf("解析hls源失败: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		var targetURL *url.URL

		if strings.HasSuffix(filename, ".m3u8") {
			targetURL = parsedHlsUrl
		} else if strings.HasSuffix(filename, ".ts") || strings.HasSuffix(filename, ".m4s") {
			tempUrl := *parsedHlsUrl
			lastSlash := strings.LastIndex(tempUrl.Path, "/")
			if lastSlash != -1 {
				// 截断路径，只保留目录部分（例如 /live-bvc/.../2500/）
				tempUrl.Path = tempUrl.Path[:lastSlash+1]
			} else {
				log.Printf("hls源路径解析有误: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}

			relativeURL, err := url.Parse(strings.TrimPrefix(filename, "/"))
			if err != nil {
				log.Printf("解析相对路径失败: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			// 自动继承 scheme, host，并正确地将相对路径附加到基准路径上
			targetURL = tempUrl.ResolveReference(relativeURL)
			// 保留原始 token
			targetURL.RawQuery = parsedHlsUrl.RawQuery
		} else {
			log.Printf("不支持的文件类型或路径: %s", r.URL.RequestURI())
			http.Error(w, "Unsupported file type or path", http.StatusNotFound)
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

func BiliRoomAddHandler(pool *bilibili.ManagerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := r.URL.Query().Get("rid")
		if rid == "" {
			http.Error(w, "rid不能为空", http.StatusBadRequest)
			return
		}

		// 检查是否已存在
		if _, ok := pool.Get(rid); ok {
			http.Error(w, fmt.Sprintf("房间[%s]已存在，请访问：http://localhost:8090/bilibili/%s/index.m3u8", rid), http.StatusBadRequest)
			return
		}

		// 新建 Manager
		manager, err := bilibili.NewManager(rid, "")
		if err != nil {
			log.Printf("添加房间 %s 失败: %v", rid, err)
			http.Error(w, fmt.Sprintf("添加房间失败: %v", err), http.StatusInternalServerError)
			return
		}

		// 添加到 ManagerPool
		pool.Add(manager.ManagerId, manager)

		// 启动自动续期
		go manager.AutoRefresh()

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(
			"添加房间[%s]成功，请访问：http://localhost:8090/bilibili/%s/index.m3u8", rid, manager.ManagerId)))
	}
}

func BiliRoomRemoveHandler(pool *bilibili.ManagerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := r.URL.Query().Get("rid")
		if rid == "" {
			http.Error(w, "rid不能为空", http.StatusBadRequest)
			return
		}
		pool.Remove(rid)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf("删除房间[%s]成功", rid)))
	}
}

func BiliRoomDetailHandler(pool *bilibili.ManagerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rid := r.URL.Query().Get("rid")
		if rid == "" {
			http.Error(w, "rid不能为空", http.StatusBadRequest)
			return
		}
		manager, ok := pool.Get(rid)
		if !ok {
			http.Error(w, "房间不存在", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf("http://localhost:8090/bilibili/%s/index.m3u8", manager.ManagerId)))
	}
}
