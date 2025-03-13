package mysql

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewMysqlClient create a gorm mysql client
func NewMysqlClient(options *Options) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		options.RdbUser, options.RdbPassword, options.RdbHost, options.RdbPort, options.RdbDbname)

	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		return nil, err
	}
	db.Logger = db.Logger.LogMode(logger.LogLevel(options.RdbLogLevel))

	return db, nil
}
