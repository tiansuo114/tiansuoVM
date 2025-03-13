package controller

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"tiansuoVM/pkg/client/k8s"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	vmapi "tiansuoVM/pkg/vm_old/apis/virtualmachine/v1alpha1"
	vmclient "tiansuoVM/pkg/vm_old/client"
)

// Controller manages virtual machines in Kubernetes
type Controller struct {
	opts *Options

	// Kubernetes clients
	k8sClient  *k8s.KubeClient
	vmClient   *vmclient.VMClient
	clientset  *kubernetes.Clientset
	restConfig *rest.Config

	// Database client
	dbResolver *dbresolver.DBResolver

	// Controller state
	stopCh chan struct{}
	wg     sync.WaitGroup
	mutex  sync.RWMutex
}

// NewController creates a new VM controller
func NewController(opts *Options) (*Controller, error) {
	// Validate options
	if errs := opts.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("invalid options: %v", errs)
	}

	// Create Kubernetes client
	k8sClient, err := k8s.NewKubeClient(opts.KubeOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create VM client
	vmClient, err := vmclient.NewVMClient(k8sClient.GetConfig(), opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM client: %w", err)
	}

	// 确保数据库解析器已初始化
	if opts.DatabaseOpts == nil {
		return nil, fmt.Errorf("database resolver cannot be nil, please provide a valid database connection")
	}

	controller := &Controller{
		opts:       opts,
		k8sClient:  k8sClient,
		vmClient:   vmClient,
		clientset:  k8sClient.GetClientset(),
		restConfig: k8sClient.GetConfig(),
		dbResolver: opts.DatabaseOpts,
		stopCh:     make(chan struct{}),
	}

	return controller, nil
}

// Start starts the VM controller
func (c *Controller) Start(ctx context.Context) error {
	zap.L().Info("Starting VM controller")

	// 创建VirtualMachine的informer工厂
	dynamicClient, err := dynamic.NewForConfig(c.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	vmInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		dynamicClient,
		time.Minute*30,
		c.opts.Namespace,
		nil,
	)

	// 获取VirtualMachine的GVR (GroupVersionResource)
	vmGVR := schema.GroupVersionResource{
		Group:    vmapi.GroupName,
		Version:  vmapi.Version,
		Resource: "virtualmachines",
	}

	// 创建informer
	vmInformer := vmInformerFactory.ForResource(vmGVR).Informer()

	// 创建handler
	handler := NewVMHandler(c)

	// 添加事件处理器
	vmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    handler.OnAdd,
		UpdateFunc: handler.OnUpdate,
		DeleteFunc: handler.OnDelete,
	})

	// 启动informer
	vmInformerFactory.Start(c.stopCh)

	// 等待缓存同步
	if !cache.WaitForCacheSync(c.stopCh, vmInformer.HasSynced) {
		return fmt.Errorf("failed to wait for VM cache to sync")
	}

	// 启动同步循环
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		wait.Until(func() {
			if err := c.syncVMs(ctx); err != nil {
				zap.L().Error("Failed to sync VMs", zap.Error(err))
			}
		}, c.opts.SyncPeriod, c.stopCh)
	}()

	return nil
}

// Stop stops the VM controller
func (c *Controller) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	zap.L().Info("Stopping VM controller")
	close(c.stopCh)
	c.wg.Wait()
}

