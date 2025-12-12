package recorder

import (
	"context"
	"net/http"
	"testing"
	"video-factory/pkg/fetcher"
)

func TestRecorder_GetPlayList(t *testing.T) {

	fetcher.GlobalClient = &http.Client{}

	hlsURL := "http://d1-missevan104.bilivideo.com/live-bvc/489331/maoer_30165838_869032634.m3u8?cdn=missevan104&oi=2095728767&pt=web&expires=1765474733&qn=10000&len=0&trid=3ef137597bdf1130afdd27d1064cf076&sigparams=cdn,oi,pt,expires,qn,len,trid&sign=8eb58dafc298fe4ba1b17edd166a049a&sk=dd6689e451588085222b5317170891cad671f642910ae3a3ef2cc131fb53adaf"
	path := "test.ts"
	recorder, _ := NewRecorder(hlsURL, path)

	_ = recorder.Start(context.Background())

}
