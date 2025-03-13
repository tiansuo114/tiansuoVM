package auth

import (
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/server/encoding"
	"tiansuoVM/pkg/server/errutil"
	"tiansuoVM/pkg/token"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// authManager 是包内私有的认证管理器
type authManager struct {
	tokenManager token.Manager
}

// 包内私有的全局变量
var manager *authManager

// Init 初始化认证管理器
func Init(tokenManager token.Manager) {
	manager = &authManager{
		tokenManager: tokenManager,
	}
}

// AuthMiddleware 认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if manager == nil {
			encoding.HandleError(c, errutil.ErrUnauthorized)
			zap.L().Warn("AuthMiddleware tokenManager is nil")
			return
		}

		// 从context中获取token
		tokenString, err := manager.tokenManager.GetTokenFromCtx(c.Request.Context())
		if err != nil {
			encoding.HandleError(c, errutil.ErrUnauthorized)
			zap.L().Warn("Get Token Error")
			return
		}

		// 验证token
		info, err := manager.tokenManager.Verify(tokenString)
		if err != nil {
			encoding.HandleError(c, errutil.ErrUnauthorized)
			zap.L().Warn("Verify Token Error")
			return
		}

		// 将用户信息存储到context中
		c.Set("token_info", info)
		c.Set("user_role", info.Role)
		c.Set("user_id", info.UID)
		c.Set("username", info.Username)

		// 将token信息添加到context中
		newCtx := token.WithPayload(c.Request.Context(), info)
		c.Request = c.Request.WithContext(newCtx)

		c.Next()
	}
}

// AdminRequired 管理员权限检查中间件
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := token.GetUserRoleFromCtx(c.Request.Context())
		if role != model.UserRoleAdmin {
			encoding.HandleError(c, errutil.ErrPermissionDenied)
			zap.L().Warn("AdminRequired permission denied")
			return
		}
		c.Next()
	}
}

// ResourceOwnerRequired 资源所有者权限检查中间件
func ResourceOwnerRequired(getResourceUserID func(*gin.Context) (int64, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户角色
		role := token.GetUserRoleFromCtx(c.Request.Context())

		// 管理员有所有权限
		if role == model.UserRoleAdmin {
			c.Next()
			return
		}

		// 获取当前用户ID
		userUID := token.GetUIDFromCtx(c.Request.Context())
		if userUID == "" {
			encoding.HandleError(c, errutil.ErrUnauthorized)
			zap.L().Warn("ResourceOwnerRequired permission denied")
			return
		}

		// 检查资源所有者
		resourceUserID, err := getResourceUserID(c)
		if err != nil {
			encoding.HandleError(c, errutil.ErrNotFound)
			zap.L().Warn("ResourceOwnerRequired permission denied")
			return
		}

		// 将userUID转换为int64进行比较
		// TODO: 这里需要根据实际情况修改比较逻辑
		if resourceUserID != 0 { // 临时的比较逻辑，需要根据实际UID格式修改
			encoding.HandleError(c, errutil.ErrPermissionDenied)
			return
		}

		c.Next()
	}
}

// GetCurrentUser 从context中获取当前用户信息
func GetCurrentUser(c *gin.Context) (*token.Info, bool) {
	info, exists := c.Get("token_info")
	if !exists {
		return nil, false
	}

	tokenInfo, ok := info.(token.Info)
	if !ok {
		return nil, false
	}

	return &tokenInfo, true
}
