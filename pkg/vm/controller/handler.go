package controller

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/model"
)

// Handler VM操作处理器
type Handler struct {
	controller *Controller
	workqueue  workqueue.TypedRateLimitingInterface[string]
}

// NewHandler 创建新的处理器
func NewHandler(controller *Controller) *Handler {
	return &Handler{
		controller: controller,
		workqueue:  workqueue.NewTypedRateLimitingQueue[string](workqueue.DefaultTypedControllerRateLimiter[string]()),
	}
}

// OnPodAdd 处理Pod添加事件
func (h *Handler) OnPodAdd(obj interface{}) {
	pod := obj.(*corev1.Pod)
	if !h.isVMPod(pod) {
		return
	}

	h.enqueuePod(pod)
}

// OnPodUpdate 处理Pod更新事件
func (h *Handler) OnPodUpdate(old, new interface{}) {
	oldPod := old.(*corev1.Pod)
	newPod := new.(*corev1.Pod)

	if !h.isVMPod(newPod) {
		return
	}

	if oldPod.Status.Phase == newPod.Status.Phase {
		return
	}

	h.enqueuePod(newPod)
}

// OnPodDelete 处理Pod删除事件
func (h *Handler) OnPodDelete(obj interface{}) {
	pod := obj.(*corev1.Pod)
	if !h.isVMPod(pod) {
		return
	}

	h.enqueuePod(pod)
}

// isVMPod 判断是否为VM的Pod
func (h *Handler) isVMPod(pod *corev1.Pod) bool {
	if pod.Labels == nil {
		return false
	}
	return pod.Labels["app"] == "vm"
}

// enqueuePod 将Pod加入工作队列
func (h *Handler) enqueuePod(pod *corev1.Pod) {
	key, err := cache.MetaNamespaceKeyFunc(pod)
	if err != nil {
		zap.L().Error("Failed to get pod key", zap.Error(err))
		return
	}
	h.workqueue.Add(key)
}

// Run 运行处理器
func (h *Handler) Run(ctx context.Context, workers int) {
	defer h.workqueue.ShutDown()

	for i := 0; i < workers; i++ {
		go h.runWorker(ctx)
	}

	<-ctx.Done()
}

// runWorker 运行工作协程
func (h *Handler) runWorker(ctx context.Context) {
	for h.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem 处理下一个工作项
func (h *Handler) processNextWorkItem(ctx context.Context) bool {
	key, shutdown := h.workqueue.Get()
	if shutdown {
		return false
	}

	defer h.workqueue.Done(key)

	if err := h.syncPod(ctx, key); err != nil {
		h.workqueue.AddRateLimited(key)
		zap.L().Error("Failed to sync pod", zap.String("key", key), zap.Error(err))
	} else {
		h.workqueue.Forget(key)
	}

	return true
}

// syncPod 同步Pod状态
func (h *Handler) syncPod(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	pod, err := h.controller.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return h.handlePodDeletion(ctx, namespace, name)
		}
		return err
	}

	return h.handlePodUpdate(ctx, pod)
}

// handlePodDeletion 处理Pod删除
func (h *Handler) handlePodDeletion(ctx context.Context, namespace, name string) error {
	// 查找对应的VM
	vm, err := dao.GetVMByPodName(ctx, h.controller.dbResolver, name)
	if err != nil {
		return err
	}
	if vm == nil {
		return nil
	}

	// 更新VM状态
	if vm.Status != model.VMStatusTerminating {
		vm.Status = model.VMStatusStopped
		vm.IP = ""
		vm.NodeName = ""
		updater := map[string]interface{}{
			"status":    model.VMStatusStopped,
			"ip":        "",
			"node_name": "",
		}
		if err := dao.UpdateVMByID(ctx, h.controller.dbResolver, vm.ID, updater); err != nil {
			return err
		}
	}

	return nil
}

// handlePodUpdate 处理Pod更新
func (h *Handler) handlePodUpdate(ctx context.Context, pod *corev1.Pod) error {
	// 查找对应的VM
	vm, err := dao.GetVMByPodName(ctx, h.controller.dbResolver, pod.Name)
	if err != nil {
		return err
	}
	if vm == nil {
		return nil
	}

	// 更新VM状态
	status := h.getPodVMStatus(pod)
	if status == vm.Status {
		return nil
	}

	vm.Status = status
	vm.IP = pod.Status.PodIP
	vm.NodeName = pod.Spec.NodeName
	updater := map[string]interface{}{
		"status":    status,
		"ip":        pod.Status.PodIP,
		"node_name": pod.Spec.NodeName,
	}
	if err := dao.UpdateVMByID(ctx, h.controller.dbResolver, vm.ID, updater); err != nil {
		return err
	}

	// 如果Pod运行成功，创建Service暴露SSH端口
	if status == model.VMStatusRunning {
		if err := h.createSSHService(ctx, vm, pod); err != nil {
			return err
		}
	}

	return nil
}

// getPodVMStatus 获取Pod对应的VM状态
func (h *Handler) getPodVMStatus(pod *corev1.Pod) model.VMStatus {
	switch pod.Status.Phase {
	case corev1.PodPending:
		return model.VMStatusPending
	case corev1.PodRunning:
		return model.VMStatusRunning
	case corev1.PodFailed:
		return model.VMStatusFailed
	case corev1.PodSucceeded:
		return model.VMStatusStopped
	default:
		return model.VMStatusError
	}
}

// createSSHService 创建SSH服务
func (h *Handler) createSSHService(ctx context.Context, vm *model.VirtualMachine, pod *corev1.Pod) error {
	// 生成SSH端口
	if vm.SSHPort == 0 {
		port, err := h.allocateSSHPort(ctx)
		if err != nil {
			return err
		}
		vm.SSHPort = port
		updater := map[string]interface{}{
			"ssh_port": port,
		}
		if err := dao.UpdateVMByID(ctx, h.controller.dbResolver, vm.ID, updater); err != nil {
			return err
		}
	}

	// 创建Service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("vm-%s-ssh", vm.UID),
			Namespace: pod.Namespace,
			Labels: map[string]string{
				"app": "vm",
				"vm":  vm.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Name:       "ssh",
					Port:       22,
					TargetPort: intstr.FromInt32(22),
					NodePort:   vm.SSHPort,
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": "vm",
				"vm":  vm.Name,
			},
		},
	}

	_, err := h.controller.clientset.CoreV1().Services(pod.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// allocateSSHPort 分配SSH端口
func (h *Handler) allocateSSHPort(ctx context.Context) (int32, error) {
	opts := h.controller.opts
	maxTries := 10
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < maxTries; i++ {
		port := opts.SSHPortStart + rand.Int31n(opts.SSHPortEnd-opts.SSHPortStart+1)

		// 检查端口是否已被使用
		exists, err := dao.CheckSSHPortExists(ctx, h.controller.dbResolver, port)
		if err != nil {
			return 0, err
		}
		if !exists {
			return port, nil
		}
	}

	return 0, fmt.Errorf("failed to allocate SSH port after %d tries", maxTries)
}
