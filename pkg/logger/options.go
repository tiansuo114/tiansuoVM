package logger

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	logLevel      = "log-level"
	logFilename   = "log-filename"
	logMaxSize    = "log-max-size"
	logMaxBackups = "log-max-backups"
	logMaxAge     = "log-max-age"
	logCompress   = "log-compress"
)

type Options struct {
	LogLevel      int
	AddStackLevel int
	Filename      string
	MaxSize       int
	MaxBackups    int
	MaxAge        int
	Compress      bool
	v             *viper.Viper
}

func NewLoggerOptions() *Options {
	o := &Options{
		LogLevel:      int(zap.DebugLevel),
		AddStackLevel: int(zap.FatalLevel),
		Filename:      "",
		MaxSize:       1,
		MaxBackups:    500,
		MaxAge:        180,
		Compress:      false,
		v:             viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))),
	}

	o.v.AutomaticEnv()
	return o
}

func (o *Options) loadEnv() {
	o.LogLevel = o.v.GetInt(logLevel)
}

// Validate check options
func (o *Options) Validate() []error {
	errors := make([]error, 0)

	if o.LogLevel < int(zap.DebugLevel) || o.LogLevel > int(zap.FatalLevel) {
		errors = append(errors, fmt.Errorf("invalid log level"))
	}
	if o.AddStackLevel < int(zap.DebugLevel) || o.AddStackLevel > int(zap.FatalLevel) {
		errors = append(errors, fmt.Errorf("invalid AddStackLevel"))
	}

	return errors
}

// AddFlags add option flags to command line flags,
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&o.LogLevel, logLevel, o.LogLevel, "log level. env LOG_LEVEL")
	fs.StringVar(&o.Filename, logFilename, o.Filename, "log level. env LOG_FILENAME")
	fs.IntVar(&o.MaxSize, logMaxSize, o.MaxSize, "log level. env LOG_MAX_SIZE")
	fs.IntVar(&o.MaxBackups, logMaxBackups, o.MaxBackups, "log level. env LOG_MAX_BACKUPS")
	fs.IntVar(&o.MaxAge, logMaxAge, o.MaxAge, "log level. env LOG_MAX_AGE")
	fs.BoolVar(&o.Compress, logCompress, o.Compress, "log level. env LOG_COMPRESS")

	_ = o.v.BindPFlags(fs)
	o.loadEnv()
}
