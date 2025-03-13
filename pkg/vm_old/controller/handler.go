package controller

import (
	"context"
	"fmt"
	"tiansuoVM/pkg/token"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/model"
	vmapi "tiansuoVM/pkg/vm_old/apis/virtualmachine/v1alpha1"
)

// VMHandler 处理VirtualMachine CRD事件
type VMHandler struct {
	controller *Controller
}

// NewVMHandler 创建一个新的VMHandler
func NewVMHandler(controller *Controller) *VMHandler {
	return &VMHandler{
		controller: controller,
	}
}

// OnAdd 处理VirtualMachine CRD的添加事件
func (h *VMHandler) OnAdd(obj interface{}) {
	vm, ok := obj.(*vmapi.VirtualMachine)
	if !ok {
		zap.L().Error("Failed to convert to VirtualMachine", zap.Any("obj", obj))
		return
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), h.controller.opts.WaitTimeout)
	defer cancel()

	// 处理VM的创建
	if err := h.handleVMCreate(ctx, vm); err != nil {
		zap.L().Error("Failed to handle VM creation",
			zap.String("name", vm.Name),
			zap.Error(err))
	}
}

// OnUpdate 处理VirtualMachine CRD的更新事件
func (h *VMHandler) OnUpdate(oldObj, newObj interface{}) {
	oldVM, ok := oldObj.(*vmapi.VirtualMachine)
	if !ok {
		zap.L().Error("Failed to convert old object to VirtualMachine", zap.Any("oldObj", oldObj))
		return
	}

	newVM, ok := newObj.(*vmapi.VirtualMachine)
	if !ok {
		zap.L().Error("Failed to convert new object to VirtualMachine", zap.Any("newObj", newObj))
		return
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), h.controller.opts.WaitTimeout)
	defer cancel()

	// 处理VM的更新
	if err := h.handleVMUpdate(ctx, oldVM, newVM); err != nil {
		zap.L().Error("Failed to handle VM update",
			zap.String("name", newVM.Name),
			zap.Error(err))
	}
}

// OnDelete 处理VirtualMachine CRD的删除事件
func (h *VMHandler) OnDelete(obj interface{}) {
	vm, ok := obj.(*vmapi.VirtualMachine)
	if !ok {
		zap.L().Error("Failed to convert to VirtualMachine", zap.Any("obj", obj))
		return
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), h.controller.opts.WaitTimeout)
	defer cancel()

	// 处理VM的删除
	if err := h.handleVMDelete(ctx, vm); err != nil {
		zap.L().Error("Failed to handle VM deletion",
			zap.String("name", vm.Name),
			zap.Error(err))
	}
}

