package image

import (
	"github.com/gin-gonic/gin"
	"tiansuoVM/pkg/auth"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/server/middleware"
	"tiansuoVM/pkg/token"
)

func RegisterRoutes(router *gin.RouterGroup, tokenManager token.Manager, dbResolver *dbresolver.DBResolver) {
	h := newHandler(dbResolver)

	// 所有镜像路由都需要认证
	imageGroup := router.Group("/images")
	imageGroup.Use(middleware.CheckToken(tokenManager))
	imageGroup.Use(auth.AuthMiddleware())

	// 公共访问接口
	{
		imageGroup.GET("", h.listImages)
		imageGroup.GET("/:id", h.getImage)
	}

	// 需要管理员权限的接口
	adminImageGroup := imageGroup.Group("")
	adminImageGroup.Use(auth.AdminRequired())
	{
		adminImageGroup.POST("", h.createImage)
		adminImageGroup.PUT("/:id", h.updateImage)
		//adminImageGroup.DELETE("/:id", h.deleteImage)
	}
}
