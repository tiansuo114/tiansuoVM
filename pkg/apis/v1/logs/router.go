package logs

import (
	"tiansuoVM/pkg/auth"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/server/middleware"
	"tiansuoVM/pkg/token"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(group *gin.RouterGroup, tokenManager token.Manager, dbResolver *dbresolver.DBResolver) {
	// 初始化日志监听器
	startUserOperatorLogListener(dbResolver)
	startEventLogListener(dbResolver)

	// 创建日志处理器
	handler := newHandler(handlerOption{
		tokenManager: tokenManager,
		dbResolver:   dbResolver,
	})

	// 日志相关API路由组，需要认证和管理员权限
	logsGroup := group.Group("/logs")
	logsGroup.Use(middleware.CheckToken(tokenManager))
	logsGroup.Use(auth.AuthMiddleware())
	logsGroup.Use(auth.AdminRequired())

	// 审计日志相关路由
	logsGroup.GET("/audit", handler.listAuditLogs)

	// 事件日志相关路由
	logsGroup.GET("/events", handler.listEventLogs)
	logsGroup.GET("/events/:id", handler.getEventLogByID)

	// 用户操作日志相关路由
	logsGroup.GET("/user-operations", handler.listUserOperationLogs)
	logsGroup.GET("/user-operations/user/:uid", handler.getUserOperationLogsByUID)
}
