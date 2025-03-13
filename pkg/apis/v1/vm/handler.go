package vm

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/server/encoding"
	"tiansuoVM/pkg/server/errutil"
	"tiansuoVM/pkg/server/request"
	"tiansuoVM/pkg/token"
	"tiansuoVM/pkg/types"
	"tiansuoVM/pkg/vm/controller"
)

type handlerOption struct {
	tokenManager token.Manager
	dbResolver   *dbresolver.DBResolver
	vmController *controller.Controller
}

type handler struct {
	handlerOption
	vmHelper *vmHelper
}

func newHandler(option handlerOption) *handler {
	return &handler{
		handlerOption: option,
		vmHelper:      newVMHelper(option.dbResolver, option.vmController),
	}
}

// listVMs 获取当前用户的虚拟机列表
func (h *handler) listVMs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req listVMReq
	if err := c.ShouldBindQuery(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// 获取用户信息
	payload, err := token.PayloadFromCtx(c)
	if err != nil {
		zap.L().Error("获取用户信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 获取虚拟机列表
	vms, total, err := dao.ListVMsByUserUID(ctx, h.dbResolver, payload.UID, req.Pagination)
	if err != nil {
		zap.L().Error("获取虚拟机列表失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 转换为响应结构体
	vmInfos := make([]VMInfo, len(vms))
	for i, vm := range vms {
		vmInfos[i] = VMInfo{
			ID:        vm.ID,
			Name:      vm.Name,
			UID:       vm.UID,
			UserUID:   vm.UserUID,
			UserName:  vm.UserName,
			CPU:       vm.CPU,
			MemoryMB:  vm.MemoryMB,
			DiskGB:    vm.DiskGB,
			Status:    vm.Status,
			PodName:   vm.PodName,
			Namespace: vm.Namespace,
			NodeName:  vm.NodeName,
			IP:        vm.IP,
			SSHPort:   vm.SSHPort,
			ImageName: vm.ImageName,
			ImageID:   vm.ImageID,
			Message:   vm.Message,
			CreatedAt: vm.CreatedAt,
			Creator:   vm.Creator,
			UpdatedAt: vm.UpdatedAt,
			Updater:   vm.Updater,
		}
	}

	// 返回虚拟机列表
	encoding.HandleSuccessList(c, total, vmInfos)
}

// createVM 创建虚拟机
func (h *handler) createVM(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req createVMReq
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 从Token中获取用户信息
	payload, err := token.PayloadFromCtx(c)
	if err != nil {
		zap.L().Error("获取用户信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 解析镜像ID
	imageID, err := strconv.ParseInt(req.ImageID, 10, 64)
	if err != nil {
		zap.L().Error("解析镜像ID失败", zap.String("imageID", req.ImageID), zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 检查镜像是否存在
	image, err := dao.GetImageByID(ctx, h.dbResolver, imageID)
	if err != nil {
		zap.L().Error("获取镜像信息失败", zap.Int64("imageID", imageID), zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if image == nil {
		zap.L().Error("镜像不存在", zap.Int64("imageID", imageID))
		encoding.HandleError(c, errutil.ErrResourceNotFound)
		return
	}

	// 创建VM实例
	instance := &model.VirtualMachine{
		Name:      req.Name,
		UID:       generateUID(),
		UserUID:   payload.UID,
		UserName:  payload.Username,
		CPU:       int32(req.CPU),
		MemoryMB:  int32(req.Memory),
		DiskGB:    int32(req.Disk),
		Status:    model.VMStatusPending,
		ImageID:   imageID,
		ImageName: image.Name,
		Creator:   payload.UID,
		Updater:   payload.UID,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}

	// 保存VM记录到数据库
	if err := dao.InsertVM(ctx, h.dbResolver, instance); err != nil {
		zap.L().Error("保存VM记录失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 异步创建VM资源
	userInfo, err := token.PayloadFromCtx(ctx)
	if err != nil {
		zap.L().Error("Failed to get user info from context", zap.Error(err))
	}
	asyncCtx := context.Background()
	asyncCtx = token.WithPayload(asyncCtx, userInfo)
	h.vmHelper.asyncCreateVM(asyncCtx, instance)

	vmInfo := VMInfo{
		ID:        instance.ID,
		Name:      instance.Name,
		UID:       instance.UID,
		UserUID:   instance.UserUID,
		UserName:  instance.UserName,
		CPU:       instance.CPU,
		MemoryMB:  instance.MemoryMB,
		DiskGB:    instance.DiskGB,
		Status:    instance.Status,
		PodName:   instance.PodName,
		Namespace: instance.Namespace,
		NodeName:  instance.NodeName,
		IP:        instance.IP,
		SSHPort:   instance.SSHPort,
		ImageName: instance.ImageName,
		ImageID:   instance.ImageID,
		Message:   instance.Message,
		CreatedAt: instance.CreatedAt,
		Creator:   instance.Creator,
		UpdatedAt: instance.UpdatedAt,
		Updater:   instance.Updater,
	}

	// 返回创建成功
	encoding.HandleSuccess(c, vmInfo)
}

// deleteVM 删除虚拟机
func (h *handler) deleteVM(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req deleteVMReq
	if err := c.ShouldBindUri(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 检查VM是否存在
	vm, err := dao.GetVMByID(ctx, h.dbResolver, req.ID)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", req.ID), zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm == nil {
		encoding.HandleError(c, errutil.ErrResourceNotFound)
		return
	}

	// 检查用户权限
	payload, err := token.PayloadFromCtx(c)
	if err != nil {
		zap.L().Error("获取用户信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm.UserUID != payload.UID && payload.Role != model.UserRoleAdmin {
		encoding.HandleError(c, errutil.ErrPermissionDenied)
		return
	}

	// 检查VM状态，不能删除已经处于删除中的VM
	if vm.Status == model.VMStatusTerminating {
		zap.L().Error("VM正在删除中，不能删除", zap.Int64("vmID", req.ID))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 异步删除VM资源
	go h.vmHelper.asyncDeleteVM(context.Background(), req.ID)

	// 返回成功
	encoding.HandleSuccess(c, gin.H{"message": "VM删除请求已提交"})
}

// getVM 获取虚拟机详情
func (h *handler) getVM(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req getVMReq
	if err := c.ShouldBindUri(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 检查VM是否存在
	vm, err := dao.GetVMByID(ctx, h.dbResolver, req.ID)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", req.ID), zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm == nil {
		encoding.HandleError(c, errutil.ErrResourceNotFound)
		return
	}

	// 检查用户权限
	payload, err := token.PayloadFromCtx(c)
	if err != nil {
		zap.L().Error("获取用户信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm.UserUID != payload.UID && payload.Role != model.UserRoleAdmin {
		encoding.HandleError(c, errutil.ErrPermissionDenied)
		return
	}

	// 转换为响应结构体
	vmInfo := VMInfo{
		ID:        vm.ID,
		Name:      vm.Name,
		UID:       vm.UID,
		UserUID:   vm.UserUID,
		UserName:  vm.UserName,
		CPU:       vm.CPU,
		MemoryMB:  vm.MemoryMB,
		DiskGB:    vm.DiskGB,
		Status:    vm.Status,
		PodName:   vm.PodName,
		Namespace: vm.Namespace,
		NodeName:  vm.NodeName,
		IP:        vm.IP,
		SSHPort:   vm.SSHPort,
		ImageName: vm.ImageName,
		ImageID:   vm.ImageID,
		Message:   vm.Message,
		CreatedAt: vm.CreatedAt,
		Creator:   vm.Creator,
		UpdatedAt: vm.UpdatedAt,
		Updater:   vm.Updater,
	}

	// 返回VM详情
	encoding.HandleSuccess(c, vmInfo)
}

// startVM 启动虚拟机
func (h *handler) startVM(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req startVMReq
	if err := c.ShouldBindUri(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 检查VM是否存在
	vm, err := dao.GetVMByID(ctx, h.dbResolver, req.ID)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", req.ID), zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm == nil {
		encoding.HandleError(c, errutil.ErrResourceNotFound)
		return
	}

	// 检查用户权限
	payload, err := token.PayloadFromCtx(c)
	if err != nil {
		zap.L().Error("获取用户信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm.UserUID != payload.UID && payload.Role != model.UserRoleAdmin {
		encoding.HandleError(c, errutil.ErrPermissionDenied)
		return
	}

	// 检查VM状态，只能启动已停止的VM
	if vm.Status != model.VMStatusStopped {
		zap.L().Error("VM状态错误", zap.Int64("vmID", req.ID), zap.String("status", string(vm.Status)))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 异步启动VM
	go h.vmHelper.asyncStartVM(context.Background(), req.ID)

	// 返回成功
	encoding.HandleSuccess(c, gin.H{"message": "VM启动请求已提交"})
}

// stopVM 停止虚拟机
func (h *handler) stopVM(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req stopVMReq
	if err := c.ShouldBindUri(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 检查VM是否存在
	vm, err := dao.GetVMByID(ctx, h.dbResolver, req.ID)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", req.ID), zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm == nil {
		encoding.HandleError(c, errutil.ErrResourceNotFound)
		return
	}

	// 检查用户权限
	payload, err := token.PayloadFromCtx(c)
	if err != nil {
		zap.L().Error("获取用户信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm.UserUID != payload.UID && payload.Role != model.UserRoleAdmin {
		encoding.HandleError(c, errutil.ErrPermissionDenied)
		return
	}

	// 检查VM状态，只能停止正在运行的VM
	if vm.Status != model.VMStatusRunning {
		zap.L().Error("VM状态错误", zap.Int64("vmID", req.ID), zap.String("status", string(vm.Status)))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 异步停止VM
	go h.vmHelper.asyncStopVM(context.Background(), req.ID)

	// 返回成功
	encoding.HandleSuccess(c, gin.H{"message": "VM停止请求已提交"})
}

// restartVM 重启虚拟机
func (h *handler) restartVM(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req restartVMReq
	if err := c.ShouldBindUri(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 检查VM是否存在
	vm, err := dao.GetVMByID(ctx, h.dbResolver, req.ID)
	if err != nil {
		zap.L().Error("获取VM信息失败", zap.Int64("vmID", req.ID), zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm == nil {
		encoding.HandleError(c, errutil.ErrResourceNotFound)
		return
	}

	// 检查用户权限
	payload, err := token.PayloadFromCtx(c)
	if err != nil {
		zap.L().Error("获取用户信息失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}
	if vm.UserUID != payload.UID && payload.Role != model.UserRoleAdmin {
		encoding.HandleError(c, errutil.ErrPermissionDenied)
		return
	}

	// 检查VM状态，只能重启正在运行的VM
	if vm.Status != model.VMStatusRunning {
		zap.L().Error("VM状态错误", zap.Int64("vmID", req.ID), zap.String("status", string(vm.Status)))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 异步重启VM
	go h.vmHelper.asyncRestartVM(context.Background(), req.ID)

	// 返回成功
	encoding.HandleSuccess(c, gin.H{"message": "VM重启请求已提交"})
}

// createVM 创建虚拟机
func (h *handler) adminCreateVM(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req adminCreateVMReq
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 验证请求
	if err := request.ValidateStruct(ctx, req); err != nil {
		encoding.HandleError(c, err)
		return
	}

	// 从Token中获取操作者ID
	operatorUID := token.GetUIDFromCtx(c)

	// 检查用户是否存在
	found, userInstance, err := dao.GetUserByUsername(ctx, h.dbResolver, req.OwnerUsername)
	if err != nil {
		zap.L().Error("查询用户失败", zap.Error(err), zap.String("username", req.OwnerUsername))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	if !found {
		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "用户不存在"))
		return
	}

	imageID, err := strconv.Atoi(req.ImageID)
	if err != nil {
		zap.L().Error("解析请求参数失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}
	// 检查镜像是否存在
	imageInstance, err := dao.GetImageByID(ctx, h.dbResolver, int64(imageID))
	if err != nil {
		zap.L().Error("查询镜像失败", zap.Error(err), zap.String("image_id", req.ImageID))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 创建虚拟机记录
	vm := model.VirtualMachine{
		Name:      req.Name,
		UserUID:   userInstance.UID,
		UserName:  userInstance.Username,
		CPU:       int32(req.CPU),
		MemoryMB:  int32(req.Memory),
		DiskGB:    int32(req.Disk),
		Status:    model.VMStatusPending,
		ImageName: imageInstance.Name,
		Creator:   operatorUID,
		Updater:   operatorUID,
	}

	if err := dao.InsertVM(ctx, h.dbResolver, &vm); err != nil {
		zap.L().Error("创建虚拟机记录失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	encoding.HandleSuccess(c)

	// 异步创建VM资源
	go h.vmHelper.asyncCreateVM(context.Background(), &vm)
}

// RecoverVM 恢复已标记删除的VM
func (h *handler) recoverVM(c *gin.Context) {
	var req recoverVMReq
	if err := c.ShouldBindUri(&req); err != nil {
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	err := h.vmHelper.recoverMarkedForDeletionVM(c.Request.Context(), req.ID)
	if err != nil {
		encoding.HandleError(c, err)
		return
	}

	encoding.HandleSuccess(c)
}

// 生成唯一标识符
func generateUID() string {
	timestamp := time.Now().Unix()
	randomNum := time.Now().UnixNano() % 10000
	return fmt.Sprintf("vm-%d-%d", timestamp, randomNum)
}
