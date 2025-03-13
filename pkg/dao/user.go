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

// GetUserByUID 根据UID获取用户信息
func GetUserByUID(ctx context.Context, dbResolver *dbresolver.DBResolver, uid string) (bool, *model.User, error) {
	db := dbResolver.GetDB().Model(&model.User{})
	user := model.User{}
	err := db.WithContext(ctx).Where("uid = ?", uid).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &user, nil
}

// GetUserByUsername 根据用户名获取用户信息
func GetUserByUsername(ctx context.Context, dbResolver *dbresolver.DBResolver, username string) (bool, *model.User, error) {
	db := dbResolver.GetDB().Model(&model.User{})
	user := model.User{}
	err := db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &user, nil
}

// CreateUser 创建新用户
func CreateUser(ctx context.Context, dbResolver *dbresolver.DBResolver, user *model.User) error {
	db := dbResolver.GetDB().Model(&model.User{})
	creator := token.GetUIDFromCtx(ctx)
	if creator == "" {
		creator = user.UID // 如果没有上下文中的UID，使用用户自己的UID
	}

	user.Creator = creator
	user.Updater = creator
	user.CreatedAt = time.Now().UnixMilli()
	user.UpdatedAt = time.Now().UnixMilli()

	return db.WithContext(ctx).Create(user).Error
}

// UpdateUser 更新用户信息
func UpdateUser(ctx context.Context, dbResolver *dbresolver.DBResolver, uid string, updates map[string]interface{}) error {
	db := dbResolver.GetDB().Model(&model.User{})
	updater := token.GetUIDFromCtx(ctx)
	if updater != "" {
		updates["updater"] = updater
	}
	updates["updated_at"] = time.Now().UnixMilli()

	return db.WithContext(ctx).Model(&model.User{}).Where("uid = ?", uid).Updates(updates).Error
}

// ListUsers 获取所有用户列表
func ListUsers(ctx context.Context, dbResolver *dbresolver.DBResolver, pagination request.Pagination) ([]model.User, int64, error) {
	db := dbResolver.GetDB().Model(&model.User{})
	var users []model.User
	var total int64

	// 获取总数
	if err := db.WithContext(ctx).Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := pagination.MakeSQL(db).Find(&users).Error
	return users, total, err
}

// GetUsersByRole 根据角色获取用户列表
func GetUsersByRole(ctx context.Context, dbResolver *dbresolver.DBResolver, role model.UserRole) ([]model.User, error) {
	db := dbResolver.GetDB().Model(&model.User{})
	var users []model.User
	err := db.WithContext(ctx).Where("role = ?", role).Find(&users).Error
	return users, err
}

// DeleteUser 删除用户
func DeleteUser(ctx context.Context, dbResolver *dbresolver.DBResolver, uid string) error {
	db := dbResolver.GetDB().Model(&model.User{})
	return db.WithContext(ctx).Where("uid = ?", uid).Delete(&model.User{}).Error
}

// GetUserByID 根据ID获取用户信息
func GetUserByID(ctx context.Context, dbResolver *dbresolver.DBResolver, id int64) (bool, *model.User, error) {
	db := dbResolver.GetDB().Model(&model.User{})
	user := model.User{}
	err := db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, &user, nil
}

// InsertUserOperatorLog inserts a new user operation log into the database.
func InsertUserOperatorLog(ctx context.Context, dbResolver *dbresolver.DBResolver, uid string, operator model.UserOperatorType) (*model.UserOperatorLog, error) {
	db := dbResolver.GetDB().Model(&model.UserOperatorLog{})
	return InsertUserOperatorLogWithDB(ctx, db, uid, operator)
}

func InsertUserOperatorLogWithDB(ctx context.Context, db *gorm.DB, uid string, operator model.UserOperatorType) (*model.UserOperatorLog, error) {
	creator := token.GetUIDFromCtx(ctx)
	log := model.UserOperatorLog{
		UID:       uid,
		Operator:  operator,
		CreatedAt: time.Now().UnixMilli(),
		Creator:   creator,
	}

	err := db.WithContext(ctx).Create(&log).Error
	return &log, err
}

func InsertUserOperatorLogByModel(ctx context.Context, dbResolver *dbresolver.DBResolver, log *model.UserOperatorLog) error {
	db := dbResolver.GetDB().Model(&model.UserOperatorLog{})
	return InsertUserOperatorLogByModelWithDB(ctx, db, log)
}

func InsertUserOperatorLogByModelWithDB(ctx context.Context, db *gorm.DB, log *model.UserOperatorLog) error {
	return db.WithContext(ctx).Create(log).Error
}

// GetUserOperatorLogsByUID retrieves all user operation logs for a specific user by UID.
func GetUserOperatorLogsByUID(ctx context.Context, dbResolver *dbresolver.DBResolver, uid string) ([]model.UserOperatorLog, error) {
	db := dbResolver.GetDB().Model(&model.UserOperatorLog{})
	var logs []model.UserOperatorLog
	err := db.WithContext(ctx).Where("uid = ?", uid).Find(&logs).Error
	return logs, err
}

// GetUserOperatorLogByID retrieves a user operation log by its ID.
func GetUserOperatorLogByID(ctx context.Context, dbResolver *dbresolver.DBResolver, id int64) (*model.UserOperatorLog, error) {
	db := dbResolver.GetDB().Model(&model.UserOperatorLog{})
	log := model.UserOperatorLog{}
	err := db.WithContext(ctx).Where("id = ?", id).First(&log).Error
	return &log, err
}

// ListUserOperatorLogs retrieves all user operation logs.
func ListUserOperatorLogs(ctx context.Context, dbResolver *dbresolver.DBResolver, pagination request.Pagination) ([]model.UserOperatorLog, int64, error) {
	db := dbResolver.GetDB().WithContext(ctx).Model(&model.UserOperatorLog{})
	var logs []model.UserOperatorLog

	var count int64
	err := db.WithContext(context.Background()).Count(&count).Error
	if err != nil {
		return logs, 0, err
	}

	err = pagination.MakeSQL(db).Find(&logs).Error

	return logs, count, err
}

func FindUserOperatorLogsByUid(ctx context.Context, dbResolver *dbresolver.DBResolver, uid string) ([]model.UserOperatorLog, error) {
	db := dbResolver.GetDB()
	var logs []model.UserOperatorLog
	err := db.WithContext(ctx).Where("uid = ?", uid).Find(&logs).Error
	return logs, err
}
