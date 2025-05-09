package utils

import (
	"errors"
	"github.com/bwmarrin/snowflake"
	"sync"
)

type IDGenerator struct {
	node *snowflake.Node
	once sync.Once
	err  error
}

var generator IDGenerator

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
