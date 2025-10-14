package proxy

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// hlsSourceUrl  HLS 子播放列表链接（chunklist_...m3u8）
var hlsSourceUrl string

const RefererHeader = "https://live.bilibili.com"

// UserAgent 模拟浏览器，避免被识别为爬虫
const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36"

// CookieHeader 如果需要登录才能观看，请在此处填写完整的 Cookie 字符串
const CookieHeader = ""

// proxyHandler 处理所有来自 PotPlayer 的请求，并转发给 B站 HLS 源
func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析目标 URL
	// 解析 HLS_SOURCE_URL 以获取 Host, Scheme 和 Query
	parsedSourceURL, err := url.Parse(hlsSourceUrl)
	if err != nil {
		log.Printf("错误: 解析 HLS_SOURCE_URL 失败: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 构造一个新的请求 URL，使用 PotPlayer 请求的路径，但保持原始 HLS URL 的 Host 和 Scheme
	// 例如，PotPlayer 请求 http://localhost:8090/media_10538.ts
	// 我们需要请求 https://stream.bili.com/live/media_10538.ts?token=...

	// 目标 URL 的路径部分替换为 PotPlayer 请求的路径
	targetPath := r.URL.Path

	// 构建最终目标请求 URL
	targetRequestURL := *parsedSourceURL
	targetRequestURL.Path = targetPath                   // 使用 PotPlayer 请求的路径
	targetRequestURL.RawQuery = parsedSourceURL.RawQuery // 保留原始 token

	// log.Printf("代理请求: %s -> %s", r.URL.RequestURI(), targetRequestURL.String())

	// 2. 创建并配置代理请求
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetRequestURL.String(), nil)
	if err != nil {
		log.Printf("错误: 创建代理请求失败: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 3. **注入关键的反爬 Headers** (核心步骤)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", RefererHeader)
	if CookieHeader != "" {
		req.Header.Set("Cookie", CookieHeader)
	}
	// 转发客户端可能需要的 Accept 和 Accept-Encoding 头部
	for header, values := range r.Header {
		if header == "Accept" || header == "Accept-Encoding" {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}
	}

	// 4. 执行代理请求
	client := http.Client{
		Timeout: 30 * time.Second, // 设置一个合理的超时
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("错误: 执行 HTTP 请求失败: %v", err)
		http.Error(w, "Error fetching stream data", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 5. 转发响应给 PotPlayer
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

// func main() {
//
// 	var roomID string
//
// 	if roomID == "" {
// 		fmt.Print("请输入bilibili直播房间号或url:\n")
// 		fmt.Scanln(&roomID)
// 	}
//
// 	if roomID == "" {
// 		log.Println("房间号不能为空。")
// 		return
// 	}
//
// 	log.Printf("正在尝试获取房间号: %s 的直播流...", roomID)
//
// 	streamUrls, err := bilibili.GetRealURL(roomID)
//
// 	if err != nil {
// 		log.Println("获取真实流媒体地址失败:", err)
// 		return
// 	}
//
// 	log.Println("--- 成功获取到的 HLS 流媒体地址 ---")
// 	for line, stream := range streamUrls {
// 		log.Printf("%s: %s\n", line, stream)
// 	}
// 	log.Println("---------------------------------------------------------------------")
//
// 	for line, stream := range streamUrls {
// 		hlsSourceUrl = stream
// 		log.Printf("已选择：[%s] -> %s", line, hlsSourceUrl)
// 		break
// 	}
//
// 	// 统一处理所有请求到 proxyHandler
// 	http.HandleFunc("/", proxyHandler)
//
// 	log.Println("HLS 代理服务已启动，监听端口 8090")
//
// 	// 解析 URL 以获取正确的路径和查询字符串，用于提示用户
// 	parsedSourceURL, err := url.Parse(hlsSourceUrl)
// 	if err != nil {
// 		log.Fatalf("错误: 解析 HLS_SOURCE_URL 失败: %v", err)
// 	}
// 	// 使用 RequestURI() 获取 /path/to/stream.m3u8?token=...
// 	log.Printf("请在 PotPlayer 中打开链接: http://localhost:8090%s\n", parsedSourceURL.RequestURI())
//
// 	// 启动服务器
// 	log.Fatal(http.ListenAndServe(":8090", nil))
// }
