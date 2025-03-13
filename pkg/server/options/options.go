package options

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	bindAddress    = "bind-address"
	port           = "port"
	tlsCertFile    = "tls-cert-file"
	tlsPrivateKey  = "tls-private-key"
	configFilePath = "config-file-path"
)

const (
	defaultConfigFilePath = "configs/async-km-config.yaml"
)

type ServerRunOptions struct {
	// server bind address
	BindAddress string
	// server port number
	Port int
	// tls cert file
	TlsCertFile string
	// tls private key file
	TlsPrivateKey string
	// config file path
	ConfigFilePath string

	v *viper.Viper
}

func NewServerRunOptions() *ServerRunOptions {
	// create default server run options
	s := ServerRunOptions{
		BindAddress:    "0.0.0.0",
		Port:           9090,
		TlsCertFile:    "",
		TlsPrivateKey:  "",
		ConfigFilePath: defaultConfigFilePath,
		v:              viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))),
	}

	s.v.AutomaticEnv()
	return &s
}

func (s *ServerRunOptions) loadEnv() {
	s.BindAddress = s.v.GetString(bindAddress)
	s.Port = s.v.GetInt(port)
	s.TlsCertFile = s.v.GetString(tlsCertFile)
	s.TlsPrivateKey = s.v.GetString(tlsPrivateKey)
	s.ConfigFilePath = s.v.GetString(configFilePath)
}

func (s *ServerRunOptions) Validate() []error {
	errs := make([]error, 0)

	if s.Port < 0 || s.Port > 65535 {
		errs = append(errs, fmt.Errorf("port invalid"))
	}

	if s.TlsPrivateKey != "" && s.TlsCertFile != "" {
		if _, err := os.Stat(s.TlsCertFile); err != nil {
			errs = append(errs, err)
		}

		if _, err := os.Stat(s.TlsPrivateKey); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (s *ServerRunOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, bindAddress, s.BindAddress, "server bind address. env BIND_ADDRESS")
	fs.IntVar(&s.Port, port, s.Port, "server port number. env PORT")
	fs.StringVar(&s.TlsCertFile, tlsCertFile, s.TlsCertFile, "tls cert file")
	fs.StringVar(&s.TlsPrivateKey, tlsPrivateKey, s.TlsPrivateKey, "tls private key")
	fs.StringVar(&s.ConfigFilePath, configFilePath, s.ConfigFilePath, "config file path")

	_ = s.v.BindPFlags(fs)
	s.loadEnv()
}
