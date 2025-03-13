package controller

import (
	"fmt"
	"time"

	"tiansuoVM/pkg/client/k8s"
	"tiansuoVM/pkg/dbresolver"
)

// Options defines the configuration options for the VM controller
type Options struct {
	// KubeOpts is the configuration for Kubernetes client
	KubeOpts *k8s.Options

	// Namespace is the namespace where VMs will be created
	Namespace string

	// SyncPeriod is the period for syncing the VM status from Kubernetes to the database
	SyncPeriod time.Duration

	// WaitTimeout is the timeout for waiting for VM operations to complete
	WaitTimeout time.Duration

	// DatabaseOptions is the configuration for the database client
	DatabaseOpts *dbresolver.DBResolver

	// ImagePullPolicy is the policy for pulling container images
	ImagePullPolicy string

	// NodeSelector is the label selector for nodes where VMs can run
	NodeSelector map[string]string

	// ServiceCIDR is the CIDR for VM services
	ServiceCIDR string

	// DefaultStorageClass is the default storage class for VM volumes
	DefaultStorageClass string

	// DefaultNetworkInterface is the default network interface for VMs
	DefaultNetworkInterface string
}

// NewDefaultOptions returns a new Options instance with default values
func NewDefaultOptions() *Options {
	return &Options{
		KubeOpts:               k8s.NewKubeOptions(),
		Namespace:              "default",
		SyncPeriod:             time.Minute * 5,
		WaitTimeout:            time.Minute * 10,
		ImagePullPolicy:        "IfNotPresent",
		NodeSelector:           make(map[string]string),
		ServiceCIDR:            "10.96.0.0/12",
		DefaultStorageClass:    "standard",
		DefaultNetworkInterface: "eth0",
		// DatabaseOpts需要在使用时通过外部传入，默认为nil
		// 控制器初始化时必须提供此选项
	}
}

// Validate validates the controller options
func (o *Options) Validate() []error {
	var errs []error

	// Validate Kubernetes options
	if o.KubeOpts == nil {
		errs = append(errs, fmt.Errorf("kubernetes options cannot be nil"))
	} else if validErrs := o.KubeOpts.Validate(); len(validErrs) > 0 {
		errs = append(errs, validErrs...)
	}

	// Validate namespace
	if o.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace cannot be empty"))
	}

	// Validate sync period
	if o.SyncPeriod <= 0 {
		errs = append(errs, fmt.Errorf("sync period must be positive"))
	}

	// Validate wait timeout
	if o.WaitTimeout <= 0 {
		errs = append(errs, fmt.Errorf("wait timeout must be positive"))
	}

	// Validate database options
	if o.DatabaseOpts == nil {
		errs = append(errs, fmt.Errorf("database options cannot be nil"))
	}

	return errs
}
