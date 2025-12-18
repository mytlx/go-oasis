package recorder

import (
	"context"
	"net/http"
	"testing"
	"time"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"
	"video-factory/pkg/logger"

	"github.com/rs/zerolog/log"
)

// func TestRecorder_GetPlayList(t *testing.T) {
//
// 	fetcher.GlobalClient = &http.Client{}
//
// 	hlsURL := "http://d1-missevan104.bilivideo.com/live-bvc/489331/maoer_30165838_869032634.m3u8?cdn=missevan104&oi=2095728767&pt=web&expires=1765474733&qn=10000&len=0&trid=3ef137597bdf1130afdd27d1064cf076&sigparams=cdn,oi,pt,expires,qn,len,trid&sign=8eb58dafc298fe4ba1b17edd166a049a&sk=dd6689e451588085222b5317170891cad671f642910ae3a3ef2cc131fb53adaf"
// 	path := "test.ts"
// 	recorder, _ := NewRecorder(hlsURL, path)
//
// 	_ = recorder.Start(context.Background())
//
// }

func TestRecorder_Start(t *testing.T) {
	logger.InitLogger()
	r := &Recorder{
		Config: &config.AppConfig{
			Recorder: &config.Recorder{
				FilenamePattern: "{{.Username}}_{{.Year}}-{{.Month}}-{{.Day}}_{{.Hour}}-{{.Minute}}-{{.Second}}_{{.Sequence}}",
				MaxDuration:     60,
				MaxFilesize:     1024 * 1024 * 1024,
			},
		},
		StreamURL: "http://d1-missevan104.bilivideo.com/live-bvc/586617/maoer_5362942_868802213.m3u8?cdn=missevan104&oi=2095728767&pt=web&expires=1766048193&qn=10000&len=0&trid=05fb5209b958cf6c96b00e7bb7be951d&sigparams=cdn,oi,pt,expires,qn,len,trid&sign=964f5f5ef291cf3ef9d0733a942ce6e7&sk=dd6689e451588085222b5317170891cad671f642910ae3a3ef2cc131fb53adaf",
		Username:  "testUsername",
		StreamAt:  time.Now().Unix(),
	}

	fetcher.GlobalClient = &http.Client{}

	err := r.Start(context.Background())
	if err != nil {
		log.Err(err)
	}
}
