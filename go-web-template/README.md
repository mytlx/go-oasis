# go-web-template

gin + gorm + viper + swagger 的一个 web 项目模版

目录结构:
```
go-web-template/
├── cmd/                    # 启动程序（入口）目录
│   └── app/                # 例如：main.go 在这里
│       └── main.go
├── internal/               # 内部模块，避免被外部导入
│   ├── router/             # 路由注册逻辑
│   ├── handler/            # HTTP 处理器（Controller 层）
│   ├── service/            # 业务逻辑层（Service 层）
│   ├── dao/                # 数据访问层（数据库相关）
│   ├── model/              # 数据模型，含 GORM 结构体
│   └── config/             # 内部配置读取与定义
├── api/                    # API 定义（如 OpenAPI/Swagger 文件）
├── pkg/                    # 可复用的通用库（可被多个项目使用）
│   ├── utils/              # 通用工具包（字符串、时间、UUID 等）
│   ├── logger/             # 日志包
│   ├── errcode/            # 错误码定义和封装
│   └── config/             # 通用配置
├── scripts/                # 启动脚本或构建脚本
├── static/                 # 静态文件（如前端资源）
├── web/                    # 前端项目目录（如 vue, react）
├── test/                   # 测试代码
├── docs/                   # Swagger 文档
├── go.mod
├── go.sum
└── README.md
```

生成目录命令：
```bash
mkdir -p cmd/app internal/{router,handler,service,dao,model,config} api pkg/{utils,errcode,config} scripts static web test
```