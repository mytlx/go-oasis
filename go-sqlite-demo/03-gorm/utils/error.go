package utils

import "log"

// HandleError 通用的错误处理方法
func HandleError(err error, format string, args ...any) {
	if err != nil {
		log.Printf("错误: "+format+",\n错误信息: %v\n", append(args, err)...)
	}
}
