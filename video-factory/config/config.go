package config

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

// AppConfig 包含应用程序的所有配置项
type AppConfig struct {
	Port int `json:"port"` // 监听端口
	Bili struct {
		Cookie string `json:"cookie"` // B站 Cookie
	} `json:"bili"`
	Missevan struct {
		Cookie string `json:"cookie"` // 猫耳 Cookie
	} `json:"missevan"`
}

// GlobalConfig 存储加载后的配置实例
var GlobalConfig AppConfig

// InitViper 负责 Viper 的初始化、加载和反序列化
func InitViper(configFilePath string, cmdFlags map[string]interface{}) error {
	v := viper.New()

	// 用于标记是否成功读取了配置文件
	configReadSuccess := false

	// 1. 设置默认值 (最低优先级)
	v.SetDefault("port", 8090)
	v.SetDefault("bili.cookie", "")
	v.SetDefault("missevan.cookie", "")

	// 2. 配置并读取配置文件 (次低优先级)
	if configFilePath != "" {
		v.SetConfigFile(configFilePath) // 从命令行指定的路径加载
	} else {
		// 如果未指定路径，则设置默认搜索路径和文件名
		v.SetConfigName("config")  // 文件名（无扩展名）
		v.SetConfigType("json")    // 文件类型
		v.AddConfigPath("./conf/") // 搜索目录
		v.AddConfigPath("$HOME/.config/video-factory/")
	}

	// 3. 尝试读取配置文件。如果文件不存在，不返回错误，使用默认值
	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			// 文件存在，但格式错误等其他错误，则返回
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 如果是文件找不到，则忽略，使用默认值
		log.Info().Msg("未找到配置文件，使用默认值和命令行参数。")
	} else {
		// 文件读取成功
		configReadSuccess = true
		log.Printf("成功加载配置文件: %s", v.ConfigFileUsed())
	}

	// 4. 绑定命令行 Flag (最高优先级)
	for key, value := range cmdFlags {
		v.Set(key, value)
	}

	// 5. 将配置反序列化到结构体
	if err := v.Unmarshal(&GlobalConfig); err != nil {
		log.Fatal().Err(err).Msg("反序列化配置失败")
	}

	// 6. **持久化逻辑**：如果未读取到文件，则创建并写入文件
	if !configReadSuccess {
		// 确定要写入的最终文件路径
		writePath := configFilePath
		if writePath == "" {
			// 如果命令行未指定，则使用默认路径（这里简单地写到当前目录）
			// 在生产环境中，推荐使用 v.AddConfigPath 中某个路径，例如 $HOME/.config/...
			writePath = "./conf/config.json"
		}

		// 写入配置
		if err := safeWriteConfig(v, writePath); err != nil {
			// 注意：这里写入失败不应该导致程序退出，只是打印警告
			log.Warn().Msgf("警告：无法创建或写入配置文件 %s: %v", writePath, err)
		} else {
			log.Info().Msgf("配置文件 %s 已基于默认值和命令行参数生成。", writePath)
		}
	}

	return nil
}

// safeWriteConfig 确保目录存在，并使用 SafeWriteConfigAs 安全地写入配置
func safeWriteConfig(v *viper.Viper, filePath string) error {
	// 1. 确保目标目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 2. SafeWriteConfigAs 只有在文件不存在时才写入，避免覆盖用户可能已手动创建的文件
	// 如果您确定要覆盖，请使用 v.WriteConfigAs(filePath)
	return v.SafeWriteConfigAs(filePath)
}
