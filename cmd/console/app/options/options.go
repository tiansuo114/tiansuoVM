package options

import (
	"tiansuoVM/pkg/client/k8s"
	"tiansuoVM/pkg/client/ldap"
	"tiansuoVM/pkg/client/mysql"
	"tiansuoVM/pkg/logger"
	genericoptions "tiansuoVM/pkg/server/options"
	"tiansuoVM/pkg/vm/controller"

	cliflag "k8s.io/component-base/cli/flag"
)

type ServerRunOptions struct {
	GenericServerRunOptions *genericoptions.ServerRunOptions
	//CacheOptions            *cache.Options
	RDBOptions          *mysql.Options
	LoggerOptions       *logger.Options
	K8sOptions          *k8s.Options
	LDAPOptions         *ldap.Options
	VMControllerOptions *controller.Options

	//K8sNameSpace    string
	//K8sStorageClass string
	DeletedVMRetentionPeriod int
	DebugMode                bool
	JWTSecret                string
	ImageCSVFilePath         string
}

var S ServerRunOptions

const (
	defaultJwtSecret        = "cebc04fc7d4383ebf11bf661ba69977e" // MD5 async-km
	defaultImageCSVFilePath = "configs/os_mirror.csv"
)

func NewServerRunOptions() *ServerRunOptions {
	S = ServerRunOptions{
		GenericServerRunOptions: genericoptions.NewServerRunOptions(),
		//CacheOptions:            cache.NewRedisOptions(),
		RDBOptions:    mysql.NewMysqlOptions(),
		LoggerOptions: logger.NewLoggerOptions(),
		K8sOptions:    k8s.NewKubeOptions(),
		LDAPOptions:   ldap.NewLDAPOptions(),
		VMControllerOptions: &controller.Options{
			Namespace:    "tiansuo-vm",
			SSHPortStart: 30000,
			SSHPortEnd:   32767,
		},
		DeletedVMRetentionPeriod: 7,
		JWTSecret:                defaultJwtSecret,
		ImageCSVFilePath:         defaultImageCSVFilePath,
	}
	return &S
}

func (s *ServerRunOptions) Flags() (fss cliflag.NamedFlagSets) {
	fs := fss.FlagSet("generic")
	fs.BoolVar(&s.DebugMode, "debug", false, "启用调试模式，除非您知道这意味着什么，否则不要启用它。")
	//fs.StringVar(&s.K8sNameSpace, "k8s-namespace", "tiansuo-vm", "Kubernetes集群的命名空间。")
	//fs.StringVar(&s.K8sStorageClass, "k8s-storage-class", "tiansuo-vm-sc", "Kubernetes集群的存储类。")
	fs.StringVar(&s.JWTSecret, "jwt-secret", defaultJwtSecret, "JWT令牌的密钥。")
	fs.IntVar(&s.DeletedVMRetentionPeriod, "deleted-vm-retention-period", 7, "删除的VM保留小时数。")
	fs.StringVar(&s.ImageCSVFilePath, "image-csv-file-path", defaultImageCSVFilePath, "镜像CSV文件路径。")
	s.GenericServerRunOptions.AddFlags(fs)
	//s.CacheOptions.AddFlags(fss.FlagSet("cache"))
	s.RDBOptions.AddFlags(fss.FlagSet("rdb"))
	s.LoggerOptions.AddFlags(fss.FlagSet("log"))
	s.LDAPOptions.AddFlags(fss.FlagSet("ldap"))
	s.K8sOptions.AddFlags(fss.FlagSet("k8s"))

	// VM控制器选项
	vmfs := fss.FlagSet("vm-controller")
	vmfs.StringVar(&s.VMControllerOptions.Namespace, "vm-namespace", s.VMControllerOptions.Namespace, "VM Pod的命名空间")
	vmfs.Int32Var(&s.VMControllerOptions.SSHPortStart, "ssh-port-start", s.VMControllerOptions.SSHPortStart, "SSH端口范围起始值")
	vmfs.Int32Var(&s.VMControllerOptions.SSHPortEnd, "ssh-port-end", s.VMControllerOptions.SSHPortEnd, "SSH端口范围结束值")
	vmfs.Int32Var(&s.VMControllerOptions.ReplicaNum, "replica-num", 1, "VM副本数量")
	vmfs.StringVar(&s.VMControllerOptions.StorageClassName, "storage-class-name", "tiansuo-sc", "VM存储类名")

	return fss
}
