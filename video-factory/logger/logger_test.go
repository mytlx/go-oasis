package logger

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"testing"
)

func TestLogger(t *testing.T) {
	InitLogger()

	testError := errors.New("这是一个测试错误，用于模拟失败情况")

	log.Debug().Msg("处理所有来自客户端的请求，转发给B站")
	log.Info().Msg("代理服务启动: http://localhost:8080")
	log.Warn().Msg("不要在生产环境中使用 DEBUG 级别")
	log.Error().Msgf("操作失败: %v", testError)
	log.Error().Err(fmt.Errorf("获取真实流地址失败")).Msg("操作失败")
	log.Err(testError).Msg("操作失败")
	log.Printf("[Error] 获取真实流地址失败: %v", testError)
	// log.Fatal().Err(fmt.Errorf("获取真实流地址失败")).Msg("应用启动失败")

}
