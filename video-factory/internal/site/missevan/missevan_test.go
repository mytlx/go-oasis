package missevan

import (
	"regexp"
	"testing"
)

func TestNewMissevan(t *testing.T) {
	// missevan, err := NewMissevan("109896001", "")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// t.Log(missevan.StreamUrls)
}

func TestRegex(t *testing.T) {
	urls := []string{
		"https://fm.missevan.com/live/195526168",
		"http://fm.missevan.com/live/195526168",
		"fm.missevan.com/live/195526168",
		"https://fm.missevan.com/live/195526168?live_from=85001&spm_id_from=444.41.live_users.item.click",
	}

	compile := regexp.MustCompile(`(?:https?://)?fm\.missevan\.com/live/(\d+)`)

	for i := 0; i < len(urls); i++ {
		if matches := compile.FindStringSubmatch(urls[i]); len(matches) >= 2 {
			t.Log(matches[1])
		}
	}
}