// handleVMCreate 处理VM的创建
func (h *VMHandler) handleVMCreate(ctx context.Context, vm *vmapi.VirtualMachine) error {
	zap.L().Info("Handling VM creation", zap.String("name", vm.Name))

	// 检查VM是否已经存在于数据库中
	exists, dbVM, err := dao.GetVMByUID(ctx, h.controller.dbResolver, vm.Name)
	if err != nil {
		return fmt.Errorf("failed to check if VM exists in database: %w", err)
	}

	if !exists {
		// VM不存在于数据库中，通过UID查询失败，可能是通过kubectl或其他方式直接创建的
		// 此时需要在数据库中创建相应的记录

		// 从镜像注解中获取镜像信息
		imageName := vm.Spec.ImageName
		imageExists, image, err := dao.GetImageByName(ctx, h.controller.dbResolver, imageName)
		if err != nil {
			return fmt.Errorf("failed to get image: %w", err)
		}
		if !imageExists {
			return fmt.Errorf("image %s not found", imageName)
		}

		// 创建数据库记录
		newVM := &model.VirtualMachine{
			Name:      vm.Name,
			UID:       vm.Name,
			CPU:       vm.Spec.CPU,
			MemoryMB:  vm.Spec.MemoryMB,
			DiskGB:    vm.Spec.DiskGB,
			Status:    model.VMStatusPending,
			ImageName: image.Name,
			UserUID:   token.GetUIDFromCtx(ctx), // 默认用户ID，实际应从上下文或其他地方获取
		}

		if err := dao.InsertVM(ctx, h.controller.dbResolver, newVM); err != nil {
			return fmt.Errorf("failed to insert VM into database: %w", err)
		}

		// 重新获取VM
		exists, dbVM, err = dao.GetVMByUID(ctx, h.controller.dbResolver, vm.Name)
		if err != nil || !exists {
			return fmt.Errorf("failed to get newly inserted VM: %w", err)
		}
	}

	// 如果VM不处于终止状态，继续处理
	if dbVM.Status != model.VMStatusTerminating {
		// 生成Pod配置
		pod, err := h.controller.generateVMPod(vm)
		if err != nil {
			return fmt.Errorf("failed to generate VM pod: %w", err)
		}

		// 创建Pod
		_, err = h.controller.clientset.CoreV1().Pods(vm.Namespace).Create(ctx, pod, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			// 更新VM状态为失败
			updateErr := dao.UpdateVMByUID(ctx, h.controller.dbResolver, vm.Name, map[string]interface{}{
				"status": model.VMStatusFailed,
			})
			if updateErr != nil {
				zap.L().Error("Failed to update VM status", zap.Error(updateErr))
			}
			return fmt.Errorf("failed to create VM pod: %w", err)
		}

		// 更新VM状态
		updateErr := dao.UpdateVMByUID(ctx, h.controller.dbResolver, vm.Name, map[string]interface{}{
			"status": model.VMStatusRunning,
		})
		if updateErr != nil {
			zap.L().Error("Failed to update VM status", zap.Error(updateErr))
		}

		// 更新VM状态
		return h.updateVMStatus(ctx, vm, vmapi.VMStateRunning)
	}

	return nil
}

// handleVMUpdate 处理VM的更新
func (h *VMHandler) handleVMUpdate(ctx context.Context, oldVM, newVM *vmapi.VirtualMachine) error {
	zap.L().Info("Handling VM update",
		zap.String("name", newVM.Name),
		zap.String("oldAction", string(oldVM.Spec.Action)),
		zap.String("newAction", string(newVM.Spec.Action)))

	// 检查动作是否发生变化
	if oldVM.Spec.Action != newVM.Spec.Action {
		// 处理动作变化
		switch newVM.Spec.Action {
		case vmapi.VMActionStart:
			return h.handleVMStart(ctx, newVM)
		case vmapi.VMActionStop:
			return h.handleVMStop(ctx, newVM)
		case vmapi.VMActionRestart:
			return h.handleVMRestart(ctx, newVM)
		case vmapi.VMActionPause:
			return h.handleVMPause(ctx, newVM)
		case vmapi.VMActionResume:
			return h.handleVMResume(ctx, newVM)
		default:
			return fmt.Errorf("unknown VM action: %s", newVM.Spec.Action)
		}
	}

	// 检查节点名称是否发生变化
	if oldVM.Spec.NodeName != newVM.Spec.NodeName {
		// 处理迁移
		return h.handleVMMigrate(ctx, oldVM, newVM)
	}

	return nil
}

// handleVMDelete 处理VM的删除
func (h *VMHandler) handleVMDelete(ctx context.Context, vm *vmapi.VirtualMachine) error {
	zap.L().Info("Handling VM deletion", zap.String("name", vm.Name))

	// 删除VM Pod
	err := h.controller.clientset.CoreV1().Pods(vm.Namespace).Delete(ctx, fmt.Sprintf("vm_old-%s", vm.Name), metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete VM pod: %w", err)
	}

	// 从数据库中删除VM
	exists, dbVM, err := dao.GetVMByUID(ctx, h.controller.dbResolver, vm.Name)
	if err != nil {
		return fmt.Errorf("failed to get VM from database: %w", err)
	}

	if exists {
		if err := dao.DeleteVMByID(ctx, h.controller.dbResolver, dbVM.ID); err != nil {
			return fmt.Errorf("failed to delete VM from database: %w", err)
		}
	}

	return nil
}

