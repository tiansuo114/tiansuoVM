package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"tiansuoVM/pkg/apis/v1/image"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/term"

	"tiansuoVM/cmd/console/app/options"
	"tiansuoVM/pkg/apis/v1/admin"
	"tiansuoVM/pkg/apis/v1/logs"
	"tiansuoVM/pkg/apis/v1/user"
	"tiansuoVM/pkg/apis/v1/vm"
	"tiansuoVM/pkg/logger"
	"tiansuoVM/pkg/server"
	"tiansuoVM/pkg/server/config"
	"tiansuoVM/pkg/server/middleware"
	"tiansuoVM/pkg/version/verflag"
)

// NewAPIServerCommand 创建API服务器命令
func NewAPIServerCommand() *cobra.Command {
	s := options.NewServerRunOptions()
	cmd := &cobra.Command{
		Use:   "console",
		Short: "启动天锁虚拟机管理API服务器",
		Long:  `天锁虚拟机管理API服务器提供REST API端点，用于管理虚拟机和相关资源。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()

			if errs := s.Validate(); len(errs) != 0 {
				return utilerrors.NewAggregate(errs)
			}

			if err := config.ParseConfigFile(s.GenericServerRunOptions.ConfigFilePath); err != nil {
				return err
			}

			return Run(s, server.SetupSignalHandler())
		},
		SilenceUsage: true,
	}

	fs := cmd.Flags()
	namedFlagSets := s.Flags()

	// 添加全局标志
	globalFlagSet := namedFlagSets.FlagSet("global")
	globalFlagSet.BoolP("help", "h", false, fmt.Sprintf("help for %s", cmd.Name()))

	// 添加所有标志集
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	// 设置使用说明
	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cliflag.SetUsageAndHelpFunc(cmd, namedFlagSets, cols)

	return cmd
}

// Run 运行服务器
func Run(options *options.ServerRunOptions, stopCh <-chan struct{}) error {
	// 创建并初始化服务器
	s, err := NewConsoleServer(options, stopCh)
	if err != nil {
		return err
	}

	// 准备运行
	if err := s.PrepareRun(stopCh); err != nil {
		return err
	}

	// 启动服务器
	return s.Run(stopCh)
}

// PrepareRun 准备运行服务器
func (s *ConsoleServer) PrepareRun(stopCh <-chan struct{}) error {
	// 设置Gin模式
	if options.S.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	s.router = gin.New()
	s.router.ContextWithFallback = true
	s.router.Use(gin.Recovery())
	s.router.Use(logger.GinLogger())
	s.router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	}))

	// 初始化系统
	if err := s.initSystem(); err != nil {
		zap.L().Panic("初始化系统失败", zap.Error(err))
	}

	// 安装API
	s.installAPIs()
	s.Server.Handler = s.router

	return nil
}

// Run 运行服务器
func (s *ConsoleServer) Run(stopCh <-chan struct{}) error {
	// 启动VM控制器
	if err := s.VMController.Start(context.Background()); err != nil {
		return fmt.Errorf("启动VM控制器失败: %w", err)
	}

	// 启动HTTP服务器
	go func() {
		if err := s.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Panic("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	zap.L().Info("服务器已启动", zap.String("地址", s.Server.Addr))

	// 等待停止信号
	<-stopCh
	zap.L().Info("正在关闭服务器...")

	// 创建关闭上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 停止VM控制器
	s.VMController.Stop()

	// 优雅关闭HTTP服务器
	if err := s.Server.Shutdown(ctx); err != nil {
		zap.L().Error("服务器被强制关闭", zap.Error(err))
		return err
	}

	// 关闭数据库连接
	if err := s.DBResolver.Close(); err != nil {
		zap.L().Error("关闭数据库连接失败", zap.Error(err))
	}

	zap.L().Info("服务器已退出")
	return nil
}

// installAPIs 安装API路由
func (s *ConsoleServer) installAPIs() {
	apiV1Group := s.router.Group("/api/v1")
	apiV1Group.Use(middleware.AddAuditLog(s.DBResolver))

	// 注册各模块路由
	admin.RegisterRoutes(apiV1Group, s.TokenManager, s.DBResolver, s.LDAPClient)
	logs.RegisterRoutes(apiV1Group, s.TokenManager, s.DBResolver)
	user.RegisterRoutes(apiV1Group, s.TokenManager, s.DBResolver, s.LDAPClient)
	vm.RegisterRoutes(apiV1Group, s.TokenManager, s.DBResolver, s.VMController)
	image.RegisterRoutes(apiV1Group, s.TokenManager, s.DBResolver)
}
