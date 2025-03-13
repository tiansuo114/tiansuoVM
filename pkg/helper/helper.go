package helper

import (
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var LDAPGroupCache map[string]string

func Retry(attempts int, delay time.Duration, operation func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = operation(); err == nil {
			return nil
		}
		zap.L().Warn("Operation failed, retrying", zap.Int("attempt", i+1), zap.Error(err))
		time.Sleep(delay)
	}
	return err
}

// GetRootByCaller 通过runtime.Caller方式获取
func GetRootByCaller() string {
	// 获取调用者信息（这里获取helper.go自己的路径）
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	// 典型项目结构示例：
	// your-project/
	// ├── cmd/
	// ├── pkg/
	// │   └── helper/
	// │       └── helper.go <- 当前文件位置
	// └── go.mod

	// 计算到项目根目录的相对路径（根据实际目录结构调整）
	projectRoot := filepath.Join(filepath.Dir(filename), "../..")

	// 解析符号链接并标准化路径
	if cleanedPath, err := filepath.EvalSymlinks(projectRoot); err == nil {
		projectRoot = cleanedPath
	}

	return filepath.Clean(projectRoot)
}

// GetRootByWD 通过工作目录方式获取（备用）
func GetRootByWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// 从当前目录向上查找包含go.mod的目录
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}

		parent := filepath.Dir(wd)
		if parent == wd { // 到达根目录
			break
		}
		wd = parent
	}
	return ""
}
