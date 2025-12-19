package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"video-factory/internal/iface"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// AppConfig 包含应用程序的所有配置项
type AppConfig struct {
	Viper       *viper.Viper `json:"-" mapstructure:"-"`
	subscribers []iface.ConfigSubscriber

	Port       int    `json:"port" mapstructure:"port"`                 // 监听端口
	GinLogMode string `json:"gin_log_mode" mapstructure:"gin_log_mode"` // gin 日志模式
	Proxy      struct {
		Enabled     bool   `json:"enabled" mapstructure:"enabled"`           // 是否启用 HTTP 代理
		SystemProxy bool   `json:"system_proxy" mapstructure:"system_proxy"` // 是否使用系统代理
		Protocol    string `json:"protocol" mapstructure:"protocol"`         // 代理协议
		Host        string `json:"host" mapstructure:"host"`                 // 代理主机
		Port        int    `json:"port" mapstructure:"port"`                 // 代理端口
		Username    string `json:"username" mapstructure:"username"`
		Password    string `json:"password" mapstructure:"password"`
	} `json:"proxy" mapstructure:"proxy"`
	Bili struct {
		Cookie string `json:"cookie" mapstructure:"cookie"` // B站 Cookie
	} `json:"bili"`
	Missevan struct {
		Cookie string `json:"cookie" mapstructure:"cookie"` // 猫耳 Cookie
	} `json:"missevan" mapstructure:"missevan"`
	Recorder *Recorder `json:"recorder" mapstructure:"recorder"`
}

type Recorder struct {
	FilenamePattern string `json:"filename_pattern" mapstructure:"filename_pattern"` // 文件名格式
	MaxFilesize     int    `json:"max_filesize" mapstructure:"max_filesize"`         // 最大文件大小
	MaxDuration     int    `json:"max_duration" mapstructure:"max_duration"`         // 最大录制时长
}

// GlobalConfig 存储加载后的配置实例
var GlobalConfig AppConfig

// MarshalZerologObject 实现 zerolog 接口，用于高效且安全地打印日志
func (config *AppConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Int("port", config.Port).
		Str("gin_log_mode", config.GinLogMode)

	// 使用 Dict 嵌套打印 Proxy 信息
	e.Dict("proxy", zerolog.Dict().
		Bool("enabled", config.Proxy.Enabled).
		Bool("system_proxy", config.Proxy.SystemProxy).
		Str("protocol", config.Proxy.Protocol).
		Str("host", config.Proxy.Host).
		Int("port", config.Proxy.Port).
		Str("username", config.Proxy.Username).
		Str("password", maskSecret(config.Proxy.Password)))

	// 嵌套打印 Bili 信息
	e.Dict("bili", zerolog.Dict().
		Str("cookie", config.Bili.Cookie))

	// 嵌套打印 Missevan 信息
	e.Dict("missevan", zerolog.Dict().
		Str("cookie", config.Missevan.Cookie))

	e.Dict("recorder", zerolog.Dict().
		Str("filename_pattern", config.Recorder.FilenamePattern).
		Str("max_filesize", strconv.Itoa(config.Recorder.MaxFilesize)).
		Str("max_duration", strconv.Itoa(config.Recorder.MaxDuration)),
	)
}

func (config *AppConfig) AddSubscriber(subscriber iface.ConfigSubscriber) {
	config.subscribers = append(config.subscribers, subscriber)
	log.Info().Msgf("[Config] 订阅者注册成功")
}

func (config *AppConfig) OnUpdate(key string, value string) error {
	log.Info().Msgf("[Config] 更新配置, key: %s, value: %s", key, value)
	config.Viper.Set(key, value)
	if err := config.Viper.Unmarshal(&GlobalConfig); err != nil {
		log.Error().Err(err).Msgf("[config] 反序列化更新失败, key: %s", key)
		return fmt.Errorf("反序列化更新失败: %w", err)
	}

	// 通知所有订阅者
	log.Info().Msgf("[config] 通知订阅者, key: %s, value: %s", key, value)
	for _, subscriber := range config.subscribers {
		subscriber.OnConfigUpdate(key, value)
	}
	log.Info().Msgf("[config] 配置更新成功: %s = %v", key, value)
	log.Warn().Object("config", &GlobalConfig).Msg("[config] 配置更新成功")
	return nil
}

