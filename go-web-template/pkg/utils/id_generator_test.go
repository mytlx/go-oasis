package utils

import (
	"fmt"
	"log"
	"testing"
)

func TestGetId(t *testing.T) {
	// 初始化 ID 生成器
	err := Init(1)
	if err != nil {
		log.Fatalf("初始化 ID 生成器失败: %v", err)
	}

	// 获取唯一的 int64 ID
	id, err := NextID()
	if err != nil {
		log.Fatalf("生成 ID 失败: %v", err)
	}
	fmt.Println("Int64 ID:", id)

	// 获取唯一的 string ID
	strID, err := NextIDString()
	if err != nil {
		log.Fatalf("生成 ID 失败: %v", err)
	}
	fmt.Println("String ID:", strID)
}
