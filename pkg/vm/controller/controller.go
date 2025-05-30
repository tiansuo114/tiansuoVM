package controller

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"tiansuoVM/pkg/client/k8s"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
)

// Controller 虚拟机控制器
type Controller struct {
	opts *Options

	// Kubernetes clients
	k8sClient  *k8s.KubeClient
	clientset  *kubernetes.Clientset
	restConfig *rest.Config

	// Database client
	dbResolver *dbresolver.DBResolver

	scName string

	// Controller state
	stopCh chan struct{}
	wg     sync.WaitGroup
	mutex  sync.RWMutex
}

var nodeIpMap map[string]string

func (c *Controller) InitNodeIpMap() error {
	ipMap := make(map[string]string)
	nodes, err := c.clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}
	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				ipMap[node.Name] = address.Address
			}
		}
	}

	nodeIpMap = ipMap

	return nil
}

// NewController 创建新的VM控制器
func NewController(opts *Options) (*Controller, error) {
	if errs := opts.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("invalid options: %v", errs)
	}

	k8sClient, err := k8s.NewKubeClient(opts.KubeOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	if opts.DatabaseOpts == nil {
		return nil, fmt.Errorf("database resolver cannot be nil")
	}

	controller := &Controller{
		opts:       opts,
		k8sClient:  k8sClient,
		clientset:  k8sClient.GetClientset(),
		restConfig: k8sClient.GetConfig(),
		dbResolver: opts.DatabaseOpts,
		stopCh:     make(chan struct{}),
		scName:     opts.StorageClassName,
	}

	return controller, nil
}

// Start 启动控制器
func (c *Controller) Start(ctx context.Context) error {
	zap.L().Info("Starting VM controller")

	// 启动Pod监控
	go c.runPodWatcher(ctx)

	// 启动状态同步
	go c.runStatusSync(ctx)

	return nil
}

// Stop 停止控制器
func (c *Controller) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// CreateVM 创建虚拟机
func (c *Controller) CreateVM(ctx context.Context, vm *model.VirtualMachine) error {
	// 如果没有分配SSH端口，先分配一个
	if vm.SSHPort == 0 {
		port, err := c.allocateSSHPort(ctx)
		if err != nil {
			return fmt.Errorf("failed to allocate SSH port: %w", err)
		}
		vm.SSHPort = port
	}

	// 创建Pod
	statefulSet, err := c.createPod(ctx, vm)
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	// 更新VM信息
	vm.PodName = statefulSet.Name
	vm.Namespace = statefulSet.Namespace
	vm.Status = model.VMStatusPending
	updater := map[string]interface{}{
		"pod_name":  statefulSet.Name,
		"namespace": statefulSet.Namespace,
		"status":    model.VMStatusPending,
		"ssh_port":  vm.SSHPort,
		"node_ip":   nodeIpMap[statefulSet.Spec.NodeName],
	}

	// 保存到数据库
	if err := dao.UpdateVMByID(ctx, c.dbResolver, vm.ID, updater); err != nil {
		// 删除已创建的Pod
		_ = c.clientset.AppsV1().StatefulSets(statefulSet.Namespace).Delete(ctx, statefulSet.Name, metav1.DeleteOptions{})
		return fmt.Errorf("failed to save vm: %w", err)
	}

	// 立即创建SSH服务
	if err := c.createSSHService(ctx, vm); err != nil {
		zap.L().Error("Failed to create SSH service", zap.String("vm", vm.Name), zap.Error(err))
		// 不返回错误，允许虚拟机继续创建，状态同步时会重试创建服务
	}

	return nil
}

