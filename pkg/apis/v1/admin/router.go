package admin

import (
	"github.com/gin-gonic/gin"
	"tiansuoVM/pkg/auth"

	"tiansuoVM/pkg/client/ldap"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/server/middleware"
	"tiansuoVM/pkg/token"
)

// RegisterRoutes 注册管理员相关路由
func RegisterRoutes(router *gin.RouterGroup, tokenManager token.Manager, dbResolver *dbresolver.DBResolver, ldapClient *ldap.LDAPClient) {
	handler := newHandler(handlerOption{
		tokenManager: tokenManager,
		dbResolver:   dbResolver,
		ldapClient:   ldapClient,
	})

	// 所有管理员路由都需要认证和管理员权限
	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.CheckToken(tokenManager))
	adminGroup.Use(auth.AuthMiddleware())
	adminGroup.Use(auth.AdminRequired())

	// 用户管理
	userGroup := adminGroup.Group("/users")
	{
		userGroup.GET("", handler.listUsers)
		userGroup.PUT("/:uid", handler.updateUser)
		userGroup.DELETE("/:uid", handler.deleteUser)
		userGroup.POST("/role", handler.setUserRole)
		userGroup.GET("/admins", handler.getAdminUsers)
	}

	// 虚拟机管理
	vmGroup := adminGroup.Group("/vms")
	{
		vmGroup.GET("", handler.listVMs)
	}
}
