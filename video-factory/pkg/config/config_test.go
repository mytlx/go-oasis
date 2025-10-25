package config

import (
	"fmt"
	"testing"
)

func TestUnflattenConfig(t *testing.T) {
	appConfig := &AppConfig{}

	configMap := map[string]string{
		"port": "8090",
		"proxy.enabled": "true",
		"proxy.system_proxy": "false",
		"proxy.protocol": "http",
		"proxy.host": "127.0.0.1",
		"bili.cookie": "bili_cookie",
	}

	err := UnflattenConfig(appConfig, configMap)
	if err != nil {
		t.Errorf("UnflattenConfig failed: %v", err)
	}

	// marshal, _ := json.Marshal(appConfig)
	// fmt.Println(string(marshal))
	fmt.Printf("%+v\n", appConfig)

}
