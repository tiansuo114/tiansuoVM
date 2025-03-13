package mysql

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gorm.io/gorm/logger"
)

const (
	rdbUser     = "rdb-user"
	rdbPassword = "rdb-password"
	rdbHost     = "rdb-host"
	rdbPort     = "rdb-port"
	rdbDbname   = "rdb-dbname"
	rdbLogLevel = "rdb-log-level"
)

type DefaultOption func(o *Options)

// SetDefaultRdbUser returns a DefaultOption that specifies default RdbUser parameters
func SetDefaultRdbUser(s string) DefaultOption {
	return func(o *Options) {
		o.RdbUser = s
	}
}

// SetDefaultRdbPassword returns a DefaultOption that specifies default RdbPassword parameters
func SetDefaultRdbPassword(s string) DefaultOption {
	return func(o *Options) {
		o.RdbPassword = s
	}
}

// SetDefaultRdbHost returns a DefaultOption that specifies default RdbHost parameters
func SetDefaultRdbHost(s string) DefaultOption {
	return func(o *Options) {
		o.RdbHost = s
	}
}

// SetDefaultRdbPort returns a DefaultOption that specifies default RdbPort parameters
func SetDefaultRdbPort(n int) DefaultOption {
	return func(o *Options) {
		o.RdbPort = n
	}
}

// SetDefaultRdbDbname returns a DefaultOption that specifies default RdbDbname parameters
func SetDefaultRdbDbname(s string) DefaultOption {
	return func(o *Options) {
		o.RdbDbname = s
	}
}

// SetDefaultRdbLogLevel returns a DefaultOption that specifies default RdbLogLevel parameters
func SetDefaultRdbLogLevel(n logger.LogLevel) DefaultOption {
	return func(o *Options) {
		o.RdbLogLevel = int(n)
	}
}

type Options struct {
	RdbUser     string
	RdbPassword string
	RdbHost     string
	RdbPort     int
	RdbDbname   string
	RdbLogLevel int
	v           *viper.Viper
}

func NewMysqlOptions(opts ...DefaultOption) *Options {
	o := &Options{
		RdbUser:     "root",
		RdbPassword: "123456",
		RdbHost:     "localhost",
		RdbPort:     3306,
		RdbDbname:   "async_vm",
		RdbLogLevel: int(logger.Info),
		v:           viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))),
	}

	for _, opt := range opts {
		opt(o)
	}

	o.v.AutomaticEnv()
	return o
}

func (o *Options) loadEnv() {
	o.RdbUser = o.v.GetString(rdbUser)
	o.RdbPassword = o.v.GetString(rdbPassword)
	o.RdbHost = o.v.GetString(rdbHost)
	o.RdbPort = o.v.GetInt(rdbPort)
	o.RdbDbname = o.v.GetString(rdbDbname)
	o.RdbLogLevel = o.v.GetInt(rdbLogLevel)
}

// Validate check options
func (o *Options) Validate() []error {
	errors := make([]error, 0)

	if o.RdbUser == "" {
		errors = append(errors, fmt.Errorf("rdb user is empty"))
	}
	if o.RdbPassword == "" {
		errors = append(errors, fmt.Errorf("rdb password is empty"))
	}
	if o.RdbHost == "" {
		errors = append(errors, fmt.Errorf("rdb host is empty"))
	}
	if o.RdbPort < 0 || o.RdbPort > 65535 {
		errors = append(errors, fmt.Errorf("rdb port is invaild"))
	}
	if o.RdbDbname == "" {
		errors = append(errors, fmt.Errorf("rdb dbname is empty"))
	}

	return errors
}

// AddFlags add option flags to command line flags,
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.RdbUser, rdbUser, o.RdbUser, "env RDB_USER")
	fs.StringVar(&o.RdbPassword, rdbPassword, o.RdbPassword, "env RDB_PASSWORD")
	fs.StringVar(&o.RdbHost, rdbHost, o.RdbHost, "env RDB_HOST")
	fs.IntVar(&o.RdbPort, rdbPort, o.RdbPort, "env RDB_PORT")
	fs.StringVar(&o.RdbDbname, rdbDbname, o.RdbDbname, "env RDB_DBNAME")
	fs.IntVar(&o.RdbLogLevel, rdbLogLevel, o.RdbLogLevel, "logs level. env RDB_LOG_LEVEL")

	_ = o.v.BindPFlags(fs)
	o.loadEnv()
}