// InitViper 负责 Viper 的初始化、加载和反序列化
func InitViper(configFilePath string, cmdFlags map[string]interface{}, configMap map[string]string) error {
	v := viper.New()

	// 1. 设置默认值 (最低优先级)
	v.SetDefault("port", 8090)
	v.SetDefault("bili.cookie", "")
	v.SetDefault("missevan.cookie", "")

	// 从数据库加载配置
	for key, value := range configMap {
		v.SetDefault(key, value) // 注意：这里使用 SetDefault
	}

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
		// fmt.Printf("原始错误类型：%T\n", err)
		// fmt.Printf("原始错误信息：%v\n", err)
		if !os.IsNotExist(err) {
			// 文件存在，但格式错误等其他错误，则返回
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 如果是文件找不到，则忽略，使用默认值
		log.Info().Msg("未找到配置文件，使用[默认值|数据库配置|命令行参数]")
	} else {
		// 文件读取成功
		log.Printf("成功加载配置文件: %s", v.ConfigFileUsed())
	}

	// 4. 绑定命令行 Flag (最高优先级)
	for key, value := range cmdFlags {
		v.Set(key, value)
	}

	// 打印 Viper 对该键的最终解析值
	// fmt.Println("Viper 最终解析 proxy.system_proxy 的值:", v.GetBool("proxy.system_proxy"))
	// fmt.Println("Viper 最终解析 proxy.system_proxy 的类型:", v.Get("proxy.system_proxy"))

	// 5. 将配置反序列化到结构体
	if err := v.Unmarshal(&GlobalConfig); err != nil {
		log.Fatal().Err(err).Msg("反序列化配置失败")
	}

	log.Warn().Object("config", &GlobalConfig).Msg("[config] 配置加载完成")

	GlobalConfig.Viper = v
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

// UnflattenConfig 将扁平化的 map[string]string 数据映射到嵌套的结构体中。
// configPtr 必须是一个指向 AppConfig 结构体的指针。
func UnflattenConfig(configPtr interface{}, configMap map[string]string) error {
	if configPtr == nil || configMap == nil {
		return nil
	}
	// 确保传入的是指针
	val := reflect.ValueOf(configPtr)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("configPtr 必须是非空的结构体指针")
	}

	// 获取指针指向的实际结构体
	s := val.Elem()
	if s.Kind() != reflect.Struct {
		return fmt.Errorf("configPtr 必须指向一个结构体")
	}

	for key, value := range configMap {
		// key 是扁平化路径，例如 "proxy.host"
		parts := strings.Split(key, ".")

		// 从根结构体开始遍历
		currentValue := s

		// 存储成功找到的字段
		var field reflect.Value

		// 遍历路径的每个部分 (例如 "proxy", "host")
		for i, part := range parts {
			// 1. 查找结构体字段（使用 Tag 或字段名）
			// 这里为了简单，我们使用 Tag（json tag）进行匹配
			found := false
			for j := 0; j < currentValue.NumField(); j++ {
				fieldInfo := currentValue.Type().Field(j)
				tag := fieldInfo.Tag.Get("json")

				// 匹配 Tag (如果 Tag 存在) 或字段名 (如果 Tag 为空)
				if tag == part || fieldInfo.Name == part {
					field = currentValue.Field(j)
					found = true
					break
				}
			}

			if !found {
				// 如果路径中间找不到字段，则跳过这个配置项
				// 实际项目中应记录日志
				log.Warn().Msgf("WARN: Config key '%s' part '%s' not found\n", key, part)
				break
			}

			// 如果是最后一个部分，我们进行值设置
			if i == len(parts)-1 {
				if err := setFieldValue(field, value); err != nil {
					return fmt.Errorf("failed to set value for %s: %w", key, err)
				}
			} else {
				// 如果不是最后一个部分，继续遍历下一个嵌套结构体
				if field.Kind() == reflect.Struct {
					currentValue = field
				} else {
					// 路径未结束，但字段不是结构体，则跳过
					break
				}
			}
		}
	}
	return nil
}

// setFieldValue 负责将字符串值转换为目标字段的类型并设置
func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.String:
		field.SetString(value)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// maskSecret 简单的脱敏辅助函数
func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 6 {
		return "******"
	}
	// 只显示前2位和后2位
	return s[:2] + "******" + s[len(s)-2:]
}
