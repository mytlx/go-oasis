package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/TarsCloud/TarsGo/tars"
)

// --- 对应 Dart 的 GetCdnTokenReq ---
type GetCdnTokenReq struct {
	Url          string
	CdnType      string
	StreamName   string
	PresenterUid int64
}

// --- 对应 Dart 的 GetCdnTokenResp ---
type GetCdnTokenResp struct {
	Url         string
	CdnType     string
	StreamName  string
	PresenterUid int64
	AntiCode    string
	STime       string
	FlvAntiCode string
	HlsAntiCode string
}

// 模拟 Tars 接口生成的 struct
type THLiveServicePrx struct {
	tars.TarsClient
}

// 实现调用方法
func (p *THLiveServicePrx) GetCdnTokenInfo(req *GetCdnTokenReq, resp *GetCdnTokenResp) error {
	return p.Invoke("GetCdnTokenInfo", req, resp)
}

func main() {
	roomID := int64(66025)
	userID := int64(1099531840859)

	// 初始化 Tars 通信
	comm := tars.NewCommunicator()
	proxy := new(THLiveServicePrx) // 这里是 struct 类型
	err := comm.StringToProxy("THLiveService.THLiveObj@tcp -h huya_tars_host -p 10000", proxy)
	if err != nil {
		fmt.Println("StringToProxy 失败:", err)
		return
	}

	req := &GetCdnTokenReq{
		Url:          "",
		CdnType:      "CDN_FLV",
		StreamName:   fmt.Sprintf("%d", roomID),
		PresenterUid: userID,
	}

	resp := &GetCdnTokenResp{}
	err = proxy.GetCdnTokenInfo(req, resp)
	if err != nil {
		fmt.Println("调用 Tars 服务失败:", err)
		return
	}

	realURL := generatePlayURL(resp, userID)
	fmt.Println("直播流地址:", realURL)
}

// 根据返回的 antiCode 等生成真实 URL
func generatePlayURL(resp *GetCdnTokenResp, userID int64) string {
	seqid := fmt.Sprintf("%d", time.Now().UnixMilli()+userID)
	wsTime := fmt.Sprintf("%x", time.Now().Unix()+3600)

	hash0 := md5.Sum([]byte(seqid + "|" + resp.CdnType + "|" + resp.STime))
	hash0Str := hex.EncodeToString(hash0[:])

	hash1 := md5.Sum([]byte(resp.AntiCode + fmt.Sprintf("_%d_%s_%s_%s", userID, resp.StreamName, hash0Str, wsTime)))
	hash1Str := hex.EncodeToString(hash1[:])

	url := fmt.Sprintf("%s?wsSecret=%s&wsTime=%s&uuid=&uid=%d&seqid=%s&ratio=&txyp=&fs=&ctype=%s&ver=1&t=%s",
		resp.Url, hash1Str, wsTime, userID, seqid, resp.CdnType, resp.STime)
	return url
}
