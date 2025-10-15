package proxy

import (
	"log"
	"net/url"
	"strings"
	"testing"
)

func TestUrlResolve(t *testing.T) {

	hlsUrl := "https://cn-hljheb-ct-01-03.bilivideo.com/live-bvc/707609/live_3546755249473741_79322200_2500/index.m3u8?expires=1760497678&len=0&oi=0x240e03101e846b00911d8e65f36919c3&pt=html5&qn=250&trid=10071b8304fe40d27fec7087d190b868ef01&bmt=1&sigparams=cdn,expires,len,oi,pt,qn,trid,bmt&cdn=cn-gotcha01&sign=13d3b4c0182ac011810a95eb6fc13f6e&site=b80c151fb800a63a5a38794fead29623&free_type=0&mid=0&sche=ban&bvchls=1&sid=cn-hljheb-ct-01-03&chash=1&sg=lr&trace=8388625&isp=ct&rg=NorthEast&pv=Jilin&suffix=2500&source=puv3_onetier&sl=1&hdr_type=0&sk=bd33b3d41b88a13dcbcc0d2f14d208da&deploy_env=prod&info_source=cache&origin_bitrate=1767&hot_cdn=0&flvsk=f5567232dc2e0ab32035f5e1ab4232f3&media_type=0&pp=srt&score=39&p2p_type=-1&codec=0&vd=nc&zoneid_l=151420929&sid_l=live_3546755249473741_79322200_2500&src=puv3&order=1"

	targetRequestURL := resolveUrl(hlsUrl, "/bilibili/524833285.m4s")
	log.Printf("目标URL: %s\n", targetRequestURL.String())

	targetRequestURL2 := resolveUrl(hlsUrl, "/bilibili/manager_524833285.m3u8")
	log.Printf("目标URL: %s\n", targetRequestURL2.String())

}

func resolveUrl(hlsUrl, requestUrl string) *url.URL {
	parsedHlsUrl, _ := url.Parse(hlsUrl)
	targetRequestURL := parsedHlsUrl

	// 目标URL的路径部分替换为客户端请求的路径，去掉 /bilibili 前缀
	requestPath := strings.TrimPrefix(requestUrl, "/bilibili")
	if !strings.Contains(requestPath, "manager_") {
		baseUrl := *parsedHlsUrl
		lastSlash := strings.LastIndex(baseUrl.Path, "/")
		if lastSlash != -1 {
			// 截断路径，只保留目录部分（例如 /live-bvc/.../2500/）
			baseUrl.Path = baseUrl.Path[:lastSlash+1]
		} else {
			// 如果路径中没有斜杠（不太可能），则保留原始路径 或者设置为根目录 "/"
			baseUrl.Path = "/"
		}

		relativeURL, _ := url.Parse(strings.TrimPrefix(requestPath, "/"))
		// 自动继承 scheme, host，并正确地将相对路径附加到基准路径上
		targetRequestURL = baseUrl.ResolveReference(relativeURL)
		// 保留原始 token
		targetRequestURL.RawQuery = parsedHlsUrl.RawQuery
		// // ts请求重新拼接
		// targetRequestURL.Path = requestPath                // 使用 客户端 请求的路径
		// targetRequestURL.RawQuery = parsedHlsUrl.RawQuery // 保留原始 token
	}
	return targetRequestURL
}
