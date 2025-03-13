package vm

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"tiansuoVM/pkg/apis/v1/logs"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/token"
	"tiansuoVM/pkg/types"
	"tiansuoVM/pkg/vm/controller"
)

type vmHelper struct {
	dbResolver   *dbresolver.DBResolver
	vmController *controller.Controller
}

func newVMHelper(dbResolver *dbresolver.DBResolver, vmController *controller.Controller) *vmHelper {
	return &vmHelper{
		dbResolver:   dbResolver,
		vmController: vmController,
	}
}

// asyncCreateVM 异步创建虚拟机资源
func (h *vmHelper) asyncCreateVM(ctx context.Context, instance *model.VirtualMachine) {
	maxAttempts := 3
	retryDelay := types.DefaultRetryTimeDely

	zap.L().Info("创建虚拟机资源")

	// 获取镜像信息
	image, err := dao.GetImageByID(ctx, h.dbResolver, instance.ImageID)
	if err != nil {
		zap.L().Error("获取镜像信息失败", zap.Int64("imageID", instance.ImageID), zap.Error(err))
		daoErr := h.updateVMStatus(ctx, instance.ID, model.VMStatusError, "获取镜像信息失败")
		if daoErr != nil {
			zap.L().Error("更新VM状态失败", zap.Int64("id", instance.ID), zap.Error(daoErr))
		}
		return
	}
	if image == nil {
		zap.L().Error("镜像不存在", zap.Int64("imageID", instance.ImageID))
		daoErr := h.updateVMStatus(ctx, instance.ID, model.VMStatusError, "镜像不存在")
		if daoErr != nil {
			zap.L().Error("更新VM状态失败", zap.Int64("id", instance.ID), zap.Error(daoErr))
		}
		return
	}

	// 设置实例的镜像名称
	instance.ImageName = image.Name

	// 重试创建VM资源
	err = retry(maxAttempts, retryDelay, func() error {
		return h.vmController.CreateVM(ctx, instance)
	})

	if err != nil {
		zap.L().Error("创建VM资源失败", zap.Int64("id", instance.ID), zap.Error(err))
		daoErr := h.updateVMStatus(ctx, instance.ID, model.VMStatusError, fmt.Sprintf("创建VM资源失败: %v", err))
		if daoErr != nil {
			zap.L().Error("更新VM状态失败", zap.Int64("id", instance.ID), zap.Error(daoErr))
		}
		return
	}

	// 记录事件日志
	logs.EventLogChannel <- &model.EventLog{
		ResourceType: model.ResourceTypeVM,
		ResourceUID:  instance.UID,
		EventType:    model.EventTypeCreation,
		Creator:      instance.Creator,
		CreatedAt:    time.Now().UnixMilli(),
	}
}

