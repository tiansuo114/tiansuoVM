package logs

import (
	"github.com/gin-gonic/gin"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/token"
)

func RegisterRoutes(group *gin.RouterGroup, tokenManager token.Manager, dbResolver *dbresolver.DBResolver) {
	// 初始化日志监听器
	startUserOperatorLogListener(dbResolver)
	startEventLogListener(dbResolver)
}