// CreateVM creates a new virtual machine
func (c *Controller) CreateVM(ctx context.Context, vm *model.VirtualMachine) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	zap.L().Info("Creating VM", zap.String("name", vm.Name))

	// Check if VM already exists
	exists, _, err := dao.GetVMByName(ctx, c.dbResolver, vm.Name)
	if err != nil {
		return fmt.Errorf("failed to check if VM exists: %w", err)
	}
	if exists {
		return fmt.Errorf("VM with name %s already exists", vm.Name)
	}

	// Check if image exists
	imageExists, image, err := dao.GetImageByName(ctx, c.dbResolver, vm.ImageName)
	if err != nil {
		return fmt.Errorf("failed to check if image exists: %w", err)
	}
	if !imageExists {
		return fmt.Errorf("image %s not found", vm.ImageName)
	}

	// Verify image is available
	if image.Status != model.ImageStatusAvailable {
		return fmt.Errorf("image %s is not available (status: %s)", vm.ImageName, image.Status)
	}

	// Set initial VM status
	vm.Status = model.VMStatusPending

	// Insert VM into database
	if err := dao.InsertVM(ctx, c.dbResolver, vm); err != nil {
		return fmt.Errorf("failed to insert VM into database: %w", err)
	}

	// Create VM in Kubernetes
	k8sVM := &vmapi.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vm.UID,
			Namespace: c.opts.Namespace,
			Labels: map[string]string{
				"app":          "virtualmachine",
				"vm_old-name":  vm.Name,
				"vm_old-id":    fmt.Sprintf("%d", vm.ID),
				"os-type":      image.OSType,
				"os-version":   image.OSVersion,
				"architecture": image.Architecture,
			},
			Annotations: map[string]string{
				"image.tiansuo.io/name":         image.Name,
				"image.tiansuo.io/location":     image.Location,
				"image.tiansuo.io/format":       image.Format,
				"image.tiansuo.io/os-type":      image.OSType,
				"image.tiansuo.io/os-version":   image.OSVersion,
				"image.tiansuo.io/default-user": image.DefaultUser,
				"image.tiansuo.io/path":         filepath.Base(image.Location),
				"image.tiansuo.io/storage-type": "shared",
			},
		},
		Spec: vmapi.VirtualMachineSpec{
			CPU:       vm.CPU,
			MemoryMB:  vm.MemoryMB,
			DiskGB:    vm.DiskGB,
			ImageName: vm.ImageName,
			Action:    vmapi.VMActionStart,
			Network: vmapi.NetworkConfig{
				Type:     vmapi.NetworkTypeDefault,
				PublicIP: true,
			},
			Storage: vmapi.StorageConfig{
				Type:             vmapi.StorageTypePVC,
				StorageClassName: c.opts.DefaultStorageClass,
			},
		},
	}

	// Add SSH keys if available in the image
	if image.DefaultSSHKey != "" {
		if k8sVM.Spec.SSHKeys == nil {
			k8sVM.Spec.SSHKeys = []string{image.DefaultSSHKey}
		} else {
			k8sVM.Spec.SSHKeys = append(k8sVM.Spec.SSHKeys, image.DefaultSSHKey)
		}
	}

	// Create VM resource in Kubernetes
	_, err = c.vmClient.Create(ctx, k8sVM)
	if err != nil {
		// Update VM status on failure
		updateErr := dao.UpdateVMByID(ctx, c.dbResolver, vm.ID, map[string]interface{}{
			"status": model.VMStatusFailed,
		})
		if updateErr != nil {
			zap.L().Error("Failed to update VM status", zap.Error(updateErr))
		}
		return fmt.Errorf("failed to create VM in Kubernetes: %w", err)
	}

	return nil
}

// DeleteVM deletes a virtual machine
func (c *Controller) DeleteVM(ctx context.Context, id int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	zap.L().Info("Deleting VM", zap.Int64("id", id))

	// Get VM from database
	exists, vm, err := dao.GetVMByID(ctx, c.dbResolver, id)
	if err != nil {
		return fmt.Errorf("failed to get VM: %w", err)
	}
	if !exists {
		return fmt.Errorf("VM with ID %d not found", id)
	}

	// Update VM status to terminating
	if err := dao.UpdateVMByID(ctx, c.dbResolver, id, map[string]interface{}{
		"status": model.VMStatusTerminating,
	}); err != nil {
		return fmt.Errorf("failed to update VM status: %w", err)
	}

	// Delete VM from Kubernetes
	err = c.vmClient.Delete(ctx, vm.UID, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		// Set status back if delete failed
		updateErr := dao.UpdateVMByID(ctx, c.dbResolver, id, map[string]interface{}{
			"status": vm.Status,
		})
		if updateErr != nil {
			zap.L().Error("Failed to revert VM status", zap.Error(updateErr))
		}
		return fmt.Errorf("failed to delete VM from Kubernetes: %w", err)
	}

	// Delete VM from database
	if err := dao.DeleteVMByID(ctx, c.dbResolver, id); err != nil {
		return fmt.Errorf("failed to delete VM from database: %w", err)
	}

	return nil
}

