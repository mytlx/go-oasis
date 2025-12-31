#!/bin/bash

# 定义构建参数
TIMESTAMP=$(date +"%Y%m%d")
OUTPUT_NAME="C:/Users/TLX/Desktop/temp/video-factory_${TIMESTAMP}.exe"
TARGET_PACKAGE="../cmd/app"

echo "正在构建当前平台可执行文件..."

# 执行构建
# 注意：在 shell 中，将 ldflags 作为一个整体字符串传递时，引号的处理可能比较棘手。
# 最可靠的方式是将 ldflags 作为一个整体变量传入：
go build -ldflags="-s -w -linkmode=external" -o "$OUTPUT_NAME" "$TARGET_PACKAGE"

# 检查上一个命令的退出状态
if [ $? -eq 0 ]; then
    echo ""
    echo "----------------------------------------------------"
    echo "构建成功! 文件名为: $OUTPUT_NAME"
    echo "----------------------------------------------------"
else
    echo ""
    echo "----------------------------------------------------"
    echo "错误：构建失败。"
    echo "----------------------------------------------------"
fi