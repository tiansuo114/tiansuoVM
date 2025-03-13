package mysql

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm/logger"
)

func TestNewRDBOptions(t *testing.T) {
	options := NewMysqlOptions()
	assert.Equal(t, "root", options.RdbUser)
	assert.Equal(t, "123456", options.RdbPassword)
	assert.Equal(t, "localhost", options.RdbHost)
	assert.Equal(t, 3306, options.RdbPort)
	assert.Equal(t, "async_vm", options.RdbDbname)
	assert.Equal(t, int(logger.Info), options.RdbLogLevel)

	// env test
	t.Setenv("RDB_USER", "fake")
	t.Setenv("RDB_PASSWORD", "fake")
	t.Setenv("RDB_HOST", "fake")
	t.Setenv("RDB_PORT", "1234")
	t.Setenv("RDB_DBNAME", "fake")
	t.Setenv("RDB_READONLY_HOST", "fake")
	t.Setenv("RDB_LOG_LEVEL", "1")

	fs := pflag.NewFlagSet("fake", pflag.ExitOnError)
	options.AddFlags(fs)
	errs := options.Validate()
	assert.Equal(t, len(errs), 0)

	assert.Equal(t, "fake", options.RdbUser)
	assert.Equal(t, "fake", options.RdbPassword)
	assert.Equal(t, "fake", options.RdbHost)
	assert.Equal(t, 1234, options.RdbPort)
	assert.Equal(t, "fake", options.RdbDbname)
	assert.Equal(t, 1, options.RdbLogLevel)

	// command line arguments test
	err := fs.Parse([]string{
		"--rdb-user=fake1",
		"--rdb-password=fake1",
		"--rdb-host=fake1",
		"--rdb-port=1000",
		"--rdb-dbname=fake1",
		"--rdb-log-level=2",
	})
	assert.NoError(t, err)

	assert.Equal(t, "fake1", options.RdbUser)
	assert.Equal(t, "fake1", options.RdbPassword)
	assert.Equal(t, "fake1", options.RdbHost)
	assert.Equal(t, 1000, options.RdbPort)
	assert.Equal(t, "fake1", options.RdbDbname)
	assert.Equal(t, 2, options.RdbLogLevel)
}
