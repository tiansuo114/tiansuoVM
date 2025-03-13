package user

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"tiansuoVM/pkg/auth"
	"time"

	"tiansuoVM/pkg/client/ldap"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/server/middleware"
	"tiansuoVM/pkg/token"
	"tiansuoVM/pkg/utils/limiter"
)

// RegisterRoutes 注册用户相关路由
func RegisterRoutes(router *gin.RouterGroup, tokenManager token.Manager, dbResolver *dbresolver.DBResolver, ldapClient *ldap.LDAPClient) {
	handler := newHandler(handlerOption{
		tokenManager:   tokenManager,
		dbResolver:     dbResolver,
		ldapClient:     ldapClient,
		captchaLimiter: limiter.NewKeyLimiter(rate.Every(time.Second), 3),
		loginLimiter:   limiter.NewLoginLimiter(),
	})

	authGroup := router.Group("/auth")
	authGroup.POST("/login", handler.login)
	authGroup.GET("/captcha", handler.authCaptcha)

	// 需要认证的路由
	userGroup := router.Group("/user")
	userGroup.Use(auth.AuthMiddleware())
	userGroup.Use(middleware.CheckToken(tokenManager))
	{
		userGroup.POST("/logout", handler.logout)
		userGroup.GET("/me", handler.getCurrentUser)
		userGroup.PUT("/me", handler.updateCurrentUser)
	}
}
