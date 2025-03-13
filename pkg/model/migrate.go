package model

import "gorm.io/gorm"

var GlobalDst = []any{
	&User{},
	&UserOperatorLog{},
	&VirtualMachine{},
	&VMOperationLog{},
	&VMImage{},
	&ImageOperationLog{},
	&AuditLog{},
	&EventLog{},
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(GlobalDst...)
}