// StartVM starts a virtual machine
func (c *Controller) StartVM(ctx context.Context, id int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	zap.L().Info("Starting VM", zap.Int64("id", id))

	// Get VM from database
	exists, vm, err := dao.GetVMByID(ctx, c.dbResolver, id)
	if err != nil {
		return fmt.Errorf("failed to get VM: %w", err)
	}
	if !exists {
		return fmt.Errorf("VM with ID %d not found", id)
	}

	// Check if VM is already running
	if vm.Status == model.VMStatusRunning {
		return fmt.Errorf("VM is already running")
	}

	// Update VM status to starting
	if err := dao.UpdateVMByID(ctx, c.dbResolver, id, map[string]interface{}{
		"status": model.VMStatusPending,
	}); err != nil {
		return fmt.Errorf("failed to update VM status: %w", err)
	}

	// Start VM in Kubernetes
	_, err = c.vmClient.StartVM(ctx, vm.UID)
	if err != nil {
		// Revert status on failure
		updateErr := dao.UpdateVMByID(ctx, c.dbResolver, id, map[string]interface{}{
			"status": vm.Status,
		})
		if updateErr != nil {
			zap.L().Error("Failed to revert VM status", zap.Error(updateErr))
		}
		return fmt.Errorf("failed to start VM in Kubernetes: %w", err)
	}

	return nil
}

// StopVM stops a virtual machine
func (c *Controller) StopVM(ctx context.Context, id int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	zap.L().Info("Stopping VM", zap.Int64("id", id))

	// Get VM from database
	exists, vm, err := dao.GetVMByID(ctx, c.dbResolver, id)
	if err != nil {
		return fmt.Errorf("failed to get VM: %w", err)
	}
	if !exists {
		return fmt.Errorf("VM with ID %d not found", id)
	}

	// Check if VM is already stopped
	if vm.Status == model.VMStatusStopped {
		return fmt.Errorf("VM is already stopped")
	}

	// Update VM status to stopping
	if err := dao.UpdateVMByID(ctx, c.dbResolver, id, map[string]interface{}{
		"status": model.VMStatusStopped,
	}); err != nil {
		return fmt.Errorf("failed to update VM status: %w", err)
	}

	// Stop VM in Kubernetes
	_, err = c.vmClient.StopVM(ctx, vm.UID)
	if err != nil {
		// Revert status on failure
		updateErr := dao.UpdateVMByID(ctx, c.dbResolver, id, map[string]interface{}{
			"status": vm.Status,
		})
		if updateErr != nil {
			zap.L().Error("Failed to revert VM status", zap.Error(updateErr))
		}
		return fmt.Errorf("failed to stop VM in Kubernetes: %w", err)
	}

	return nil
}

// RestartVM restarts a virtual machine
func (c *Controller) RestartVM(ctx context.Context, id int64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	zap.L().Info("Restarting VM", zap.Int64("id", id))

	// Get VM from database
	exists, vm, err := dao.GetVMByID(ctx, c.dbResolver, id)
	if err != nil {
		return fmt.Errorf("failed to get VM: %w", err)
	}
	if !exists {
		return fmt.Errorf("VM with ID %d not found", id)
	}

	// Update VM status to restarting
	if err := dao.UpdateVMByID(ctx, c.dbResolver, id, map[string]interface{}{
		"status": model.VMStatusPending,
	}); err != nil {
		return fmt.Errorf("failed to update VM status: %w", err)
	}

	// Restart VM in Kubernetes
	_, err = c.vmClient.RestartVM(ctx, vm.UID)
	if err != nil {
		// Revert status on failure
		updateErr := dao.UpdateVMByID(ctx, c.dbResolver, id, map[string]interface{}{
			"status": vm.Status,
		})
		if updateErr != nil {
			zap.L().Error("Failed to revert VM status", zap.Error(updateErr))
		}
		return fmt.Errorf("failed to restart VM in Kubernetes: %w", err)
	}

	return nil
}

