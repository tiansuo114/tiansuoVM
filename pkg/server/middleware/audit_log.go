package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strings"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/token"
	"time"
)

const (
	AuditLogKeyOperation = "audit_operation"
	AuditLogKeyDetail    = "audit_detail"
)

// 不需要审计的路径
var skipPaths = map[string]struct{}{
	"/apis/v1/common/fs":      {},
	"/apis/v1/common/healthy": {},
}

// AddAuditLog 审计日志中间件
func AddAuditLog(db *dbresolver.DBResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过不需要审计的路径
		if _, ok := skipPaths[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		startTime := time.Now()

		// 获取用户信息
		userInfo, _ := token.PayloadFromCtx(c)

		// 继续处理请求
		c.Next()

		// 收集审计信息
		duration := time.Since(startTime).Milliseconds()
		log := &model.AuditLog{
			UID:       userInfo.UID,
			Username:  userInfo.Username,
			Module:    getModule(c.Request.URL.Path),
			Method:    c.Request.Method,
			Duration:  duration,
			IP:        c.ClientIP(),
			Status:    c.Writer.Status(),
			URI:       c.Request.URL.Path,
			Operation: c.GetString(AuditLogKeyOperation),
			Detail:    c.GetString(AuditLogKeyDetail),
			CreatedAt: time.Now().Unix(),
		}

		// 异步写入审计日志
		go func() {
			if err := dao.InsertAuditLog(&gin.Context{}, db, log); err != nil {
				zap.L().Error("failed to insert audit log", zap.Error(err))
			}
		}()
	}
}

// getModule 从URL路径获取模块名
func getModule(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return "unknown"
}