// handleVMStart 处理VM的启动
func (h *VMHandler) handleVMStart(ctx context.Context, vm *vmapi.VirtualMachine) error {
	zap.L().Info("Starting VM", zap.String("name", vm.Name))

	// 检查Pod是否存在
	pod, err := h.controller.clientset.CoreV1().Pods(vm.Namespace).Get(ctx, fmt.Sprintf("vm_old-%s", vm.Name), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod不存在，创建新的Pod
			pod, err := h.controller.generateVMPod(vm)
			if err != nil {
				return fmt.Errorf("failed to generate VM pod: %w", err)
			}

			_, err = h.controller.clientset.CoreV1().Pods(vm.Namespace).Create(ctx, pod, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create VM pod: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get VM pod: %w", err)
		}
	} else if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		// Pod已经终止，删除并重新创建
		err = h.controller.clientset.CoreV1().Pods(vm.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete terminated VM pod: %w", err)
		}

		// 创建新的Pod
		newPod, err := h.controller.generateVMPod(vm)
		if err != nil {
			return fmt.Errorf("failed to generate VM pod: %w", err)
		}

		_, err = h.controller.clientset.CoreV1().Pods(vm.Namespace).Create(ctx, newPod, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create new VM pod: %w", err)
		}
	}

	// 更新数据库中的VM状态
	err = dao.UpdateVMByUID(ctx, h.controller.dbResolver, vm.Name, map[string]interface{}{
		"status": model.VMStatusRunning,
	})
	if err != nil {
		return fmt.Errorf("failed to update VM status: %w", err)
	}

	// 更新VM状态
	return h.updateVMStatus(ctx, vm, vmapi.VMStateRunning)
}

// handleVMStop 处理VM的停止
func (h *VMHandler) handleVMStop(ctx context.Context, vm *vmapi.VirtualMachine) error {
	zap.L().Info("Stopping VM", zap.String("name", vm.Name))

	// 删除VM Pod
	err := h.controller.clientset.CoreV1().Pods(vm.Namespace).Delete(ctx, fmt.Sprintf("vm_old-%s", vm.Name), metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete VM pod: %w", err)
	}

	// 更新数据库中的VM状态
	err = dao.UpdateVMByUID(ctx, h.controller.dbResolver, vm.Name, map[string]interface{}{
		"status": model.VMStatusStopped,
	})
	if err != nil {
		return fmt.Errorf("failed to update VM status: %w", err)
	}

	// 更新VM状态
	return h.updateVMStatus(ctx, vm, vmapi.VMStateStopped)
}

// handleVMRestart 处理VM的重启
func (h *VMHandler) handleVMRestart(ctx context.Context, vm *vmapi.VirtualMachine) error {
	zap.L().Info("Restarting VM", zap.String("name", vm.Name))

	// 先停止VM
	if err := h.handleVMStop(ctx, vm); err != nil {
		return fmt.Errorf("failed to stop VM during restart: %w", err)
	}

	// 等待一小段时间确保VM已停止
	time.Sleep(2 * time.Second)

	// 然后启动VM
	if err := h.handleVMStart(ctx, vm); err != nil {
		return fmt.Errorf("failed to start VM during restart: %w", err)
	}

	return nil
}

// handleVMPause 处理VM的暂停
func (h *VMHandler) handleVMPause(ctx context.Context, vm *vmapi.VirtualMachine) error {
	zap.L().Info("Pausing VM", zap.String("name", vm.Name))

	// 暂停VM的实现（需要在Pod中运行QEMU命令）
	// 实际暂停操作需要在Pod中执行，这里只更新状态

	// 更新数据库中的VM状态
	err := dao.UpdateVMByUID(ctx, h.controller.dbResolver, vm.Name, map[string]interface{}{
		"status": model.VMStatusStopped, // 暂时用stopped代替paused
	})
	if err != nil {
		return fmt.Errorf("failed to update VM status: %w", err)
	}

	// 更新VM状态
	return h.updateVMStatus(ctx, vm, vmapi.VMStatePaused)
}