//// createPod 创建Pod
//func (c *Controller) createPod(ctx context.Context, vm *model.VirtualMachine) (*v1.StatefulSet, error) {
//	// 获取镜像信息
//	image, err := dao.GetImageByID(ctx, c.dbResolver, vm.ImageID)
//	if err != nil {
//		return nil, fmt.Errorf("failed to get image: %w", err)
//	}
//	if image == nil {
//		return nil, fmt.Errorf("image not found: %d", vm.ImageID)
//	}
//
//	// 准备SSH密钥环境变量
//	sshEnv := []corev1.EnvVar{}
//	if vm.SSHKey != "" {
//		sshEnv = append(sshEnv, corev1.EnvVar{
//			Name:  "SSH_PUBLIC_KEY",
//			Value: vm.SSHKey,
//		})
//	}
//
//	pod := v1.StatefulSet{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      vm.UID,
//			Namespace: c.opts.Namespace,
//			Labels: map[string]string{
//				"app": "vm",
//				"vm":  vm.Name,
//			},
//		},
//		Spec: v1.StatefulSetSpec{
//			ServiceName: vm.Name + "_vm_service",
//			Replicas:    &vm.ReplicasNum,
//			Template: corev1.PodTemplateSpec{
//				ObjectMeta: metav1.ObjectMeta{
//					Labels: map[string]string{
//						"app": "tiansuo-vm",
//					},
//				},
//				Spec: corev1.PodSpec{
//					Containers: []corev1.Container{
//						{
//							Name:  "vm",
//							Image: image.ImageURL,
//							Resources: corev1.ResourceRequirements{
//								Requests: corev1.ResourceList{
//									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", vm.CPU)),
//									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", vm.MemoryMB)),
//								},
//								Limits: corev1.ResourceList{
//									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", vm.CPU)),
//									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", vm.MemoryMB)),
//								},
//							},
//							VolumeMounts: []corev1.VolumeMount{
//								{
//									Name:      "vm-data",
//									MountPath: "/",
//								},
//							},
//							Ports: []corev1.ContainerPort{
//								{
//									Name:          "ssh",
//									ContainerPort: 22,
//									Protocol:      corev1.ProtocolTCP,
//								},
//							},
//							Env: sshEnv,
//							// 启动命令 - 确保SSH服务启动
//							Command: []string{
//								"/bin/sh",
//								"-c",
//								`
//# 安装SSH服务（如果需要）
//if command -v apt-get >/dev/null 2>&1; then
//  apt-get update -y && apt-get install -y openssh-server
//elif command -v yum >/dev/null 2>&1; then
//  yum install -y openssh-server
//elif command -v apk >/dev/null 2>&1; then
//  apk add --no-cache openssh
//fi
//
//# 配置SSH
//mkdir -p /run/sshd
//chmod 0755 /run/sshd
//mkdir -p /root/.ssh
//
//# 如果提供了SSH公钥，添加到authorized_keys
//if [ ! -z "$SSH_PUBLIC_KEY" ]; then
//  echo "$SSH_PUBLIC_KEY" > /root/.ssh/authorized_keys
//  chmod 600 /root/.ssh/authorized_keys
//fi
//
//# 启动SSH服务
///usr/sbin/sshd -D
//`,
//							},
//						},
//					},
//				},
//			},
//			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
//				{
//					ObjectMeta: metav1.ObjectMeta{
//						Name: "vm-data",
//					},
//					Spec: corev1.PersistentVolumeClaimSpec{
//						AccessModes: []corev1.PersistentVolumeAccessMode{
//							corev1.ReadWriteOnce,
//						},
//						Resources: corev1.VolumeResourceRequirements{
//							Requests: corev1.ResourceList{
//								corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", vm.DiskGB)),
//							},
//						},
//						StorageClassName: &c.scName,
//					},
//				},
//			},
//		},
//	}
//
//	// 创建Pod
//	return c.clientset.AppsV1().StatefulSets(c.opts.Namespace).Create(ctx, &pod, metav1.CreateOptions{})
//}

