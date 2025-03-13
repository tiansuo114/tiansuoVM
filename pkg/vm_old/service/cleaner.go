package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/vm_old/controller"
)

const (
	// 清理间隔
	cleanupInterval = time.Hour
	// 删除等待时间
	deleteWaitDuration = 7 * 24 * time.Hour // 一周
)

// VMCleaner VM资源清理服务
type VMCleaner struct {
	dbResolver   *dbresolver.DBResolver
	vmController *controller.Controller
	stopCh       chan struct{}
}

// NewVMCleaner 创建VM清理服务
func NewVMCleaner(dbResolver *dbresolver.DBResolver, vmController *controller.Controller) *VMCleaner {
	return &VMCleaner{
		dbResolver:   dbResolver,
		vmController: vmController,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动清理服务
func (c *VMCleaner) Start() {
	go c.run()
}

// Stop 停止清理服务
func (c *VMCleaner) Stop() {
	close(c.stopCh)
}

// run 运行清理服务
func (c *VMCleaner) run() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup 执行清理操作
func (c *VMCleaner) cleanup() {
	ctx := context.Background()

	// 获取所有待删除的VM
	vms, err := dao.ListVMsByStatusWithNoPagination(ctx, c.dbResolver, model.VMStatusPendingDelete)
	if err != nil {
		zap.L().Error("获取待删除VM列表失败", zap.Error(err))
		return
	}

	now := time.Now()
	for _, vm := range vms {
		// 检查是否超过等待时间
		pendingDeleteTime := time.UnixMilli(vm.UpdatedAt)
		if now.Sub(pendingDeleteTime) < deleteWaitDuration {
			continue
		}

		// 删除VM资源
		if err := c.vmController.DeleteVM(ctx, vm.ID); err != nil {
			zap.L().Error("删除VM资源失败",
				zap.String("uid", vm.UID),
				zap.Int64("id", vm.ID),
				zap.Error(err))
			continue
		}

		// 更新VM状态为已删除
		err = dao.DeleteVMByID(ctx, c.dbResolver, vm.ID)
		if err != nil {
			zap.L().Error("更新VM状态失败",
				zap.String("uid", vm.UID),
				zap.Int64("id", vm.ID),
				zap.Error(err))
		}
	}
}
