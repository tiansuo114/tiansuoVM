package ldap

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"regexp"
	"strings"
)

// Define default configuration item names
const (
	ldapHost     = "ldap-host"
	ldapPort     = "ldap-port"
	ldapUserName = "ldap-user-name"
	ldapPassword = "ldap-password"
	ldapBaseDN   = "ldap-base-dn"
)

type DefaultOption func(o *Options)

// SetDefaultLDAPHost Set default LDAP configuration
func SetDefaultLDAPHost(s string) DefaultOption {
	return func(o *Options) {
		o.Host = s
	}
}

func SetDefaultLDAPPort(n int) DefaultOption {
	return func(o *Options) {
		o.Port = n
	}
}

func SetDefaultLDAPUserName(s string) DefaultOption {
	return func(o *Options) {
		o.LDAPUserName = s
	}
}

func SetDefaultLDAPPassword(s string) DefaultOption {
	return func(o *Options) {
		o.LDAPPassword = s
	}
}

// Options stores the LDAP-related configuration items
type Options struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	LDAPUserName string `json:"ldap_user_name"`
	LDAPPassword string `json:"ldap_password"`
	BaseDN       string `json:"base_dn"`
	v            *viper.Viper
}

// NewLDAPOptions returns a new Options object with default LDAP configurations
func NewLDAPOptions(opts ...DefaultOption) *Options {
	o := &Options{
		Host:         "localhost",                  // Default LDAP host
		Port:         389,                          // Default port
		LDAPUserName: "cn=admin,dc=example,dc=com", // Default username
		LDAPPassword: "",                           // Default no password
		BaseDN:       "dc=example,dc=com",
		v:            viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))),
	}

	// Modify the configuration using DefaultOption
	for _, opt := range opts {
		opt(o)
	}

	// Automatically read environment variables
	o.v.AutomaticEnv()
	return o
}

// loadEnv loads configuration items from environment variables
func (o *Options) loadEnv() {
	o.Host = o.v.GetString(ldapHost)             // Get LDAP host configuration
	o.Port = o.v.GetInt(ldapPort)                // Get LDAP port configuration
	o.LDAPUserName = o.v.GetString(ldapUserName) // Get LDAP username configuration
	o.LDAPPassword = o.v.GetString(ldapPassword) // Get LDAP password configuration
	o.BaseDN = o.v.GetString(ldapBaseDN)
}

// Validate checks the validity of configuration items
func (o *Options) Validate() []error {
	var errors []error

	// Validate port
	if o.Port < 0 || o.Port > 65535 {
		errors = append(errors, fmt.Errorf("invalid ldap port"))
	}

	// Validate LDAP username
	if o.LDAPUserName == "" {
		errors = append(errors, fmt.Errorf("ldap user name is empty"))
	}
	re := regexp.MustCompile(`^cn=[^,]+(,dc=[^,]+)+$`)
	if !re.MatchString(o.LDAPUserName) {
		errors = append(errors, fmt.Errorf("invalid ldap user name"))
	}

	if o.BaseDN == "" {
		errors = append(errors, fmt.Errorf("ldap base dn is empty"))
	}

	return errors
}

// AddFlags adds configuration items to command-line flags
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Host, ldapHost, o.Host, "ldap connection URL. If left blank, means ldap is unnecessary.")
	fs.IntVar(&o.Port, ldapPort, o.Port, "ldap port")
	fs.StringVar(&o.LDAPUserName, ldapUserName, o.LDAPUserName, "ldap user name")
	fs.StringVar(&o.LDAPPassword, ldapPassword, o.LDAPPassword, "ldap password")
	fs.StringVar(&o.BaseDN, ldapBaseDN, o.BaseDN, "ldap base dn")

	// Bind command-line flags
	_ = o.v.BindPFlags(fs)
	o.loadEnv()
}
