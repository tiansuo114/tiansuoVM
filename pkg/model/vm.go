package model

import "gorm.io/gorm"

// VirtualMachine 虚拟机模型（基于Pod实现）
type VirtualMachine struct {
	ID       int64  `gorm:"primary_key;AUTO_INCREMENT"`
	Name     string `gorm:"not null;index:name;type:varchar(64)"`       // VM名称
	UID      string `gorm:"not null;index:uid,unique;type:varchar(32)"` // Pod资源名称
	UserUID  string `gorm:"not null;index:user_uid"`                    // 所属用户
	UserName string `gorm:"not null;index:user_name"`                   // 用户名称

	// 资源配置
	CPU      int32 `gorm:"not null"` // CPU核心数
	MemoryMB int32 `gorm:"not null"` // 内存大小(MB)
	DiskGB   int32 `gorm:"not null"` // 磁盘大小(GB)

	// Pod相关信息
	Status    VMStatus `gorm:"not null;type:varchar(32)"` // VM状态
	PodName   string   `gorm:"not null;type:varchar(64)"` // Pod名称
	Namespace string   `gorm:"not null;type:varchar(64)"` // Pod所在命名空间
	NodeName  string   `gorm:"type:varchar(64)"`          // 运行节点名称

	// 网络信息
	IP          string `gorm:"type:varchar(15)"` // Pod IP地址
	SSHPort     int32  `gorm:"not null"`         // SSH端口号
	ReplicasNum int32  `gorm:"not null"`

	// 镜像信息
	ImageName string `gorm:"not null;type:varchar(64)"` // 使用的镜像名称
	ImageID   int64  `gorm:"not null"`                  // 镜像ID

	// 时间和用户信息
	CreatedAt int64  `gorm:"autoCreateTime:milli;not null;index:idx_created_at"`
	Creator   string `gorm:"not null;type:varchar(32)"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli;not null"`
	Updater   string `gorm:"not null;type:varchar(32)"`

	// 其他信息
	Message string `gorm:"type:varchar(255)"` // 状态信息
	SSHKey  string `gorm:"type:text"`         // SSH公钥
	gorm.DeletedAt
}

// VMStatus 虚拟机状态
type VMStatus string

const (
	VMStatusPending           VMStatus = "pending"             // 创建中
	VMStatusRunning           VMStatus = "running"             // 运行中
	VMStatusStopped           VMStatus = "stopped"             // 已停止
	VMStatusFailed            VMStatus = "failed"              // 失败
	VMStatusTerminating       VMStatus = "terminating"         // 删除中
	VMStatusError             VMStatus = "error"               // 错误
	VMStatusMarkedForDeletion VMStatus = "marked_for_deletion" // 已标记删除
)

func (VirtualMachine) TableName() string {
	return "virtual_machines"
}

// VMOperationLog 虚拟机操作日志
type VMOperationLog struct {
	ID        int64           `gorm:"primary_key;AUTO_INCREMENT"`
	VMID      int64           `gorm:"not null;index:vm_id"`
	UserID    int64           `gorm:"not null;index:user_id"`
	Operation VMOperationType `gorm:"not null;type:varchar(32)"`
	Status    string          `gorm:"not null;type:varchar(32)"`
	Message   string          `gorm:"type:text"`
	CreatedAt int64           `gorm:"autoCreateTime:milli;not null;index:idx_created_at"`
	Creator   string          `gorm:"not null;type:varchar(32)"`
}

type VMOperationType string

const (
	VMOperationCreate  VMOperationType = "create"  // 创建VM
	VMOperationStart   VMOperationType = "start"   // 启动VM
	VMOperationStop    VMOperationType = "stop"    // 停止VM
	VMOperationDelete  VMOperationType = "delete"  // 删除VM
	VMOperationRestart VMOperationType = "restart" // 重启VM
)

func (VMOperationLog) TableName() string {
	return "vm_operation_logs"
}
