package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/util/homedir"
)

const (
	// 默认值定义
	defaultNamespace    = "async-km"
	defaultStorageClass = "async-km-sc"
	defaultJWTSecret    = "cebc04fc7d4383ebf11bf661ba69977e" // MD5 async-km

	// Server defaults
	defaultBindAddress = "0.0.0.0"
	defaultServerPort  = 9090

	// MySQL defaults
	defaultMySQLHost     = "localhost"
	defaultMySQLPort     = 3306
	defaultMySQLUser     = "root"
	defaultMySQLPassword = "123456"
	defaultMySQLDB       = "async_km"

	// Redis defaults
	defaultRedisDB = 0

	// LDAP defaults
	defaultLDAPHost   = "localhost"
	defaultLDAPPort   = 389
	defaultLDAPUserDN = "cn=admin,dc=example,dc=com"
	defaultLDAPBaseDN = "dc=example,dc=com"

	defauletKubeNameSpace   = "async-km"
	defaultKubeStorageClass = "async-km-sc"
)

var (
	globalConfig *Config
)

// Config 总配置结构体
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Cache  CacheConfig  `mapstructure:"cache"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	LDAP   LDAPConfig   `mapstructure:"ldap"`
	K8s    K8sConfig    `mapstructure:"k8s"`
	Logger LoggerConfig `mapstructure:"logger"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	BindAddress     string `mapstructure:"bind-address"`
	Port            int    `mapstructure:"port"`
	TlsCertFile     string `mapstructure:"tls-cert-file"`
	TlsPrivateKey   string `mapstructure:"tls-private-key"`
	ConfigFilePath  string `mapstructure:"config-file-path"`
	JWTSecret       string `mapstructure:"jwt-secret"`
	DebugMode       bool   `mapstructure:"debug-mode"`
	K8sNameSpace    string `mapstructure:"k8s-namespace"`
	K8sStorageClass string `mapstructure:"k8s-storage-class"`
}

// CacheConfig Redis缓存配置
type CacheConfig struct {
	Host     string `mapstructure:"redis-host"`
	Password string `mapstructure:"redis-password"`
	DB       int    `mapstructure:"redis-db"`
}

// MySQLConfig MySQL数据库配置
type MySQLConfig struct {
	RdbUser     string `mapstructure:"rdb-user"`
	RdbPassword string `mapstructure:"rdb-password"`
	RdbHost     string `mapstructure:"rdb-host"`
	RdbPort     int    `mapstructure:"rdb-port"`
	RdbDbname   string `mapstructure:"rdb-dbname"`
	RdbLogLevel int    `mapstructure:"rdb-log-level"`
}

// LDAPConfig LDAP配置
type LDAPConfig struct {
	Host         string `mapstructure:"ldap-host"`
	Port         int    `mapstructure:"ldap-port"`
	LDAPUserName string `mapstructure:"ldap-user-name"`
	LDAPPassword string `mapstructure:"ldap-password"`
	BaseDN       string `mapstructure:"ldap-base-dn"`
}

