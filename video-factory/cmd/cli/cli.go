package cli

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"os"
	"video-factory/internal/api"
	"video-factory/internal/db"
	"video-factory/internal/service"
	"video-factory/pkg/config"
	"video-factory/pkg/fetcher"
	"video-factory/pkg/pool"
)

// CliFlags 用于在 CLI 解析后临时存储 Flag 值
type CliFlags struct {
	ConfigFile     string
	Port           int
	BiliCookie     string
	MissevanCookie string
}

func Execute() error {
	// 存储 CLI 解析后的值
	cliValues := CliFlags{}

	app := &cli.App{
		Name:  "Video Factory",
		Usage: "管理多平台直播流",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config-file",
				Aliases:     []string{"c"},
				Usage:       "配置文件 (JSON) 路径",
				Destination: &cliValues.ConfigFile,
				Value:       "./conf/config.json",
			},
			&cli.IntFlag{
				Name:        "port",
				Aliases:     []string{"p"},
				Usage:       "服务监听端口",
				Destination: &cliValues.Port,
				Value:       0, // 使用 0 表示未设置，让 Viper 默认值生效
			},
			&cli.StringFlag{
				Name:        "bili-cookie",
				Usage:       "Bilibili Cookie",
				Destination: &cliValues.BiliCookie,
				Value:       "",
			},
			&cli.StringFlag{
				Name:        "missevan-cookie",
				Usage:       "猫耳 Cookie",
				Destination: &cliValues.MissevanCookie,
				Value:       "",
			},
		},
		Action: start(&cliValues),
	}

	return app.Run(os.Args)
}

func start(cliValues *CliFlags) cli.ActionFunc {
	return func(c *cli.Context) error {
		// 将解析后的命令行值转换为 Viper 键值对，仅设置非空值
		flagMap := make(map[string]interface{})
		if cliValues.Port != 0 {
			flagMap["port"] = cliValues.Port
		}
		if cliValues.BiliCookie != "" {
			flagMap["bili.cookie"] = cliValues.BiliCookie
		}
		if cliValues.MissevanCookie != "" {
			flagMap["missevan.cookie"] = cliValues.MissevanCookie
		}

		// 初始化数据库
		db.InitDB()

		// 加载配置
		configMap, err := service.ListConfigMap()
		if err != nil {
			return err
		}
		if err := config.InitViper(cliValues.ConfigFile, flagMap, configMap); err != nil {
			return err
		}

		// 打印最终配置（用于验证）
		log.Info().Msgf("服务将监听端口: %d", config.GlobalConfig.Port)
		log.Info().Msgf("B站 Cookie 已加载 (长度: %d)", len(config.GlobalConfig.Bili.Cookie))
		log.Info().Msgf("猫耳 Cookie 已加载 (长度: %d)", len(config.GlobalConfig.Missevan.Cookie))

		// ------ 启动应用程序核心逻辑 ------

		// 初始化http客户端
		fetcher.Init(&config.GlobalConfig)
		// 初始化 ManagerPool
		p := pool.NewManagerPool(&config.GlobalConfig)
		// 通过 NewEngine 创建配置好的 Gin 引擎，并将 Pool 注入
		routerEngine := api.NewEngine(p)
		return routerEngine.Run(fmt.Sprintf(":%d", config.GlobalConfig.Port))
	}
}
