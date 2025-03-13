package dao

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
)

// InsertAuditLog 插入审计日志
func InsertAuditLog(ctx *gin.Context, db *dbresolver.DBResolver, log *model.AuditLog) error {
	if err := db.GetDB().WithContext(ctx).Create(log).Error; err != nil {
		zap.L().Error("failed to insert audit log", zap.Error(err))
		return err
	}
	return nil
}

// ListAuditLogs 查询审计日志
func ListAuditLogs(ctx context.Context, dbResolver *dbresolver.DBResolver, query *ListAuditLogsQuery) (int64, []model.AuditLog, error) {
	var (
		total   int64
		records []model.AuditLog
	)

	db := dbResolver.GetDB()

	// 应用查询条件
	if query.Username != "" {
		db = db.Where("username LIKE ?", "%"+query.Username+"%")
	}
	if query.IP != "" {
		db = db.Where("source_ip LIKE ?", "%"+query.IP+"%")
	}
	if query.Operation != "" {
		db = db.Where("operation = ?", query.Operation)
	}
	if query.StartTime > 0 {
		db = db.Where("created_at >= ?", query.StartTime)
	}
	if query.EndTime > 0 {
		db = db.Where("created_at <= ?", query.EndTime)
	}
	if len(query.Status) > 0 {
		db = db.Where("status IN ?", query.Status)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	// 分页查询
	if err := db.Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).
		Order("id DESC").
		Find(&records).Error; err != nil {
		return 0, nil, err
	}

	return total, records, nil
}

type ListAuditLogsQuery struct {
	Page      int    `json:"page" validate:"required,min=1"`
	PageSize  int    `json:"page_size" validate:"required,min=1,max=100"`
	Username  string `json:"username"`
	IP        string `json:"ip"`
	Operation string `json:"operation"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    []int  `json:"status"`
}
