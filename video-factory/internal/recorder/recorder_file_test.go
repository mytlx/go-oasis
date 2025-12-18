package recorder

import (
	"fmt"
	"testing"
	"time"
	"video-factory/pkg/config"

	"github.com/rs/zerolog/log"
)

func TestGenerateFileName(t *testing.T) {

	r := &Recorder{
		Username: "test",
		StreamAt: time.Now().Unix(),
		Sequence: 1,
		Config:   &config.AppConfig{},
	}
	r.Config.Recorder.FilenamePattern = "{{.Username}}_{{.Year}}-{{.Month}}-{{.Day}}_{{.Hour}}-{{.Minute}}-{{.Second}}_{{.Sequence}}"
	name, err := r.GenerateFileName()
	if err != nil {
		log.Err(err)
		return
	}

	fmt.Printf("generated filename: %s\n", name)
}
