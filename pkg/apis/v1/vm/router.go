package vm

import (
	"github.com/gin-gonic/gin"
	"tiansuoVM/pkg/auth"
	"tiansuoVM/pkg/server/middleware"

	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/token"
	"tiansuoVM/pkg/vm/controller"
)

// RegisterRoutes 注册VM相关的路由
func RegisterRoutes(router *gin.RouterGroup, tokenManager token.Manager, dbResolver *dbresolver.DBResolver, vmController *controller.Controller) {
	h := newHandler(handlerOption{
		tokenManager: tokenManager,
		dbResolver:   dbResolver,
		vmController: vmController,
	})

	vmGroup := router.Group("/vms")
	vmGroup.Use(middleware.CheckToken(tokenManager))
	vmGroup.Use(auth.AuthMiddleware())

	{
		// 获取VM列表
		vmGroup.GET("", h.listVMs)
		// 创建VM
		vmGroup.POST("", h.createVM)
		// 获取VM详情
		vmGroup.GET("/:id", h.getVM)
		// 标记删除VM
		vmGroup.DELETE("/:id", h.deleteVM)
		// 恢复已标记删除的VM
		vmGroup.POST("/:id/recover", h.recoverVM)
		// 启动VM
		vmGroup.POST("/:id/start", h.startVM)
		// 停止VM
		vmGroup.POST("/:id/stop", h.stopVM)
		// 重启VM
		vmGroup.POST("/:id/restart", h.restartVM)
	}

	adminGroup := router.Group("admin/vms")
	adminGroup.Use(middleware.CheckToken(tokenManager))
	adminGroup.Use(auth.AdminRequired())
	vmGroup.Use(auth.AuthMiddleware())
	{
		// 创建VM
		adminGroup.POST("", h.adminCreateVM)
	}
}