// GetVM retrieves a virtual machine by ID
func (c *Controller) GetVM(ctx context.Context, id int64) (*model.VirtualMachine, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Get VM from database
	exists, vm, err := dao.GetVMByID(ctx, c.dbResolver, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("VM with ID %d not found", id)
	}

	return vm, nil
}

// ListVMs lists all virtual machines
func (c *Controller) ListVMs(ctx context.Context) ([]model.VirtualMachine, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// List VMs from database
	vms, err := dao.ListVMsWithNoPagination(ctx, c.dbResolver)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	return vms, nil
}

// ListVMsByUserID lists all virtual machines owned by a user
func (c *Controller) ListVMsByUserID(ctx context.Context, userUID string) ([]model.VirtualMachine, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// List VMs from database
	vms, err := dao.ListVMsByUserUIDWithNoPagination(ctx, c.dbResolver, userUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	return vms, nil
}

// syncVMs synchronizes VM status between Kubernetes and the database
func (c *Controller) syncVMs(ctx context.Context) error {
	zap.L().Info("Syncing VMs")

	// List VMs from database
	vms, err := dao.ListVMsWithNoPagination(ctx, c.dbResolver)
	if err != nil {
		return fmt.Errorf("failed to list VMs from database: %w", err)
	}

	// Sync each VM
	for _, vm := range vms {
		if err := c.syncVM(ctx, &vm); err != nil {
			zap.L().Error("Failed to sync VM",
				zap.String("name", vm.Name),
				zap.Error(err))
			continue
		}
	}

	return nil
}

// syncVM synchronizes a single VM's status between Kubernetes and the database
func (c *Controller) syncVM(ctx context.Context, vm *model.VirtualMachine) error {
	// Skip if VM is in a terminal state
	if vm.Status == model.VMStatusTerminating {
		return nil
	}

	// Get VM from Kubernetes
	k8sVM, err := c.vmClient.Get(ctx, vm.UID, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// VM not found in Kubernetes but exists in database
			if vm.Status != model.VMStatusStopped && vm.Status != model.VMStatusFailed {
				// Update status to failed
				return dao.UpdateVMByID(ctx, c.dbResolver, vm.ID, map[string]interface{}{
					"status": model.VMStatusFailed,
				})
			}
			return nil
		}
		return fmt.Errorf("failed to get VM from Kubernetes: %w", err)
	}

	// Prepare updates
	updates := make(map[string]interface{})

	// Update status
	var newStatus model.VMStatus
	switch k8sVM.Status.State {
	case vmapi.VMStateRunning:
		newStatus = model.VMStatusRunning
	case vmapi.VMStateStopped:
		newStatus = model.VMStatusStopped
	case vmapi.VMStateTerminating:
		newStatus = model.VMStatusTerminating
	case vmapi.VMStateFailed:
		newStatus = model.VMStatusFailed
	case vmapi.VMStatePending, vmapi.VMStateProvisioning:
		newStatus = model.VMStatusPending
	case vmapi.VMStateStarting:
		newStatus = model.VMStatusPending
	default:
		newStatus = model.VMStatusPending
	}

	if vm.Status != newStatus {
		updates["status"] = newStatus
	}

	// Update IP address
	if vm.IP != k8sVM.Status.IP {
		updates["ip"] = k8sVM.Status.IP
	}

	// Update node name
	if vm.NodeName != k8sVM.Status.NodeName {
		updates["node_name"] = k8sVM.Status.NodeName
	}

	// Apply updates if needed
	if len(updates) > 0 {
		return dao.UpdateVMByID(ctx, c.dbResolver, vm.ID, updates)
	}

	return nil
}

