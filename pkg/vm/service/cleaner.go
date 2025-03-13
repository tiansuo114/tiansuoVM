package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/vm/controller"
)

// VMCleaner VM资源清理服务
type VMCleaner struct {
	dbResolver               *dbresolver.DBResolver
	vmController             *controller.Controller
	DeletedVMRetentionPeriod int
	stopCh                   chan struct{}
}

// NewVMCleaner 创建VM清理服务
func NewVMCleaner(dbResolver *dbresolver.DBResolver, vmController *controller.Controller, deletedVMRetentionPeriod int) *VMCleaner {
	return &VMCleaner{
		dbResolver:               dbResolver,
		vmController:             vmController,
		DeletedVMRetentionPeriod: deletedVMRetentionPeriod,
		stopCh:                   make(chan struct{}),
	}
}

// Start 启动清理服务
func (c *VMCleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute) // 每2分钟检查一次
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.cleanMarkedForDeletionVMs(ctx)
		}
	}
}

// Stop 停止清理服务
func (c *VMCleaner) Stop() {
	close(c.stopCh)
}

// cleanMarkedForDeletionVMs 清理已标记删除且超过保留时间的VM资源
func (c *VMCleaner) cleanMarkedForDeletionVMs(ctx context.Context) {
	// 计算截止时间
	deadline := time.Now().Add(-(time.Hour * time.Duration(c.DeletedVMRetentionPeriod))).UnixMilli()

	// 查询需要清理的VM
	vms, err := dao.GetVMsByStatusAndDeadline(ctx, c.dbResolver, model.VMStatusMarkedForDeletion, deadline)
	if err != nil {
		zap.L().Error("查询待清理VM失败", zap.Error(err))
		return
	}

	for _, vm := range vms {
		// 删除VM资源
		err = c.vmController.DeleteVM(ctx, vm)
		if err != nil {
			zap.L().Error("清理VM资源失败",
				zap.String("name", vm.Name),
				zap.Int64("id", vm.ID),
				zap.Error(err))
			continue
		}

		// 从数据库中删除记录
		err = dao.DeleteVM(ctx, c.dbResolver, vm.ID)
		if err != nil {
			zap.L().Error("从数据库删除VM记录失败",
				zap.String("name", vm.Name),
				zap.Int64("id", vm.ID),
				zap.Error(err))
		}

		zap.L().Info("成功清理VM资源",
			zap.String("name", vm.Name),
			zap.Int64("id", vm.ID))
	}
}
