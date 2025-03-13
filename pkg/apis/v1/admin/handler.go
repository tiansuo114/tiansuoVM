package admin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tiansuoVM/pkg/apis/v1/logs"
	"tiansuoVM/pkg/client/ldap"
	"tiansuoVM/pkg/dao"
	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/server/encoding"
	"tiansuoVM/pkg/server/errutil"
	"tiansuoVM/pkg/server/request"
	"tiansuoVM/pkg/token"
	"tiansuoVM/pkg/types"
)

type handlerOption struct {
	tokenManager token.Manager
	dbResolver   *dbresolver.DBResolver
	ldapClient   *ldap.LDAPClient
}

type handler struct {
	handlerOption
}

func newHandler(option handlerOption) *handler {
	return &handler{
		handlerOption: option,
	}
}

// listUsers 获取用户列表
func (h *handler) listUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req userListReq
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

	// 获取用户列表
	users, total, err := dao.ListUsers(ctx, h.dbResolver, req.Pagination)
	if err != nil {
		zap.L().Error("获取用户列表失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	userInfoResp := make([]UserInfo, 0)

	for _, user := range users {
		userInfo := UserInfo{
			ID:        user.ID,
			UID:       user.UID,
			Username:  user.Username,
			Role:      user.Role,
			Primary:   user.Primary,
			Tel:       user.Tel,
			Email:     user.Email,
			Desc:      user.Desc,
			Status:    user.Status,
			GidNumber: user.GidNumber,
			CreatedAt: user.CreatedAt,
			Creator:   user.Creator,
			UpdatedAt: user.UpdatedAt,
			Updater:   user.Updater,
		}
		userInfoResp = append(userInfoResp, userInfo)
	}

	encoding.HandleSuccessList(c, total, userInfoResp)
}

// updateUser 更新用户信息
func (h *handler) updateUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	// 获取路径参数
	uid := c.Param("uid")
	if uid == "" {
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 从Token中获取操作者ID
	operatorUID := token.GetUIDFromCtx(c)

	var req updateUserReq
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

	// 检查用户是否存在
	found, _, err := dao.GetUserByUID(ctx, h.dbResolver, uid)
	if err != nil {
		zap.L().Error("查询用户失败", zap.Error(err), zap.String("uid", uid))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	if !found {
		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "用户不存在"))
		return
	}

	// 构建更新内容
	updates := make(map[string]interface{})
	if req.Tel != "" {
		updates["tel"] = req.Tel
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Desc != "" {
		updates["desc"] = req.Desc
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	// 如果没有任何需要更新的字段
	if len(updates) == 0 {
		encoding.HandleSuccess(c)
		return
	}

	// 执行更新
	if err := dao.UpdateUser(ctx, h.dbResolver, uid, updates); err != nil {
		zap.L().Error("更新用户信息失败", zap.Error(err), zap.String("uid", uid))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 记录操作日志
	logs.UserOperatorLogChannel <- &model.UserOperatorLog{
		UID:       uid,
		Operator:  model.UserOperatorUpdate,
		Operation: fmt.Sprintf("管理员更新用户信息"),
		CreatedAt: time.Now().UnixMilli(),
		Creator:   operatorUID,
	}

	encoding.HandleSuccess(c)
}

// deleteUser 删除用户
func (h *handler) deleteUser(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	// 获取路径参数
	uid := c.Param("uid")
	if uid == "" {
		encoding.HandleError(c, errutil.ErrIllegalParameter)
		return
	}

	// 从Token中获取操作者ID
	operatorUID := token.GetUIDFromCtx(c)

	// 检查用户是否存在
	found, _, err := dao.GetUserByUID(ctx, h.dbResolver, uid)
	if err != nil {
		zap.L().Error("查询用户失败", zap.Error(err), zap.String("uid", uid))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	if !found {
		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "用户不存在"))
		return
	}

	// 删除用户
	if err := dao.DeleteUser(ctx, h.dbResolver, uid); err != nil {
		zap.L().Error("删除用户失败", zap.Error(err), zap.String("uid", uid))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 记录操作日志
	logs.UserOperatorLogChannel <- &model.UserOperatorLog{
		UID:       uid,
		Operator:  model.UserOperatorUpdate,
		Operation: fmt.Sprintf("管理员删除用户"),
		CreatedAt: time.Now().UnixMilli(),
		Creator:   operatorUID,
	}

	encoding.HandleSuccess(c)
}

// setUserRole 设置用户角色
func (h *handler) setUserRole(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req setUserRoleReq
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
	found, _, err := dao.GetUserByUID(ctx, h.dbResolver, req.UID)
	if err != nil {
		zap.L().Error("查询用户失败", zap.Error(err), zap.String("uid", req.UID))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	if !found {
		encoding.HandleError(c, errutil.NewError(http.StatusNotFound, "用户不存在"))
		return
	}

	// 执行更新
	updates := map[string]interface{}{
		"role": req.Role,
	}

	if err := dao.UpdateUser(ctx, h.dbResolver, req.UID, updates); err != nil {
		zap.L().Error("设置用户角色失败", zap.Error(err), zap.String("uid", req.UID))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 记录操作日志
	logs.UserOperatorLogChannel <- &model.UserOperatorLog{
		UID:       req.UID,
		Operator:  model.UserOperatorUpdate,
		Operation: fmt.Sprintf("管理员设置用户角色为 %s", req.Role),
		CreatedAt: time.Now().UnixMilli(),
		Creator:   operatorUID,
	}

	encoding.HandleSuccess(c)
}

// getAdminUsers 获取管理员用户列表
func (h *handler) getAdminUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	// 获取管理员列表
	users, err := dao.GetUsersByRole(ctx, h.dbResolver, model.UserRoleAdmin)
	if err != nil {
		zap.L().Error("获取管理员列表失败", zap.Error(err))
		encoding.HandleError(c, errutil.ErrInternalServer)
		return
	}

	// 转换为响应结构体
	userInfos := make([]UserInfo, len(users))
	for i, user := range users {
		userInfos[i] = UserInfo{
			ID:        user.ID,
			UID:       user.UID,
			Username:  user.Username,
			Role:      user.Role,
			Primary:   user.Primary,
			Tel:       user.Tel,
			Email:     user.Email,
			Desc:      user.Desc,
			Status:    user.Status,
			GidNumber: user.GidNumber,
			CreatedAt: user.CreatedAt,
			Creator:   user.Creator,
			UpdatedAt: user.UpdatedAt,
			Updater:   user.Updater,
		}
	}

	// 返回管理员列表
	encoding.HandleSuccessList(c, int64(len(users)), userInfos)
}

// 以下是虚拟机管理相关的接口

// listVMs 获取虚拟机列表
func (h *handler) listVMs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c, types.DefaultTimeout)
	defer cancel()

	var req vmListReq
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

	// 获取虚拟机列表
	vms, total, err := dao.ListVMs(ctx, h.dbResolver, req.Pagination)
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

// 更多管理员功能...（省略镜像管理等功能的实现）
