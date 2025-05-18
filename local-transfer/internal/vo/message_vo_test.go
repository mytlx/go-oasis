package vo

import (
	"encoding/json"
	"log"
	"testing"
)

func TestMessageVO(t *testing.T) {

	messageStr := "{\n  \"id\": 1746789001123,\n  \"content\": \"你好，这是测试消息\",\n  \"source\": {\n    \"id\": 1,\n    \"device_name\": \"手机A\",\n    \"device_type\": \"Android\",\n    \"ip\": \"192.168.1.10\"\n  },\n  \"target\": {\n    \"id\": 2,\n    \"device_name\": \"电脑B\",\n    \"device_type\": \"Windows\",\n    \"ip\": \"192.168.1.20\"\n  },\n  \"type\": \"text\",\n  \"status\": 1,\n  \"create_time\": \"2025-05-17T22:30:00+08:00\"\n}\n"

	var message MessageVO
	err := json.Unmarshal([]byte(messageStr), &message)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(message)

}
