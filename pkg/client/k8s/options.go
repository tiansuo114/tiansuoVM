package k8s

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Define default configuration item names
const (
	kubeConfigPath = "kube-config-path"
	kubeContext    = "kube-context"
	kubeInCluster  = "kube-in-cluster"
)

type DefaultOption func(o *Options)

// SetDefaultKubeConfigPath sets the default path to the kubeconfig file
func SetDefaultKubeConfigPath(s string) DefaultOption {
	return func(o *Options) {
		o.KubeConfigPath = s
	}
}

// SetDefaultKubeContext sets the default Kubernetes context
func SetDefaultKubeContext(s string) DefaultOption {
	return func(o *Options) {
		o.KubeContext = s
	}
}

// SetDefaultKubeInCluster sets whether to use in-cluster configuration
func SetDefaultKubeInCluster(b bool) DefaultOption {
	return func(o *Options) {
		o.InCluster = b
	}
}

// Options stores the Kubernetes-related configuration items
type Options struct {
	KubeConfigPath string `json:"kube_config_path"` // Path to the kubeconfig file
	KubeContext    string `json:"kube_context"`     // Kubernetes context to use
	InCluster      bool   `json:"in_cluster"`       // Whether to use in-cluster configuration
	v              *viper.Viper
}

// NewKubeOptions returns a new Options object with default Kubernetes configurations
func NewKubeOptions(opts ...DefaultOption) *Options {
	// Default kubeconfig path is ~/.kube/config
	defaultKubeConfig := ""
	if home := homedir.HomeDir(); home != "" {
		defaultKubeConfig = filepath.Join(home, ".kube", "config")
	}

	o := &Options{
		KubeConfigPath: defaultKubeConfig,
		KubeContext:    "", // Default context is empty, which means use the current context
		InCluster:      false,
		v:              viper.NewWithOptions(viper.EnvKeyReplacer(strings.NewReplacer("-", "_"))),
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
	o.KubeConfigPath = o.v.GetString(kubeConfigPath) // Get kubeconfig path configuration
	o.KubeContext = o.v.GetString(kubeContext)       // Get Kubernetes context configuration
	o.InCluster = o.v.GetBool(kubeInCluster)         // Get in-cluster configuration flag
}

// Validate checks the validity of configuration items
func (o *Options) Validate() []error {
	var errors []error

	// Validate kubeconfig path if not using in-cluster configuration
	if !o.InCluster && o.KubeConfigPath == "" {
		errors = append(errors, fmt.Errorf("kubeconfig path is empty"))
	}

	return errors
}

// AddFlags adds configuration items to command-line flags
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.KubeConfigPath, kubeConfigPath, o.KubeConfigPath, "Path to the kubeconfig file. If left blank, in-cluster configuration will be used.")
	fs.StringVar(&o.KubeContext, kubeContext, o.KubeContext, "Kubernetes context to use. If left blank, the current context will be used.")
	fs.BoolVar(&o.InCluster, kubeInCluster, o.InCluster, "Whether to use in-cluster configuration.")

	// Bind command-line flags
	_ = o.v.BindPFlags(fs)
	o.loadEnv()
}

// ToRESTConfig converts the Options to a Kubernetes REST config
func (o *Options) ToRESTConfig() (*rest.Config, error) {
	if o.InCluster {
		// Use in-cluster configuration
		return rest.InClusterConfig()
	}

	// Use kubeconfig file
	config, err := clientcmd.BuildConfigFromFlags("", o.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	// Set the context if specified
	if o.KubeContext != "" {
		config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: o.KubeConfigPath},
			&clientcmd.ConfigOverrides{CurrentContext: o.KubeContext},
		).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to set kube context: %w", err)
		}
	}

	return config, nil
}