// createPod 创建Pod
func (c *Controller) createPod(ctx context.Context, vm *model.VirtualMachine) (*corev1.Pod, error) {
	// 获取镜像信息
	image, err := dao.GetImageByID(ctx, c.dbResolver, vm.ImageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	if image == nil {
		return nil, fmt.Errorf("image not found: %d", vm.ImageID)
	}

	// 准备SSH密钥环境变量
	sshEnv := []corev1.EnvVar{}
	if vm.SSHKey != "" {
		sshEnv = append(sshEnv, corev1.EnvVar{
			Name:  "SSH_PUBLIC_KEY",
			Value: vm.SSHKey,
		})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vm.UID,
			Namespace: c.opts.Namespace,
			Labels: map[string]string{
				"app": "vm",
				"vm":  vm.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "vm",
					Image: image.ImageURL,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", vm.CPU)),
							corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", vm.MemoryMB)),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", vm.CPU)),
							corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", vm.MemoryMB)),
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "ssh",
							ContainerPort: 22,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Env: sshEnv,
					// 启动命令 - 确保SSH服务启动
					Command: []string{
						"/bin/sh",
						"-c",
						`
	# 配置SSH
	mkdir -p /run/sshd
	chmod 0755 /run/sshd
	mkdir -p /root/.ssh

	# 如果提供了SSH公钥，添加到authorized_keys
	if [ ! -z "$SSH_PUBLIC_KEY" ]; then
	 echo "$SSH_PUBLIC_KEY" > /root/.ssh/authorized_keys
	 chmod 600 /root/.ssh/authorized_keys
	fi

	# 启动SSH服务
	/usr/sbin/sshd -D
	`,
					},
				},
			},
		},
	}

	// 创建Pod
	return c.clientset.CoreV1().Pods(c.opts.Namespace).Create(ctx, pod, metav1.CreateOptions{})
}

// DeleteVM 删除虚拟机
func (c *Controller) DeleteVM(ctx context.Context, vm *model.VirtualMachine) error {
	// 删除Pod
	err := c.clientset.CoreV1().Pods(vm.Namespace).Delete(ctx, vm.PodName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	if err := dao.DeleteVM(ctx, c.dbResolver, vm.ID); err != nil {
		return fmt.Errorf("failed to update vm status: %w", err)
	}

	return nil
}

// StartVM 启动虚拟机
func (c *Controller) StartVM(ctx context.Context, vm *model.VirtualMachine) error {
	// 对于基于Pod的实现，Start操作等同于Create
	if vm.Status == model.VMStatusStopped {
		return c.CreateVM(ctx, vm)
	}
	return nil
}

// StopVM 停止虚拟机
func (c *Controller) StopVM(ctx context.Context, vm *model.VirtualMachine) error {
	// 删除Pod
	err := c.clientset.CoreV1().Pods(vm.Namespace).Delete(ctx, vm.PodName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to stop pod: %w", err)
	}

	// 更新VM状态
	vm.Status = model.VMStatusStopped
	updater := map[string]interface{}{
		"status": model.VMStatusStopped,
	}
	if err := dao.UpdateVMByID(ctx, c.dbResolver, vm.ID, updater); err != nil {
		return fmt.Errorf("failed to update vm status: %w", err)
	}

	return nil
}

// runPodWatcher 运行Pod监控
func (c *Controller) runPodWatcher(ctx context.Context) {
	c.wg.Add(1)
	defer c.wg.Done()

	// TODO: 实现Pod事件监控
}

// runStatusSync 运行状态同步
func (c *Controller) runStatusSync(ctx context.Context) {
	c.wg.Add(1)
	defer c.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			if err := c.syncVMStatus(ctx); err != nil {
				zap.L().Error("Failed to sync VM status", zap.Error(err))
			}
			if err := c.ensureVMServices(ctx); err != nil {
				zap.L().Error("Failed to ensure VM services", zap.Error(err))
			}
		}
	}
}

