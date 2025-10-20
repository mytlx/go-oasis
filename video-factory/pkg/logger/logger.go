package logger

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"strings"
	"time"
)

// InitLogger 初始化 zerolog，实现类似 Spring Boot 的控制台格式
func InitLogger() {
	// 设置全局日志级别 (例如：Debug 或 info)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// 禁用默认的时间戳字段名 (默认为 "time")
	zerolog.TimestampFieldName = "timestamp"
	// 设置时间格式，与 Spring Boot 默认的 ISO 8601 兼容
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// 配置 ConsoleWriter 以输出人类可读的格式
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,                 // 默认输出到标准错误
		NoColor:    false,                     // 启用颜色，让日志更醒目
		TimeFormat: "2006-01-02 15:04:05.000", // 自定义日期时间格式
		// 固定长度的日志级别 [LEVEL]
		FormatLevel: func(i interface{}) string {
			// 将级别转为大写，并使用 fmt.Sprintf 进行右侧填充，宽度为 7
			levelStr := strings.ToUpper(i.(string))
			return fmt.Sprintf(" %5s ", levelStr)
		},
		// 固定长度的调用方信息 (文件名:行号)
		FormatCaller: func(i interface{}) string {
			// i 是 "file:line" 格式。我们希望它靠左对齐，例如固定宽度 20
			callerStr := i.(string)
			// 找到最后一个 / 后的文件名
			if lastSlash := strings.LastIndexByte(callerStr, '/'); lastSlash != -1 {
				callerStr = callerStr[lastSlash+1:]
			}
			return fmt.Sprintf("%-25s", callerStr)
		},
		// 关键：自定义输出格式，模拟 Spring Boot 的字段顺序
		FormatMessage: func(i interface{}) string {
			return fmt.Sprintf(" : %s", i.(string))
		},
	}

	log.Logger = zerolog.New(consoleWriter).
		Level(zerolog.DebugLevel).
		With().
		Timestamp().
		CallerWithSkipFrameCount(2). // 设置跳过帧数，以正确显示调用代码的文件名
		Logger()

}
