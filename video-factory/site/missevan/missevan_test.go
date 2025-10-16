package missevan

import "testing"

func TestNewMissevan(t *testing.T) {
	missevan, err := NewMissevan("109896001", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(missevan.StreamUrls)
}