// generateVMPod 根据虚拟机CRD生成Pod配置
func (c *Controller) generateVMPod(vm *vmapi.VirtualMachine) (*corev1.Pod, error) {
	// 从注解中获取镜像信息
	imagePath := vm.ObjectMeta.Annotations["image.tiansuo.io/path"]
	imageFormat := vm.ObjectMeta.Annotations["image.tiansuo.io/format"]
	imageLocation := vm.ObjectMeta.Annotations["image.tiansuo.io/location"]
	storageType := vm.ObjectMeta.Annotations["image.tiansuo.io/storage-type"]

	if imagePath == "" || imageFormat == "" {
		return nil, fmt.Errorf("missing required image annotations")
	}

	// 主机路径卷类型
	hostPathFile := corev1.HostPathFile

	// 创建QEMU命令行
	qemuCmd := []string{
		"qemu-system-x86_64",
		"-enable-kvm",
		"-cpu", "host",
		"-smp", fmt.Sprintf("%d", vm.Spec.CPU),
		"-m", fmt.Sprintf("%d", vm.Spec.MemoryMB),
		"-drive", fmt.Sprintf("file=/images/%s,format=%s,if=virtio",
			imagePath, imageFormat),
		"-device", "virtio-net-pci,netdev=net0",
		"-netdev", "user,id=net0,hostfwd=tcp::22-:22",
		"-vnc", ":0",
		"-daemonize",
	}

	// 创建Pod配置
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("vm_old-%s", vm.Name),
			Namespace: vm.Namespace,
			Labels: map[string]string{
				"app":    "vm_old",
				"vm_old": vm.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "virtualmachine.tiansuo.io/v1alpha1",
					Kind:       "VirtualMachine",
					Name:       vm.Name,
					UID:        vm.UID,
					Controller: &[]bool{true}[0],
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "vm_old",
					Image:   "qemux/qemu:6.20", // QEMU容器镜像，需事先准备好
					Command: qemuCmd,
					SecurityContext: &corev1.SecurityContext{
						Privileged: &[]bool{true}[0], // 需要特权模式运行QEMU
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "ssh",
							ContainerPort: 22,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "vnc",
							ContainerPort: 5900,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "vm_old-image",
							MountPath: "/images/" + imagePath,
							ReadOnly:  true,
						},
						{
							Name:      "vm_old-data",
							MountPath: "/data",
						},
						{
							Name:      "dev-kvm",
							MountPath: "/dev/kvm",
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%d", vm.Spec.CPU)),
							corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", vm.Spec.MemoryMB)),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%d", vm.Spec.CPU)),
							corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", vm.Spec.MemoryMB)),
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "vm_old-image",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: imageLocation,
							Type: &hostPathFile,
						},
					},
				},
				{
					Name: "vm_old-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: fmt.Sprintf("vm_old-data-%s", vm.Name),
							ReadOnly:  false,
						},
					},
				},
				{
					Name: "dev-kvm",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/dev/kvm",
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	// 如果是共享存储，修改卷配置
	if storageType == "shared" {
		// 使用NFS或其他共享存储
		// 这里简化处理，仍然使用主机路径，但在实际实现中应该使用PV/PVC
		// 对于共享存储，应该在所有节点上挂载同一个NFS或其他共享存储
	}

	// 如果设置了节点名称，添加节点选择器
	if vm.Spec.NodeName != "" {
		pod.Spec.NodeName = vm.Spec.NodeName
	}

	return pod, nil
}

// SuspendVM 暂停VM但不更新其状态
func (c *Controller) SuspendVM(ctx context.Context, vmID int64) error {
	// 获取VM信息
	exists, vm, err := dao.GetVMByID(ctx, c.dbResolver, vmID)
	if err != nil {
		return fmt.Errorf("获取VM信息失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("VM不存在")
	}

	// Stop VM in Kubernetes
	_, err = c.vmClient.StopVM(ctx, vm.UID)
	if err != nil {
		// Revert status on failure
		updateErr := dao.UpdateVMByID(ctx, c.dbResolver, vmID, map[string]interface{}{
			"status": vm.Status,
		})
		if updateErr != nil {
			zap.L().Error("Failed to revert VM status", zap.Error(updateErr))
		}
		return fmt.Errorf("failed to stop VM in Kubernetes: %w", err)
	}

	return nil
}
