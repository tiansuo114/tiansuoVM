package dao

import (
	"context"
	"errors"
	"tiansuoVM/pkg/server/request"
	"time"

	"gorm.io/gorm"

	"tiansuoVM/pkg/dbresolver"
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/token"
)

// InsertVM creates a new virtual machine record in the database.
func InsertVM(ctx context.Context, dbResolver *dbresolver.DBResolver, vm *model.VirtualMachine) error {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	return db.WithContext(ctx).Create(vm).Error
}

// checkUserPermission checks if the user has permission to access the VM
func checkUserPermission(ctx context.Context, db *gorm.DB, vm *model.VirtualMachine) (bool, error) {
	payload, err := token.PayloadFromCtx(ctx)
	if err != nil {
		return false, err
	}

	// 如果是管理员，直接返回true
	if payload.Role == model.UserRoleAdmin {
		return true, nil
	}

	// 检查是否是VM的所有者
	return vm.UserUID == payload.UID, nil
}

// GetVMByID retrieves a virtual machine record by its ID.
func GetVMByID(ctx context.Context, dbResolver *dbresolver.DBResolver, id int64) (*model.VirtualMachine, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	vm := &model.VirtualMachine{}
	err := db.WithContext(ctx).Where("id = ?", id).First(vm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return vm, nil
}

// GetVMByUID retrieves a virtual machine record by its UID.
func GetVMByUID(ctx context.Context, dbResolver *dbresolver.DBResolver, uid string) (*model.VirtualMachine, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	vm := &model.VirtualMachine{}
	err := db.WithContext(ctx).Where("uid = ?", uid).First(vm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return vm, nil
}

// GetVMByPodName retrieves a virtual machine record by its pod name.
func GetVMByPodName(ctx context.Context, dbResolver *dbresolver.DBResolver, podName string) (*model.VirtualMachine, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	vm := &model.VirtualMachine{}
	err := db.WithContext(ctx).Where("pod_name = ?", podName).First(vm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return vm, nil
}

// GetVMByName retrieves a virtual machine record by its name.
func GetVMByName(ctx context.Context, dbResolver *dbresolver.DBResolver, name string) (bool, *model.VirtualMachine, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	vm := model.VirtualMachine{}
	err := db.WithContext(ctx).Where("name = ?", name).First(&vm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}

	// 检查用户权限
	hasPermission, err := checkUserPermission(ctx, db, &vm)
	if err != nil {
		return false, nil, err
	}
	if !hasPermission {
		return false, nil, errors.New("permission denied")
	}

	return true, &vm, nil
}

// UpdateVMByID 更新vm信息
func UpdateVMByID(ctx context.Context, dbResolver *dbresolver.DBResolver, id int64, updates map[string]interface{}) error {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	updater := token.GetUIDFromCtx(ctx)
	if updater != "" {
		updates["updater"] = updater
	}
	updates["updated_at"] = time.Now().UnixMilli()

	return db.WithContext(ctx).Where("id = ?", id).Updates(updates).Error
}

func UpdateVMByUid(ctx context.Context, dbResolver *dbresolver.DBResolver, uid string, updates map[string]interface{}) error {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	updater := token.GetUIDFromCtx(ctx)
	if updater != "" {
		updates["updater"] = updater
	}
	updates["updated_at"] = time.Now().UnixMilli()

	return db.WithContext(ctx).Where("uid = ?", uid).Updates(updates).Error
}

// DeleteVM deletes a virtual machine record by its ID.
func DeleteVM(ctx context.Context, dbResolver *dbresolver.DBResolver, id int64) error {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	return db.WithContext(ctx).Delete(&model.VirtualMachine{ID: id}).Error
}

func ListVMsWithNoPagination(ctx context.Context, dbResolver *dbresolver.DBResolver) ([]model.VirtualMachine, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var vms []model.VirtualMachine
	err := db.WithContext(ctx).Find(&vms).Error
	return vms, err
}

// ListVMs retrieves all virtual machine records from the database.
func ListVMs(ctx context.Context, dbResolver *dbresolver.DBResolver, pagination request.Pagination) ([]model.VirtualMachine, int64, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var vms []model.VirtualMachine

	var count int64
	err := db.WithContext(context.Background()).Count(&count).Error
	if err != nil {
		return vms, 0, err
	}

	err = pagination.MakeSQL(db.WithContext(ctx)).Find(&vms).Error
	return vms, count, err
}

// ListVMsByUserUID retrieves all virtual machine records belonging to a specific user.
func ListVMsByUserUID(ctx context.Context, dbResolver *dbresolver.DBResolver, userUID string, pagination request.Pagination) ([]model.VirtualMachine, int64, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var vms []model.VirtualMachine
	var count int64
	db = db.WithContext(ctx).Where("user_uid = ?", userUID)

	err := db.WithContext(context.Background()).Count(&count).Error
	if err != nil {
		return vms, 0, err
	}

	err = pagination.MakeSQL(db).Find(&vms).Error
	return vms, count, nil
}

// ListVMsByStatus retrieves all virtual machine records with a specific status.
func ListVMsByStatus(ctx context.Context, dbResolver *dbresolver.DBResolver, status model.VMStatus, pagination request.Pagination) ([]model.VirtualMachine, int64, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{}).Model(&model.VirtualMachine{})
	var vms []model.VirtualMachine
	db = db.WithContext(ctx).Where("status = ?", status)

	var count int64
	err := db.WithContext(context.Background()).Count(&count).Error
	if err != nil {
		return vms, 0, err
	}

	err = pagination.MakeSQL(db).Find(&vms).Error
	return vms, count, err
}

func ListVMsByStatusWithNoPagination(ctx context.Context, dbResolver *dbresolver.DBResolver, status model.VMStatus) ([]model.VirtualMachine, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var vms []model.VirtualMachine
	db = db.WithContext(ctx).Where("status = ?", status)

	err := db.Find(&vms).Error
	return vms, err
}

// CountVMsByUserUID counts the number of virtual machine records belonging to a specific user.
func CountVMsByUserUID(ctx context.Context, dbResolver *dbresolver.DBResolver, userUID string) (int64, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var count int64
	err := db.WithContext(ctx).Where("user_uid = ?", userUID).Count(&count).Error
	return count, err
}

// ListUserVMs retrieves all virtual machine records for the current user.
func ListUserVMs(ctx context.Context, dbResolver *dbresolver.DBResolver, pagination request.Pagination) ([]model.VirtualMachine, int64, error) {
	payload, err := token.PayloadFromCtx(ctx)
	if err != nil {
		return nil, 0, err
	}

	return ListVMsByUserUID(ctx, dbResolver, payload.UID, pagination)
}

// ListUserVMsByStatus retrieves all virtual machine records with a specific status for the current user.
func ListUserVMsByStatus(ctx context.Context, dbResolver *dbresolver.DBResolver, status model.VMStatus, pagination request.Pagination) ([]model.VirtualMachine, int64, error) {
	payload, err := token.PayloadFromCtx(ctx)
	if err != nil {
		return nil, 0, err
	}

	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var vms []model.VirtualMachine
	db = db.WithContext(ctx).Where("user_uid = ? AND status = ?", payload.UID, status)

	var count int64
	err = db.WithContext(context.Background()).Count(&count).Error
	if err != nil {
		return vms, 0, err
	}

	err = pagination.MakeSQL(db).Find(&vms).Error
	return vms, count, err
}

// CountUserVMs counts the number of virtual machine records for the current user.
func CountUserVMs(ctx context.Context, dbResolver *dbresolver.DBResolver) (int64, error) {
	payload, err := token.PayloadFromCtx(ctx)
	if err != nil {
		return 0, err
	}

	return CountVMsByUserUID(ctx, dbResolver, payload.UID)
}

// CheckSSHPortExists checks if a SSH port is already in use.
func CheckSSHPortExists(ctx context.Context, dbResolver *dbresolver.DBResolver, port int32) (bool, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var count int64
	err := db.WithContext(ctx).Where("ssh_port = ?", port).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// InsertVMOperationLog creates a new virtual machine operation log.
func InsertVMOperationLog(ctx context.Context, dbResolver *dbresolver.DBResolver, log *model.VMOperationLog) error {
	db := dbResolver.GetDB().Model(&model.VMOperationLog{})
	return db.WithContext(ctx).Create(log).Error
}

// ListVMOperationLogs retrieves a list of virtual machine operation logs.
func ListVMOperationLogs(ctx context.Context, dbResolver *dbresolver.DBResolver, vmID int64) ([]*model.VMOperationLog, error) {
	db := dbResolver.GetDB().Model(&model.VMOperationLog{})
	var logs []*model.VMOperationLog
	err := db.WithContext(ctx).Where("vm_id = ?", vmID).Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

// ListVMsByActiveStatus 获取所有活跃状态的VM列表
func ListVMsByActiveStatus(ctx context.Context, dbResolver *dbresolver.DBResolver) ([]model.VirtualMachine, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var vms []model.VirtualMachine
	err := db.WithContext(ctx).Where("status IN ?", []model.VMStatus{
		model.VMStatusPending,
		model.VMStatusRunning,
	}).Find(&vms).Error
	if err != nil {
		return nil, err
	}
	return vms, nil
}

// GetVMsByStatusAndDeadline 获取指定状态且更新时间早于截止时间的VM列表
func GetVMsByStatusAndDeadline(ctx context.Context, dbResolver *dbresolver.DBResolver, status model.VMStatus, deadline int64) ([]*model.VirtualMachine, error) {
	db := dbResolver.GetDB().Model(&model.VirtualMachine{})
	var vms []*model.VirtualMachine
	err := db.WithContext(ctx).Where("status = ? AND updated_at <= ?", status, deadline).Find(&vms).Error
	if err != nil {
		return nil, err
	}
	return vms, nil
}
