package dbresolver

import (
	"fmt"

	"gorm.io/gorm"

	"tiansuoVM/pkg/client/mysql"
	"tiansuoVM/pkg/model"
)

type DBResolver struct {
	db    *gorm.DB
	dbOpt *mysql.Options
}

func NewDBResolver(dbOpt *mysql.Options) (*DBResolver, error) {
	dr := DBResolver{dbOpt: dbOpt}

	db, err := mysql.NewMysqlClient(dbOpt)
	if err != nil {
		return nil, fmt.Errorf("connect to database error: %w", err)
	}
	db.Logger = NewResolverLogger(db.Logger, "global") // 使用日志记录器
	dr.db = db

	if err = db.AutoMigrate(model.GlobalDst...); err != nil {
		return nil, fmt.Errorf("db.AutoMigrate error: %w", err)
	}

	return &dr, nil
}

func (dr *DBResolver) GetDB() *gorm.DB {
	return dr.db
}

func (dr *DBResolver) Close() error {
	sqlDB, err := dr.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (dr *DBResolver) IteratorDB(f func(db *gorm.DB)) {
	f(dr.db)
}