// syncVMStatus 同步VM状态
func (c *Controller) syncVMStatus(ctx context.Context) error {
	// 查询所有活跃状态的VM
	vms, err := dao.ListVMsByActiveStatus(ctx, c.dbResolver)
	if err != nil {
		return fmt.Errorf("failed to list active VMs: %w", err)
	}

	// 遍历每个VM，同步其状态
	for _, vm := range vms {
		if vm.PodName == "" || vm.Namespace == "" {
			continue
		}

		// 获取Pod状态
		pod, err := c.clientset.CoreV1().Pods(vm.Namespace).Get(ctx, vm.PodName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Pod不存在，更新VM状态为停止
				vm.Status = model.VMStatusStopped
				vm.IP = ""
				vm.NodeName = ""
				vm.Message = "Pod not found"
				updater := map[string]interface{}{
					"status":    model.VMStatusStopped,
					"ip":        "",
					"node_name": "",
					"message":   "Pod not found",
				}
				if err := dao.UpdateVMByID(ctx, c.dbResolver, vm.ID, updater); err != nil {
					zap.L().Error("Failed to update VM status", zap.Int64("vmID", vm.ID), zap.Error(err))
				}
			}
			continue
		}

		// 获取VM状态
		status := getPodVMStatus(pod)
		if status != vm.Status {
			// 状态变化，更新VM
			vm.Status = status
			vm.IP = pod.Status.PodIP
			vm.NodeName = pod.Spec.NodeName
			updater := map[string]interface{}{
				"status":    status,
				"ip":        pod.Status.PodIP,
				"node_name": pod.Spec.NodeName,
				"node_ip":   nodeIpMap[pod.Spec.NodeName],
			}
			if err := dao.UpdateVMByID(ctx, c.dbResolver, vm.ID, updater); err != nil {
				zap.L().Error("Failed to update VM status", zap.Int64("vmID", vm.ID), zap.Error(err))
			}
		}
	}

	return nil
}

// getPodVMStatus 获取Pod对应的VM状态
func getPodVMStatus(pod *corev1.Pod) model.VMStatus {
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

// ensureVMServices 确保所有运行中的VM都有对应的SSH服务
func (c *Controller) ensureVMServices(ctx context.Context) error {
	// 查询所有运行中的VM
	vms, err := dao.ListVMsByActiveStatus(ctx, c.dbResolver)
	if err != nil {
		return fmt.Errorf("failed to list running VMs: %w", err)
	}

	// 遍历每个VM，确保有对应的SSH服务
	for _, vm := range vms {
		if vm.PodName == "" || vm.Namespace == "" {
			continue
		}

		// 检查SSH服务是否存在
		svcName := fmt.Sprintf("vm-%s-ssh", vm.UID)
		_, err := c.clientset.CoreV1().Services(vm.Namespace).Get(ctx, svcName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// 服务不存在，创建它
				zap.L().Info("SSH service not found, creating it", zap.String("vm", vm.Name), zap.String("service", svcName))

				// 分配SSH端口（如果需要）
				if vm.SSHPort == 0 {
					port, err := c.allocateSSHPort(ctx)
					if err != nil {
						zap.L().Error("Failed to allocate SSH port", zap.String("vm", vm.Name), zap.Error(err))
						continue
					}
					vm.SSHPort = port
					updater := map[string]interface{}{
						"ssh_port": port,
					}
					if err := dao.UpdateVMByID(ctx, c.dbResolver, vm.ID, updater); err != nil {
						zap.L().Error("Failed to update VM SSH port", zap.String("vm", vm.Name), zap.Error(err))
						continue
					}
				}

				// 创建SSH服务
				if err := c.createSSHService(ctx, &vm); err != nil {
					zap.L().Error("Failed to create SSH service", zap.String("vm", vm.Name), zap.Error(err))
				}
			} else {
				zap.L().Error("Failed to get SSH service", zap.String("vm", vm.Name), zap.String("service", svcName), zap.Error(err))
			}
		}
	}

	return nil
}

// createSSHService 创建SSH服务
func (c *Controller) createSSHService(ctx context.Context, vm *model.VirtualMachine) error {
	if vm.SSHPort == 0 {
		return fmt.Errorf("vm has no SSH port assigned")
	}

	// 创建Service
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("vm-%s-ssh", vm.UID),
			Namespace: vm.Namespace,
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

	_, err := c.clientset.CoreV1().Services(vm.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create SSH service: %w", err)
	}

	return nil
}

// allocateSSHPort 分配SSH端口
func (c *Controller) allocateSSHPort(ctx context.Context) (int32, error) {
	opts := c.opts
	maxTries := 10
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < maxTries; i++ {
		port := opts.SSHPortStart + r.Int31n(opts.SSHPortEnd-opts.SSHPortStart+1)

		// 检查端口是否已被使用
		exists, err := dao.CheckSSHPortExists(ctx, c.dbResolver, port)
		if err != nil {
			return 0, err
		}
		if !exists {
			return port, nil
		}
	}

	return 0, fmt.Errorf("failed to allocate SSH port after %d tries", maxTries)
}