// K8sConfig Kubernetes配置
type K8sConfig struct {
	KubeConfigPath   string `mapstructure:"kube-config-path"`
	KubeContext      string `mapstructure:"kube-context"`
	InCluster        bool   `mapstructure:"kube-in-cluster"`
	KubeNameSpace    string `mapstructure:"kube-namespace"`
	KubeStorageClass string `mapstructure:"kube-storage-class"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	LogLevel      int    `mapstructure:"log-level"`
	AddStackLevel int    `mapstructure:"add-stack-level"`
	Filename      string `mapstructure:"log-filename"`
	MaxSize       int    `mapstructure:"log-max-size"`
	MaxBackups    int    `mapstructure:"log-max-backups"`
	MaxAge        int    `mapstructure:"log-max-age"`
	Compress      bool   `mapstructure:"log-compress"`
}

func GetGlobalConfig() *Config {
	return globalConfig
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig() *Config {
	// 获取默认的kubeconfig路径
	defaultKubeConfig := ""
	if home := homedir.HomeDir(); home != "" {
		defaultKubeConfig = filepath.Join(home, ".kube", "config")
	}

	return &Config{
		Server: ServerConfig{
			BindAddress:     defaultBindAddress,
			Port:            defaultServerPort,
			JWTSecret:       defaultJWTSecret,
			K8sNameSpace:    defaultNamespace,
			K8sStorageClass: defaultStorageClass,
			DebugMode:       false,
		},
		Cache: CacheConfig{
			Host:     "", // 默认为空,表示不启用Redis
			Password: "",
			DB:       defaultRedisDB,
		},
		MySQL: MySQLConfig{
			RdbHost:     defaultMySQLHost,
			RdbPort:     defaultMySQLPort,
			RdbUser:     defaultMySQLUser,
			RdbPassword: defaultMySQLPassword,
			RdbDbname:   defaultMySQLDB,
			RdbLogLevel: 1,
		},
		LDAP: LDAPConfig{
			Host:         defaultLDAPHost,
			Port:         defaultLDAPPort,
			LDAPUserName: defaultLDAPUserDN,
			BaseDN:       defaultLDAPBaseDN,
		},
		K8s: K8sConfig{
			KubeConfigPath:   defaultKubeConfig,
			InCluster:        false,
			KubeNameSpace:    defauletKubeNameSpace,
			KubeStorageClass: defaultKubeStorageClass,
		},
		Logger: LoggerConfig{
			LogLevel:      int(zap.DebugLevel),
			AddStackLevel: int(zap.FatalLevel),
			MaxSize:       1,
			MaxBackups:    500,
			MaxAge:        180,
			Compress:      false,
		},
	}
}

// ParseConfigFile 从YAML文件解析配置
func ParseConfigFile(configFile string) error {
	// 设置默认配置
	globalConfig = NewDefaultConfig()

	// 使用file.open函数并尝试打印出文件内容,以测试该文件路径是否能被正常读取
	if file, err := os.Open(configFile); err == nil {
		defer func() {
			_ = file.Close()
		}()
		zap.L().Debug("config file content", zap.String("configFile", configFile))
		data, _ := io.ReadAll(file)
		zap.L().Debug("config file content", zap.String("configFile", string(data)))
	}

	// 读取配置文件
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	// 支持环境变量覆盖,环境变量中的-和.会被转换为_
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			zap.L().Error("failed to find config file", zap.String("configFile", configFile))
			return err
		}
		return fmt.Errorf("config file %q loading error: %s", configFile, err)
	}

	// 将配置解析到结构体
	if err := viper.Unmarshal(globalConfig); err != nil {
		return err
	}

	zap.L().Debug("load config ok", zap.Any("config", globalConfig))
	return nil
}

// ValidateConfig 验证配置
func ValidateConfig(cfg *Config) []error {
	var errs []error

	// 验证Server配置
	if cfg.Server.Port < 0 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("invalid server port"))
	}

	// 验证TLS配置
	if cfg.Server.TlsPrivateKey != "" && cfg.Server.TlsCertFile != "" {
		if _, err := os.Stat(cfg.Server.TlsCertFile); err != nil {
			errs = append(errs, err)
		}
		if _, err := os.Stat(cfg.Server.TlsPrivateKey); err != nil {
			errs = append(errs, err)
		}
	}

	// 验证Redis配置
	if cfg.Cache.DB < 0 || cfg.Cache.DB > 15 {
		errs = append(errs, fmt.Errorf("invalid redis db"))
	}

	// 验证MySQL配置
	if cfg.MySQL.RdbPort < 0 || cfg.MySQL.RdbPort > 65535 {
		errs = append(errs, fmt.Errorf("invalid mysql port"))
	}

	// 验证LDAP配置
	if cfg.LDAP.Port < 0 || cfg.LDAP.Port > 65535 {
		errs = append(errs, fmt.Errorf("invalid ldap port"))
	}

	// 验证日志配置
	if cfg.Logger.LogLevel < int(zap.DebugLevel) || cfg.Logger.LogLevel > int(zap.FatalLevel) {
		errs = append(errs, fmt.Errorf("invalid log level"))
	}
	if cfg.Logger.AddStackLevel < int(zap.DebugLevel) || cfg.Logger.AddStackLevel > int(zap.FatalLevel) {
		errs = append(errs, fmt.Errorf("invalid add stack level"))
	}

	return errs
}
