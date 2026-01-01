package bili

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// func TestGetRealUrl(t *testing.T) {
// 	urlMap, _ := GetRealURL("1912366159")
// 	fmt.Print(urlMap)
// }

func TestGetMethod(t *testing.T) {
	resp, err := http.DefaultClient.Get("https://b23.tv/sG66zUl")
	fmt.Println(resp, err)

}

func TestGetLongUrl(t *testing.T) {
	shortURL := "https://b23.tv/sG66zUl"

	client := &http.Client{
		// 禁止自动跟随跳转
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(shortURL)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// 输出 Location 头
	longURL := resp.Header.Get("Location")
	fmt.Println("长链接:", longURL)
}

var inputs = []string{
	"1912366159",
	"live.bili.com/1912366159",
	"https://live.bilibili.com/1912366159",
	"https://live.bilibili.com/1912366159?live_from=85001&spm_id_from=444.41.live_users.item.click",
	"https://live.bilibili.com/h5/31084516?broadcast_type=0&is_room_feed=1&plat_id=365&share_from=live&share_medium=android_i&share_plat=android&share_session_id=66a7cc92-af2b-4442-b2bc-8917dc47da70&share_source=COPY&share_tag=s_i&timestamp=1759680785&unique_k=sG66zUl",
	"https://b23.tv/sG66zUl",
	"b23.tv/sG66zUl",
	"【直播描述，巴拉巴拉】 https://b23.tv/sG66zUl",
	"ftuyiosfsdaf",
	"8sdf0sag8",
}

func TestGetRid(t *testing.T) {

	// for _, input := range inputs {
	// 	rid, err := checkAndGetRid(input)
	// 	if err != nil {
	// 		fmt.Printf("current: %s, err: %s \n", input, err)
	// 		continue
	// 	}
	// 	fmt.Printf("current: %s, rid: %s \n", input, rid)
	// }

}

func TestRegex(t *testing.T) {

	reLive := regexp.MustCompile(`(?:https?://)?live\.bili\.com/(?:h5/)?(\d+)`)

	for _, input := range inputs {
		if matches := reLive.FindStringSubmatch(input); len(matches) >= 2 {
			fmt.Printf("long current: %s, rid: ", input)
			fmt.Println(matches[1])
			continue
		}

		reShort := regexp.MustCompile(`b23\.tv/[A-Za-z0-9]+`)
		if matches := reShort.FindStringSubmatch(input); len(matches) >= 1 {
			fmt.Printf("short current: %s, url: ", input)
			fmt.Println(matches[0])
			continue
		}
		fmt.Printf("current: %s, no match\n", input)
	}

}

func Test304(t *testing.T) {

	url := "https://d1--cn-gotcha104.bilivideo.com/live-bvc/397976/live_7734200_bs_1348183_fhd2avc.m3u8?expires=1760444138&len=0&oi=0x240e03101e846b000d7d77bc6def97e7&pt=h5&qn=15000&trid=10030156a8d2ac464f2f461acbf9e168ee30&bmt=1&sigparams=cdn,expires,len,oi,pt,qn,trid,bmt&cdn=cn-gotcha104&sign=1d368792532e0758ae9dfb706e04a0e9&site=85a6fc2c161c5ade3975b320cfb4d262&free_type=0&mid=22846327&sche=ban&trace=16&isp=ct&rg=NorthEast&pv=Jilin&source=puv3_onetier&score=100&strategy_ids=112&sk=e1f4ebebe465a2c143c02b0228f25a8b&long_ab_flag_value=test&hdr_type=0&codec=0&suffix=fhd2avc&strategy_version=latest&long_ab_flag=live_default_longitudinal&hot_cdn=909789&info_source=hot_cache&pp=rtmp&sl=10&origin_bitrate=1543&strategy_types=1&p2p_type=-1&media_type=0&deploy_env=prod&long_ab_id=45&vd=bc&src=puv3&order=1"

	// 2. 创建并配置代理请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("错误: 创建代理请求失败: %v", err)
		return
	}

	// UserAgent 模拟浏览器，避免被识别为爬虫
	const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36"
	// CookieHeader 如果需要登录才能观看，请在此处填写完整的 Cookie 字符串
	const CookieHeader = "DedeUserID=22846327; DedeUserID__ckMd5=35a0abb6b30b3bcf; buvid_fp_plain=undefined; enable_web_push=DISABLE; header_theme_version=CLOSE; buvid_fp=fd8d27c5b2471d4e025799e15162b4b2; buvid3=15763448-D1BB-8ECC-C9E3-9AA2D106972076485infoc; b_nut=1730467376; rpdid=|(Yu|Jl~YlY0J'u~J|RY~J)k; enable_feed_channel=ENABLE; CURRENT_FNVAL=4048; home_feed_column=5; browser_resolution=1638-820; CURRENT_QUALITY=80; theme-tip-show=SHOWED; theme-avatar-tip-show=SHOWED; buvid4=D5E3870B-91EC-81D3-1E2D-FC9C5A5A59ED50985-023092314-VYy5D/aeqCVqylXnHYoaKQ%3D%3D; _uuid=10D10DC68A-A5510-8B98-AEED-1034109373F9F254191infoc; SESSDATA=4e86d512%2C1775223137%2C15215%2Aa1CjChGpUfsEzOBVFcJ5Nw2f4Vs3VcYU1MUwRlqKIdg168GkHod78SHLntf7NI_hU8JroSVkNWNE9iQVVyR1FubWJYU19yZGZmUnNsWk10SVhmeDZlbFRkbmdMWXk5bWtjUXZ0ZDgxQnI3cGs1c1dFSGZBSVgtQlVhRW9YdHQ1d3EzU0FfOWg0aGp3IIEC; bili_jct=d09dc4cd60ee05af4df76425ce086611; LIVE_BUVID=AUTO2417596711856227; bp_t_offset_22846327=1120309933892435968; bili_ticket=eyJhbGciOiJIUzI1NiIsImtpZCI6InMwMyIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NjA2OTUyNzksImlhdCI6MTc2MDQzNjAxOSwicGx0IjotMX0.zLbUn6R6d95EdrQ82F3WuJKJbd1NGZOtzamrX6TAzI0; bili_ticket_expires=1760695219; PVID=2; b_lsid=52E2810FC_199E2328ABD"
	const RefererHeader = "https://live.bilibili.com"

	// 3. **注入关键的反爬 Headers** (核心步骤)
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", RefererHeader)
	if CookieHeader != "" {
		req.Header.Set("Cookie", CookieHeader)
	}

	client := http.Client{
		Timeout: 30 * time.Second, // 设置一个合理的超时
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("错误: 执行 HTTP 请求失败: %v", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("错误: HTTP 请求失败,状态码: %d", resp.StatusCode)
		return
	}
	defer resp.Body.Close()
}

func TestTime(t *testing.T) {

	// 模拟 parsedUrl.Query().Get("expires") 的结果
	// 假设 URL 是 "http://example.com/?expires=1760460102"
	expiresStr := "1760460102"

	// --- 转换步骤 ---

	// 1. 将字符串转换为 int64
	// Unix时间戳通常需要 int64 来确保能容纳未来的时间
	expiresInt, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		fmt.Println("转换时间戳字符串为整数失败:", err)
		return
	}

	// 2. 使用 time.Unix() 转换为 time.Time 类型
	// 第一个参数是秒 (sec)，第二个参数是纳秒 (nsec)，这里设为 0
	expirationTime := time.Unix(expiresInt, 0)

	// --- 打印结果 ---

	fmt.Printf("原始时间戳 (秒): %s\n", expiresStr)
	fmt.Printf("time.Time 对象: %v\n", expirationTime)
	fmt.Printf("格式化后的时间 (UTC): %s\n", expirationTime.UTC().Format(time.RFC3339))
	fmt.Printf("格式化后的时间 (本地时区): %s\n", expirationTime.Local().Format("2006-01-02 15:04:05"))

}

func TestGetLiveTime(t *testing.T) {
	liveTimeStr := "2026-01-01 20:47:32"

	parse, err := time.Parse(time.DateTime, liveTimeStr)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	fmt.Println(parse.String())
	fmt.Println(parse.Unix())
	fmt.Println(parse.UnixMilli())

	tt := time.Unix(parse.Unix(), 0)
	fmt.Println(tt.Format("2006"))
}
