package cache

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	redisHost     = "redis-host"     // 使用点号分隔
	redisPassword = "redis-password" // 使用点号分隔
	redisDB       = "redis-db"       // 使用点号分隔
)

type Options struct {
	Host     string `json:"host"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	v        *viper.Viper
}

// NewRedisOptions returns options points to nowhere,
// because redis is not required for some components
func NewRedisOptions() *Options {
	o := &Options{
		Host:     "",
		Password: "",
		DB:       0,
		v:        viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))),
	}

	// Automatically read environment variables
	o.v.AutomaticEnv()
	return o
}

// Validate check options
func (r *Options) Validate() []error {
	errors := make([]error, 0)

	if r.DB > 15 || r.DB < 0 {
		errors = append(errors, fmt.Errorf("invalid redis db"))
	}

	return errors
}

// AddFlags add option flags to command line flags,
// if redis-host left empty, the following options will be ignored.
func (r *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.Host, redisHost, r.Host, "Redis connection URL. If left blank, means redis is unnecessary, redis will be disabled.")
	fs.StringVar(&r.Password, redisPassword, r.Password, "Redis password")
	fs.IntVar(&r.DB, redisDB, r.DB, "Redis db")

	// Bind command-line flags
	_ = r.v.BindPFlags(fs)
	r.loadEnv()
}

// loadEnv loads configuration items from environment variables
func (r *Options) loadEnv() {
	r.Host = r.v.GetString(redisHost)
	r.Password = r.v.GetString(redisPassword)
	r.DB = r.v.GetInt(redisDB)
}
