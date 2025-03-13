package ldap

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLDAPOptions(t *testing.T) {
	// Test default options
	opts := NewLDAPOptions()
	assert.Equal(t, "localhost", opts.Host)
	assert.Equal(t, 389, opts.Port)
	assert.Equal(t, "cn=admin,dc=example,dc=com", opts.LDAPUserName)
	assert.Equal(t, "", opts.LDAPPassword)
	assert.Equal(t, "dc=example,dc=com", opts.BaseDN)

	// Test with custom options
	opts = NewLDAPOptions(
		SetDefaultLDAPHost("ldap.example.com"),
		SetDefaultLDAPPort(636),
		SetDefaultLDAPUserName("cn=admin,dc=test,dc=com"),
		SetDefaultLDAPPassword("admin_password"),
	)
	assert.Equal(t, "ldap.example.com", opts.Host)
	assert.Equal(t, 636, opts.Port)
	assert.Equal(t, "cn=admin,dc=test,dc=com", opts.LDAPUserName)
	assert.Equal(t, "admin_password", opts.LDAPPassword)
}

func TestOptions_Validate(t *testing.T) {
	// Test valid options
	opts := NewLDAPOptions(
		SetDefaultLDAPHost("ldap.example.com"),
		SetDefaultLDAPPort(636),
		SetDefaultLDAPUserName("cn=admin,dc=test,dc=com"),
		SetDefaultLDAPPassword("admin_password"),
	)
	errs := opts.Validate()
	assert.Empty(t, errs)

	// Test invalid port
	opts = NewLDAPOptions(
		SetDefaultLDAPPort(70000), // Invalid port
	)
	errs = opts.Validate()
	assert.NotEmpty(t, errs)
	assert.Equal(t, "invalid ldap port", errs[0].Error())

	// Test empty LDAP username
	opts = NewLDAPOptions(
		SetDefaultLDAPUserName(""), // Empty username
	)
	errs = opts.Validate()
	assert.NotEmpty(t, errs)
	assert.Equal(t, "ldap user name is empty", errs[0].Error())

	// Test invalid LDAP username format
	opts = NewLDAPOptions(
		SetDefaultLDAPUserName("admin"), // Invalid format
	)
	errs = opts.Validate()
	assert.NotEmpty(t, errs)
	assert.Equal(t, "invalid ldap user name", errs[0].Error())

	// Test empty LDAP password
	opts = NewLDAPOptions(
		SetDefaultLDAPPassword(""), // Empty password
	)
	errs = opts.Validate()
	assert.NotEmpty(t, errs)
	assert.Equal(t, "ldap password is empty", errs[0].Error())
}

func TestOptions_AddFlags(t *testing.T) {
	opts := NewLDAPOptions()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	opts.AddFlags(fs)

	// Test flag parsing
	err := fs.Parse([]string{
		"--ldap-host=ldap.example.com",
		"--ldap-port=636",
		"--ldap-user-name=cn=admin,dc=test,dc=com",
		"--ldap-password=admin_password",
	})
	require.NoError(t, err)

	assert.Equal(t, "ldap.example.com", opts.Host)
	assert.Equal(t, 636, opts.Port)
	assert.Equal(t, "cn=admin,dc=test,dc=com", opts.LDAPUserName)
	assert.Equal(t, "admin_password", opts.LDAPPassword)
}

func TestOptions_LoadEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("LDAP_HOST", "ldap.example.com")
	os.Setenv("LDAP_PORT", "636")
	os.Setenv("LDAP_USER_NAME", "cn=admin,dc=test,dc=com")
	os.Setenv("LDAP_PASSWORD", "admin_password")

	// Create options and load environment variables
	opts := NewLDAPOptions()
	opts.loadEnv()

	assert.Equal(t, "ldap.example.com", opts.Host)
	assert.Equal(t, 636, opts.Port)
	assert.Equal(t, "cn=admin,dc=test,dc=com", opts.LDAPUserName)
	assert.Equal(t, "admin_password", opts.LDAPPassword)
}
