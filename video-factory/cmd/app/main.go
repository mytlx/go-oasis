package main

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"syscall"
	"unsafe"
	"video-factory/cmd/cli"
	"video-factory/pkg/logger"
)

const ConsoleTitle = "video-factory"

// Windows API 函数 SetConsoleTitle 的定义
// https://learn.microsoft.com/en-us/windows/console/setconsoletitle
var (
	kernel32        = syscall.MustLoadDLL("kernel32.dll")
	setConsoleTitle = kernel32.MustFindProc("SetConsoleTitleW")
)

// setConsoleTitleW 调用函数
func setWinTitle(title string) (err error) {
	if setConsoleTitle != nil {
		// 将 Go 字符串转换为 UTF-16 编码的指针，这是 Windows API 所需要的
		ptr, err := syscall.UTF16PtrFromString(title)
		if err != nil {
			return err
		}

		// 调用 SetConsoleTitleW
		ret, _, callErr := setConsoleTitle.Call(uintptr(unsafe.Pointer(ptr)))

		// 检查返回码
		if ret == 0 {
			if callErr != nil {
				return callErr
			}
			// 某些情况下 ret=0 但 callErr=nil，表示调用失败
			return fmt.Errorf("SetConsoleTitle failed")
		}
	}
	return nil
}

func main() {
	// 打包命令 go build -ldflags="-s -w -linkmode=external" -o "video-factory.exe" ./cmd/app

	// 仅在 Windows 平台上设置标题
	if os.Getenv("GOOS") == "" || os.Getenv("GOOS") == "windows" {
		err := setWinTitle(ConsoleTitle)
		if err != nil {
			log.Err(err).Msg("设置标题失败")
		}
	}

	// 1. 设置日志格式/系统
	logger.InitLogger()

	// 2. 启动 CLI 应用和配置加载 (核心逻辑)
	if err := cli.Execute(); err != nil {
		// 所有的配置加载、CLI 解析错误都在这里捕获
		log.Fatal().Err(err).Msg("应用启动失败")
	}
}