// asyncDeleteVM 异步处理待删除的虚拟机
func (h *vmHelper) asyncDeleteVM(ctx context.Context, id int64) {
	vm, err := dao.GetVMByID(ctx, h.dbResolver, id)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}
	if vm == nil {
		zap.L().Warn("VM不存在", zap.Int64("vmID", id))
		return
	}

	// 更新VM状态为已标记删除
	updates := map[string]interface{}{
		"status": model.VMStatusMarkedForDeletion,
	}
	err = dao.UpdateVMByID(ctx, h.dbResolver, vm.ID, updates)
	if err != nil {
		zap.L().Error("更新VM状态失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}

	// 暂停相关的Pod
	err = h.vmController.StopVM(ctx, vm)
	if err != nil {
		zap.L().Error("暂停VM Pod失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}

	// 记录事件日志
	logs.EventLogChannel <- &model.EventLog{
		ResourceType: model.ResourceTypeVM,
		ResourceUID:  vm.UID,
		EventType:    model.EventTypeDeletion,
		Creator:      token.GetUIDFromCtx(ctx),
		CreatedAt:    time.Now().UnixMilli(),
	}
}

// updateVMStatus 更新虚拟机状态
func (h *vmHelper) updateVMStatus(ctx context.Context, id int64, status model.VMStatus, message string) error {
	vm := &model.VirtualMachine{
		ID:        id,
		Status:    status,
		Message:   message,
		UpdatedAt: time.Now().Unix(),
		Updater:   token.GetUIDFromCtx(ctx),
	}

	updates := map[string]interface{}{
		"status":  status,
		"message": message,
	}

	return dao.UpdateVMByID(ctx, h.dbResolver, vm.ID, updates)
}

// cleanupFailedVMCreation 清理创建失败的VM资源
func (h *vmHelper) cleanupFailedVMCreation(ctx context.Context, instance *model.VirtualMachine) {
	// 尝试删除Pod资源
	err := h.vmController.DeleteVM(ctx, instance)
	if err != nil {
		zap.L().Error("清理失败的VM资源失败", zap.Int64("vmID", instance.ID), zap.Error(err))
	}
}

// asyncStartVM 异步启动虚拟机
func (h *vmHelper) asyncStartVM(ctx context.Context, id int64) {
	vm, err := dao.GetVMByID(ctx, h.dbResolver, id)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}
	if vm == nil {
		zap.L().Warn("VM不存在", zap.Int64("vmID", id))
		return
	}

	// 更新VM状态为启动中
	updates := map[string]interface{}{
		"status": model.VMStatusPending,
	}
	err = dao.UpdateVMByID(ctx, h.dbResolver, vm.ID, updates)
	if err != nil {
		zap.L().Error("更新VM状态失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}

	// 重试启动VM资源
	err = retry(3, types.DefaultTimeout, func() error {
		return h.vmController.StartVM(ctx, vm)
	})

	if err != nil {
		zap.L().Error("启动VM资源失败", zap.Int64("vmID", id), zap.Error(err))
		_ = h.updateVMStatus(ctx, id, model.VMStatusError, fmt.Sprintf("启动VM失败: %v", err))
		return
	}

	// 记录事件日志
	logs.EventLogChannel <- &model.EventLog{
		ResourceType: model.ResourceTypeVM,
		ResourceUID:  vm.UID,
		EventType:    model.EventTypeStart,
		Creator:      token.GetUIDFromCtx(ctx),
		CreatedAt:    time.Now().UnixMilli(),
	}
}

// asyncStopVM 异步停止虚拟机
func (h *vmHelper) asyncStopVM(ctx context.Context, id int64) {
	vm, err := dao.GetVMByID(ctx, h.dbResolver, id)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}
	if vm == nil {
		zap.L().Warn("VM不存在", zap.Int64("vmID", id))
		return
	}

	// 更新VM状态为停止中
	updates := map[string]interface{}{
		"status": model.VMStatusStopped,
	}
	err = dao.UpdateVMByID(ctx, h.dbResolver, vm.ID, updates)
	if err != nil {
		zap.L().Error("更新VM状态失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}

	// 重试停止VM资源
	err = retry(3, types.DefaultTimeout, func() error {
		return h.vmController.StopVM(ctx, vm)
	})

	if err != nil {
		zap.L().Error("停止VM资源失败", zap.Int64("vmID", id), zap.Error(err))
		_ = h.updateVMStatus(ctx, id, model.VMStatusError, fmt.Sprintf("停止VM失败: %v", err))
		return
	}

	// 记录事件日志
	logs.EventLogChannel <- &model.EventLog{
		ResourceType: model.ResourceTypeVM,
		ResourceUID:  vm.UID,
		EventType:    model.EventTypeStop,
		Creator:      token.GetUIDFromCtx(ctx),
		CreatedAt:    time.Now().UnixMilli(),
	}
}

// asyncRestartVM 异步重启虚拟机
func (h *vmHelper) asyncRestartVM(ctx context.Context, id int64) {
	vm, err := dao.GetVMByID(ctx, h.dbResolver, id)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}
	if vm == nil {
		zap.L().Warn("VM不存在", zap.Int64("vmID", id))
		return
	}

	// 先更新VM状态为停止中
	updates := map[string]interface{}{
		"status": model.VMStatusStopped,
	}
	err = dao.UpdateVMByID(ctx, h.dbResolver, vm.ID, updates)
	if err != nil {
		zap.L().Error("更新VM状态失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}

	// 先停止VM
	err = retry(3, types.DefaultTimeout, func() error {
		return h.vmController.StopVM(ctx, vm)
	})
	if err != nil {
		zap.L().Error("停止VM资源失败", zap.Int64("vmID", id), zap.Error(err))
		_ = h.updateVMStatus(ctx, id, model.VMStatusError, fmt.Sprintf("重启VM失败（停止阶段）: %v", err))
		return
	}

	// 更新VM状态为启动中
	updates = map[string]interface{}{
		"status": model.VMStatusPending,
	}
	err = dao.UpdateVMByID(ctx, h.dbResolver, vm.ID, updates)
	if err != nil {
		zap.L().Error("更新VM状态失败", zap.Int64("vmID", id), zap.Error(err))
		return
	}

	// 再启动VM
	err = retry(3, types.DefaultTimeout, func() error {
		return h.vmController.StartVM(ctx, vm)
	})
	if err != nil {
		zap.L().Error("启动VM资源失败", zap.Int64("vmID", id), zap.Error(err))
		_ = h.updateVMStatus(ctx, id, model.VMStatusError, fmt.Sprintf("重启VM失败（启动阶段）: %v", err))
		return
	}

	// 记录事件日志
	logs.EventLogChannel <- &model.EventLog{
		ResourceType: model.ResourceTypeVM,
		ResourceUID:  vm.UID,
		EventType:    model.EventTypeRestart,
		Creator:      token.GetUIDFromCtx(ctx),
		CreatedAt:    time.Now().UnixMilli(),
	}
}

// recoverMarkedForDeletionVM 恢复已标记删除的VM
func (h *vmHelper) recoverMarkedForDeletionVM(ctx context.Context, id int64) error {
	vm, err := dao.GetVMByID(ctx, h.dbResolver, id)
	if err != nil {
		return err
	}
	if vm == nil {
		return fmt.Errorf("VM不存在")
	}

	if vm.Status != model.VMStatusMarkedForDeletion {
		return fmt.Errorf("VM状态不是已标记删除")
	}

	// 更新VM状态为已停止
	updates := map[string]interface{}{
		"status": model.VMStatusStopped,
	}
	err = dao.UpdateVMByID(ctx, h.dbResolver, vm.ID, updates)
	if err != nil {
		zap.L().Error("更新VM状态失败", zap.Int64("vmID", id), zap.Error(err))
		return err
	}

	// 记录事件日志
	logs.EventLogChannel <- &model.EventLog{
		ResourceType: model.ResourceTypeVM,
		ResourceUID:  vm.UID,
		EventType:    model.EventTypeUpdate,
		Operation:    "recover",
		Creator:      token.GetUIDFromCtx(ctx),
		CreatedAt:    time.Now().UnixMilli(),
	}

	return nil
}

func retry(maxAttempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if i < maxAttempts-1 {
			time.Sleep(delay)
		}
	}
	return err
}
