package db

import (
	"bytes"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"strings"
	"testing"
	"text/template"
	"video-factory/internal/domain/model"
)

func TestGenerateConfigInit(t *testing.T) {
	var err error
	DB, err = gorm.Open(sqlite.Open("E:\\TLX\\Documents\\project\\003_Go\\go-oasis\\video-factory\\db\\video-factory.db"), &gorm.Config{})
	if err != nil {
		log.Printf("[InitDB] 数据库连接失败")
	}
	log.Printf("[InitDB] 数据库连接成功！")

	var configs []model.Config
	DB.Find(&configs)

	// 去除敏感信息
	for i := range configs {
		if strings.Contains(configs[i].Key, "cookie") ||
			strings.Contains(configs[i].Key, "password") ||
			strings.Contains(configs[i].Key, "username") {
			configs[i].Value = ""
		}
	}

	const goStructSliceTemplate = `
var InitialConfigs = []model.Config{
{{- range . }}
    {
       ID:          {{.ID}},
       Key:         "{{.Key}}",
       Value:       "{{.Value}}",
       Description: "{{.Description}}",
    },
{{- end }}
}
`
	temp, err := template.New("go_config_slice").Parse(goStructSliceTemplate)
	if err != nil {
		log.Fatalf("解析模板失败: %v", err)
	}
	// 2. 渲染模板到缓冲区
	var buf bytes.Buffer

	// 直接将切片 initialData 作为数据传给模板
	err = temp.Execute(&buf, configs)
	if err != nil {
		log.Fatalf("渲染模板失败: %v", err)
	}

	// 3. 输出结果 (即你想要的字符串)
	fmt.Println(buf.String())
}
