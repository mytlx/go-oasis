#!/bin/bash

DIRS=(
  "cmd/app"
  "internal/router"
  "internal/handler"
  "internal/service"
  "internal/dao"
  "internal/model"
  "internal/config"
  "api"
  "pkg/utils"
  "pkg/config"
  "scripts"
  "static"
  "web"
  "test"
)

for dir in "${DIRS[@]}"; do
  mkdir -p "$dir"
done

echo "所有项目子目录已创建。"
