package config

import "tiansuoVM/cmd/console/app/options"

// 将全局配置转换为服务器运行选项
func ConfigToServerRunOptions(cfg *Config) *options.ServerRunOptions {
	opts := options.NewServerRunOptions()

	// 转换通用服务器配置
	opts.GenericServerRunOptions.BindAddress = cfg.Server.BindAddress
	opts.GenericServerRunOptions.Port = cfg.Server.Port
	opts.GenericServerRunOptions.TlsCertFile = cfg.Server.TlsCertFile
	opts.GenericServerRunOptions.TlsPrivateKey = cfg.Server.TlsPrivateKey
	opts.GenericServerRunOptions.ConfigFilePath = cfg.Server.ConfigFilePath

	//// 转换缓存配置
	//opts.CacheOptions.Host = cfg.Cache.Host
	//opts.CacheOptions.Password = cfg.Cache.Password
	//opts.CacheOptions.DB = cfg.Cache.DB

	// 转换数据库配置
	opts.RDBOptions.RdbUser = cfg.MySQL.RdbUser
	opts.RDBOptions.RdbPassword = cfg.MySQL.RdbPassword
	opts.RDBOptions.RdbHost = cfg.MySQL.RdbHost
	opts.RDBOptions.RdbPort = cfg.MySQL.RdbPort
	opts.RDBOptions.RdbDbname = cfg.MySQL.RdbDbname
	opts.RDBOptions.RdbLogLevel = cfg.MySQL.RdbLogLevel

	// 转换日志配置
	opts.LoggerOptions.LogLevel = cfg.Logger.LogLevel
	opts.LoggerOptions.AddStackLevel = cfg.Logger.AddStackLevel
	opts.LoggerOptions.Filename = cfg.Logger.Filename
	opts.LoggerOptions.MaxSize = cfg.Logger.MaxSize
	opts.LoggerOptions.MaxBackups = cfg.Logger.MaxBackups
	opts.LoggerOptions.MaxAge = cfg.Logger.MaxAge
	opts.LoggerOptions.Compress = cfg.Logger.Compress

	// 转换Kubernetes配置
	opts.K8sOptions.KubeConfigPath = cfg.K8s.KubeConfigPath
	opts.K8sOptions.InCluster = cfg.K8s.InCluster
	//opts.KubevirtOptions.KubeConfigPath = cfg.K8s.KubeConfigPath
	//opts.KubevirtOptions.InCluster = cfg.K8s.InCluster

	// 转换LDAP配置
	opts.LDAPOptions.Host = cfg.LDAP.Host
	opts.LDAPOptions.Port = cfg.LDAP.Port
	opts.LDAPOptions.LDAPUserName = cfg.LDAP.LDAPUserName
	opts.LDAPOptions.LDAPPassword = cfg.LDAP.LDAPPassword
	opts.LDAPOptions.BaseDN = cfg.LDAP.BaseDN

	// 转换服务器选项
	//opts.K8sNameSpace = cfg.Server.K8sNameSpace
	//opts.K8sStorageClass = cfg.Server.K8sStorageClass
	opts.DebugMode = cfg.Server.DebugMode
	opts.JWTSecret = cfg.Server.JWTSecret

	return opts
}
