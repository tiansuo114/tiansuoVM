package options

import (
	"fmt"
)

// Validate 验证所有选项
func (s *ServerRunOptions) Validate() []error {
	var errs []error

	// 验证通用服务器选项
	errs = append(errs, s.GenericServerRunOptions.Validate()...)

	// 验证缓存选项
	//errs = append(errs, s.CacheOptions.Validate()...)

	// 验证数据库选项
	errs = append(errs, s.RDBOptions.Validate()...)

	// 验证K8s选项
	errs = append(errs, s.K8sOptions.Validate()...)

	errs = append(errs, s.LDAPOptions.Validate()...)

	errs = append(errs, s.LoggerOptions.Validate()...)

	//// 验证命名空间
	//if s.K8sNameSpace == "" {
	//	errs = append(errs, fmt.Errorf("k8s namespace cannot be empty"))
	//}
	//
	//// 验证存储类
	//if s.K8sStorageClass == "" {
	//	errs = append(errs, fmt.Errorf("k8s storage class cannot be empty"))
	//}

	// 验证JWT密钥
	if s.JWTSecret == "" {
		errs = append(errs, fmt.Errorf("jwt secret cannot be empty"))
	}

	return errs
}