// handleVMResume 处理VM的恢复
func (h *VMHandler) handleVMResume(ctx context.Context, vm *vmapi.VirtualMachine) error {
	zap.L().Info("Resuming VM", zap.String("name", vm.Name))

	// 恢复VM的实现（需要在Pod中运行QEMU命令）
	// 实际恢复操作需要在Pod中执行，这里只更新状态

	// 更新数据库中的VM状态
	err := dao.UpdateVMByUID(ctx, h.controller.dbResolver, vm.Name, map[string]interface{}{
		"status": model.VMStatusRunning,
	})
	if err != nil {
		return fmt.Errorf("failed to update VM status: %w", err)
	}

	// 更新VM状态
	return h.updateVMStatus(ctx, vm, vmapi.VMStateRunning)
}

// handleVMMigrate 处理VM的迁移
func (h *VMHandler) handleVMMigrate(ctx context.Context, oldVM, newVM *vmapi.VirtualMachine) error {
	zap.L().Info("Migrating VM",
		zap.String("name", newVM.Name),
		zap.String("fromNode", oldVM.Spec.NodeName),
		zap.String("toNode", newVM.Spec.NodeName))

	// 获取当前Pod
	pod, err := h.controller.clientset.CoreV1().Pods(newVM.Namespace).Get(ctx, fmt.Sprintf("vm_old-%s", newVM.Name), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod不存在，直接创建新的Pod
			return h.handleVMStart(ctx, newVM)
		}
		return fmt.Errorf("failed to get VM pod: %w", err)
	}

	// 删除旧Pod
	err = h.controller.clientset.CoreV1().Pods(newVM.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete old VM pod: %w", err)
	}

	err = wait.PollUntilContextTimeout(ctx, time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
		_, err := h.controller.clientset.CoreV1().Pods(newVM.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for pod deletion: %w", err)
	}

	// 创建新的Pod
	newPod, err := h.controller.generateVMPod(newVM)
	if err != nil {
		return fmt.Errorf("failed to generate VM pod: %w", err)
	}

	_, err = h.controller.clientset.CoreV1().Pods(newVM.Namespace).Create(ctx, newPod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create new VM pod: %w", err)
	}

	// 更新数据库中的VM信息
	err = dao.UpdateVMByUID(ctx, h.controller.dbResolver, newVM.Name, map[string]interface{}{
		"node_name": newVM.Spec.NodeName,
	})
	if err != nil {
		return fmt.Errorf("failed to update VM node name: %w", err)
	}

	return nil
}

// updateVMStatus 更新VM的状态
func (h *VMHandler) updateVMStatus(ctx context.Context, vm *vmapi.VirtualMachine, state vmapi.VMState) error {
	// 使用重试机制更新状态
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// 获取最新的VM
		latestVM, err := h.controller.vmClient.Get(ctx, vm.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// 更新状态
		latestVM.Status.State = state
		latestVM.Status.LastUpdateTime = metav1.Now()

		// 获取Pod信息
		pod, err := h.controller.clientset.CoreV1().Pods(vm.Namespace).Get(ctx, fmt.Sprintf("vm_old-%s", vm.Name), metav1.GetOptions{})
		if err == nil {
			// Pod存在，更新节点和IP信息
			latestVM.Status.NodeName = pod.Spec.NodeName

			// 设置Pod IP
			if pod.Status.PodIP != "" {
				latestVM.Status.IP = pod.Status.PodIP

				// 更新数据库中的IP
				updateErr := dao.UpdateVMByUID(ctx, h.controller.dbResolver, vm.Name, map[string]interface{}{
					"ip": pod.Status.PodIP,
				})
				if updateErr != nil {
					zap.L().Error("Failed to update VM IP in database", zap.Error(updateErr))
				}
			}

			// 设置访问端点
			latestVM.Status.Endpoints = []vmapi.VMEndpoint{
				{
					Type:    vmapi.VMEndpointTypeSSH,
					Address: pod.Status.PodIP,
					Port:    22,
				},
				{
					Type:    vmapi.VMEndpointTypeVNC,
					Address: pod.Status.PodIP,
					Port:    5900,
				},
			}
		}

		// 更新VM状态
		_, err = h.controller.vmClient.Update(ctx, latestVM)
		return err
	})
}
