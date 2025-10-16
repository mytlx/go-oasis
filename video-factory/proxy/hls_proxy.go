package proxy

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strings"
	"video-factory/site/missevan"
)

func MissevanHandler(m *missevan.Missevan) gin.HandlerFunc {
	return func(c *gin.Context) {
		filenameWithSlash := c.Param("file")
		filename := strings.TrimPrefix(filenameWithSlash, "/")

		currentUrl := m.StreamUrls["hls"]
		parsedUrl, _ := url.Parse(currentUrl)

		var targetUrl *url.URL

		if strings.HasSuffix(filename, ".m3u8") {
			targetUrl = parsedUrl
		} else {
			tempUrl := *parsedUrl
			lastSlash := strings.LastIndex(tempUrl.Path, "/")
			if lastSlash != -1 {
				// 截断路径，只保留目录部分（例如 /live-bvc/.../2500/）
				tempUrl.Path = tempUrl.Path[:lastSlash+1]
			} else {
				log.Error().Msg("hls源路径解析有误")
				c.String(http.StatusInternalServerError, "Internal server error")
			}
			relativeURL, _ := url.Parse(filename)
			targetUrl = tempUrl.ResolveReference(relativeURL)
			targetUrl.RawQuery = parsedUrl.RawQuery
		}
		log.Printf("代理请求: %s -> %s", c.Request.URL.RequestURI(), targetUrl.String())

		header := m.Header
		header.Set("Host", "d1-missevan04.bilivideo.com")
		header.Set("Origin", "https://fm.missevan.com")
		resp, err := m.Fetch(targetUrl.String(), nil, header)
		if err != nil {
			log.Err(err).Msg("错误: 执行 HTTP 请求失败")
			c.String(http.StatusBadGateway, "Error fetching stream data")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Error().Msgf("错误: HTTP 请求返回状态码 %d", resp.StatusCode)
			c.String(http.StatusBadGateway, "Error fetching stream data")
			return
		}

		// 转发response给客户端
		// 复制状态码和 Headers
		// 注意：M3U8 文件的 Content-Type 必须正确转发，通常是 application/vnd.apple.mpegurl
		for header, values := range resp.Header {
			for _, value := range values {
				c.Writer.Header().Add(header, value)
			}
		}
		c.Status(resp.StatusCode) // 最佳实践：先设置 Headers，再写入 Status Code

		// 复制响应体 (M3U8 内容或 TS 片段数据)
		if _, err = io.Copy(c.Writer, resp.Body); err != nil {
			log.Err(err).Msg("转发响应体失败")
		}

	}
}