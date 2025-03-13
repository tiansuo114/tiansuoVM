package app

import (
	"fmt"
	"net/http"
	"tiansuoVM/pkg/helper"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"tiansuoVM/cmd/console/app/options"
	"tiansuoVM/pkg/client/cache"
	"tiansuoVM/pkg/client/k8s"
	"tiansuoVM/pkg/client/ldap"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/token"
	"tiansuoVM/pkg/vm/controller"
	"tiansuoVM/pkg/vm/service"
)

// ConsoleServer 控制台服务器
type ConsoleServer struct {
	Server *http.Server
	router *gin.Engine

	// 核心组件
	TokenManager token.Manager
	DBResolver   *dbresolver.DBResolver
	CacheClient  cache.Interface

	// 客户端
	K8sClient  *k8s.KubeClient
	LDAPClient *ldap.LDAPClient

	// 控制器
	VMController  *controller.Controller
	VMCleaner     *service.VMCleaner
	ImageImporter *service.ImageImporter
}

// NewConsoleServer 创建新的控制台服务器
func NewConsoleServer(opts *options.ServerRunOptions, stopCh <-chan struct{}) (*ConsoleServer, error) {
	// 初始化数据库
	dbResolver, err := dbresolver.NewDBResolver(opts.RDBOptions)
	if err != nil {
		return nil, fmt.Errorf("创建数据库解析器失败: %w", err)
	}

	// 初始化缓存
	var cacheClient cache.Interface
	//if opts.CacheOptions.Host != "" {
	//	cacheClient, err = cache.NewRedisClient(opts.CacheOptions, nil)
	//	if err != nil {
	//		return nil, fmt.Errorf("创建缓存客户端失败: %w", err)
	//	}
	//} else {
	//	cacheClient = cache.NewMemoryClient()
	//}

	cacheClient = cache.NewSimpleCache()

	// 初始化K8s客户端
	k8sClient, err := k8s.NewKubeClient(opts.K8sOptions)
	if err != nil {
		return nil, fmt.Errorf("创建K8s客户端失败: %w", err)
	}

	ldapClient, err := ldap.NewLDAPClient(opts.LDAPOptions)
	if err != nil {
		return nil, fmt.Errorf("创建LDAP客户端失败: %w", err)
	}

	// 设置VM控制器选项
	vmOpts := opts.VMControllerOptions
	vmOpts.KubeOpts = opts.K8sOptions
	vmOpts.DatabaseOpts = dbResolver

	// 初始化VM控制器
	vmController, err := controller.NewController(vmOpts)
	if err != nil {
		return nil, fmt.Errorf("创建VM控制器失败: %w", err)
	}
	VMCleaner := service.NewVMCleaner(dbResolver, vmController, opts.DeletedVMRetentionPeriod)

	ImageImporter := service.NewImageImporter(dbResolver, helper.GetRootByCaller(), opts.ImageCSVFilePath)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", opts.GenericServerRunOptions.BindAddress, opts.GenericServerRunOptions.Port),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
	}

	// 创建控制台服务器
	s := &ConsoleServer{
		Server:        server,
		TokenManager:  token.NewJWTTokenManager([]byte(opts.JWTSecret), jwt.SigningMethodHS256, token.SetDuration(cacheClient, time.Minute*30)),
		DBResolver:    dbResolver,
		CacheClient:   cacheClient,
		LDAPClient:    ldapClient,
		K8sClient:     k8sClient,
		VMController:  vmController,
		VMCleaner:     VMCleaner,
		ImageImporter: ImageImporter,
	}

	return s, nil
}
