package app

import (
	"context"
	"go.uber.org/zap"
	"tiansuoVM/pkg/auth"
	"tiansuoVM/pkg/helper"
	"tiansuoVM/pkg/model"
	"time"
)

// initSystem 初始化系统
func (s *ConsoleServer) initSystem() (err error) {
	zap.L().Info("正在初始化系统...")
	defer func(t1 time.Time) {
		if err != nil {
			zap.L().Error("系统初始化异常", zap.Duration("耗时", time.Since(t1)), zap.Error(err))
		} else {
			zap.L().Info("系统初始化完成", zap.Duration("耗时", time.Since(t1)))
		}
	}(time.Now())

	// 初始化数据库表
	if err = s.initDatabase(); err != nil {
		return err
	}

	go s.VMCleaner.Start(context.Background())

	err = s.ImageImporter.ImportFromCSV(context.Background())
	if err != nil {
		return err
	}

	groups, err := s.LDAPClient.FindAllGroups()
	if err != nil {
		return err
	}

	ldapGroupCache := make(map[string]string)
	for _, group := range groups {
		ldapGroupCache[group.GIDNumber] = group.CN
	}
	helper.LDAPGroupCache = ldapGroupCache

	auth.Init(s.TokenManager)

	err = s.VMController.InitNodeIpMap()
	if err != nil {
		return err
	}

	//// 初始化默认管理员账户
	//if err = s.initAdminUser(); err != nil {
	//	return err
	//}

	return nil
}

// initDatabase 初始化数据库表
func (s *ConsoleServer) initDatabase() error {
	// 这里可以添加数据库迁移或表创建逻辑
	// 例如使用GORM的AutoMigrate功能
	if err := model.AutoMigrate(s.DBResolver.GetDB()); err != nil {
		return err
	}

	return nil
}

//// initAdminUser 初始化默认管理员账户
//func (s *ConsoleServer) initAdminUser() error {
//	// 检查是否已存在管理员账户，如果不存在则创建
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
//	defer cancel()
//
//	//这里添加创建默认管理员账户的逻辑
//	//例如：
//	//exists, err := dao.CheckAdminExists(ctx, s.DBResolver)
//	if err != nil {
//		return err
//	}
//	if !exists {
//		admin := &model.User{
//			Username: "admin",
//			Password: "admin123", // 应该加密存储
//			Role:   model.UserRoleAdmin,
//		}
//		if err := dao.CreateUser(ctx, s.DBResolver, admin); err != nil {
//			return err
//		}
//		zap.L().Info("已创建默认管理员账户", zap.String("username", admin.Username))
//	}
//
//	return nil
//}
