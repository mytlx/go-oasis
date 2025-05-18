package utils

import (
	"errors"
	"github.com/bwmarrin/snowflake"
	"log"
	"sync"
)

type IDGenerator struct {
	node *snowflake.Node
	once sync.Once
	err  error
}

var generator IDGenerator

func InitIDGenerator(nodeID int64) {
	// 初始化 ID 生成器
	err := Init(1)
	if err != nil {
		log.Fatalf("初始化 ID 生成器失败: %v", err)
	}
}

// Init 初始化 ID 生成器，nodeID 范围：0 ~ 1023
func Init(nodeID int64) error {
	generator.once.Do(func() {
		node, err := snowflake.NewNode(nodeID)
		if err != nil {
			generator.err = err
			return
		}
		generator.node = node
	})
	return generator.err
}

// MustNextID 返回一个全局唯一的 int64 ID
func MustNextID() int64 {
	id, err := NextID()
	if err != nil {
		panic("NextID failed: " + err.Error())
	}
	return id
}

// NextID 返回一个全局唯一的 int64 ID
func NextID() (int64, error) {
	if generator.node == nil {
		return 0, errors.New("ID generator not initialized")
	}
	id := generator.node.Generate()
	if id == 0 {
		return 0, errors.New("failed to generate ID")
	}
	return id.Int64(), nil
}

// MustNextIDString 返回一个全局唯一的字符串形式 ID
func MustNextIDString() string {
	id, err := NextIDString()
	if err != nil {
		panic("NextID failed: " + err.Error())
	}
	return id
}

// NextIDString 返回一个全局唯一的字符串形式 ID
func NextIDString() (string, error) {
	if generator.node == nil {
		return "", errors.New("ID generator not initialized")
	}
	id := generator.node.Generate()
	if id == 0 {
		return "", errors.New("failed to generate ID")
	}
	return id.String(), nil
}
