package k8s

import (
	"github.com/spf13/pflag"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKubeOptions(t *testing.T) {
	// Test default options
	opts := NewKubeOptions()
	assert.Equal(t, filepath.Join(homedir.HomeDir(), ".kube", "config"), opts.KubeConfigPath)
	assert.Equal(t, "", opts.KubeContext)
	assert.Equal(t, false, opts.InCluster)

	// Test with custom options
	opts = NewKubeOptions(
		SetDefaultKubeConfigPath("/custom/path/kubeconfig"),
		SetDefaultKubeContext("my-context"),
		SetDefaultKubeInCluster(true),
	)
	assert.Equal(t, "/custom/path/kubeconfig", opts.KubeConfigPath)
	assert.Equal(t, "my-context", opts.KubeContext)
	assert.Equal(t, true, opts.InCluster)
}

func TestOptions_Validate(t *testing.T) {
	// Test valid options
	opts := NewKubeOptions(
		SetDefaultKubeConfigPath("/custom/path/kubeconfig"),
	)
	errs := opts.Validate()
	assert.Empty(t, errs)

	// Test invalid options (empty kubeconfig path when not in-cluster)
	opts = NewKubeOptions(
		SetDefaultKubeConfigPath(""),
	)
	errs = opts.Validate()
	assert.NotEmpty(t, errs)
	assert.Equal(t, "kubeconfig path is empty", errs[0].Error())
}

func TestOptions_ToRESTConfig(t *testing.T) {
	// Test in-cluster configuration
	opts := NewKubeOptions(
		SetDefaultKubeInCluster(true),
	)

	// Check if in-cluster files exist
	if _, err := os.Stat("/var/run/secrets/k8s.io/serviceaccount/token"); os.IsNotExist(err) {
		t.Skip("in-cluster configuration files not found, skipping test")
	}

	config, err := opts.ToRESTConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Test kubeconfig file configuration
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		t.Skip("kubeconfig file not found, skipping test")
	}

	opts = NewKubeOptions(
		SetDefaultKubeConfigPath(kubeconfigPath),
	)
	config, err = opts.ToRESTConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Test with context
	opts = NewKubeOptions(
		SetDefaultKubeConfigPath(kubeconfigPath),
		SetDefaultKubeContext("my-context"),
	)
	config, err = opts.ToRESTConfig()
	require.NoError(t, err)
	assert.NotNil(t, config)
}

func TestOptions_AddFlags(t *testing.T) {
	opts := NewKubeOptions()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	opts.AddFlags(fs)

	// Test flag parsing
	err := fs.Parse([]string{
		"--kube-config-path=/custom/path/kubeconfig",
		"--kube-context=my-context",
		"--kube-in-cluster=true",
	})
	require.NoError(t, err)

	assert.Equal(t, "/custom/path/kubeconfig", opts.KubeConfigPath)
	assert.Equal(t, "my-context", opts.KubeContext)
	assert.Equal(t, true, opts.InCluster)
}
