package controller

import (
	"fmt"

	"tiansuoVM/pkg/client/k8s"
	"tiansuoVM/pkg/dbresolver"
)

// Options VM控制器配置选项
type Options struct {
	// Kubernetes配置
	KubeOpts *k8s.Options
	// 数据库配置
	DatabaseOpts *dbresolver.DBResolver
	// Pod命名空间
	Namespace string
	// SSH端口范围
	SSHPortStart     int32
	SSHPortEnd       int32
	StorageClassName string
	ReplicaNum       int32
}

// Validate 验证配置选项
func (o *Options) Validate() []error {
	var errs []error

	if o.KubeOpts == nil {
		errs = append(errs, fmt.Errorf("kubernetes options cannot be nil"))
	}

	if o.DatabaseOpts == nil {
		errs = append(errs, fmt.Errorf("database options cannot be nil"))
	}

	if o.Namespace == "" {
		errs = append(errs, fmt.Errorf("namespace cannot be empty"))
	}

	if o.SSHPortStart <= 0 || o.SSHPortEnd <= 0 {
		errs = append(errs, fmt.Errorf("ssh port range must be positive"))
	}

	if o.SSHPortStart >= o.SSHPortEnd {
		errs = append(errs, fmt.Errorf("ssh port start must be less than end"))
	}

	return errs
}
