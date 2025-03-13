package vm

import (
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/server/request"
)

// 虚拟机信息结构体
type VMInfo struct {
	ID        int64          `json:"id"`
	Name      string         `json:"name"`
	UID       string         `json:"uid"`
	UserUID   string         `json:"user_uid"`
	UserName  string         `json:"user_name"`
	CPU       int32          `json:"cpu"`
	MemoryMB  int32          `json:"memory_mb"`
	DiskGB    int32          `json:"disk_gb"`
	Status    model.VMStatus `json:"status"`
	PodName   string         `json:"pod_name"`
	Namespace string         `json:"namespace"`
	NodeName  string         `json:"node_name"`
	IP        string         `json:"ip"`
	SSHPort   int32         `json:"ssh_port"`
	ImageName string         `json:"image_name"`
	ImageID   int64         `json:"image_id"`
	Message   string         `json:"message,omitempty"`
	CreatedAt int64         `json:"created_at"`
	Creator   string        `json:"creator"`
	UpdatedAt int64         `json:"updated_at"`
	Updater   string        `json:"updater"`
}

// listVMReq 列出VM的请求参数
type listVMReq struct {
	request.Pagination
}

// listVMResp 列出VM的响应
type listVMResp struct {
	Total int64    `json:"total"`
	VMs   []VMInfo `json:"vms"`
}

// createVMReq 创建VM的请求参数
type createVMReq struct {
	Name    string `json:"name" validate:"required,max=64"`
	CPU     int    `json:"cpu" validate:"required,min=1,max=32"`
	Memory  int    `json:"memory" validate:"required,min=512,max=65536"` // 内存大小（MB）
	Disk    int    `json:"disk" validate:"required,min=10,max=1000"`     // 磁盘大小（GB）
	ImageID string `json:"image_id" validate:"required"`                 // 镜像ID
	SSHKey  string `json:"ssh_key,omitempty"`                            // SSH公钥
}

// deleteVMReq 删除VM的请求参数
type deleteVMReq struct {
	ID int64 `uri:"id" validate:"required,min=1"`
}

// getVMReq 获取VM详情的请求参数
type getVMReq struct {
	ID int64 `uri:"id" validate:"required,min=1"`
}

// startVMReq 启动VM的请求参数
type startVMReq struct {
	ID int64 `uri:"id" validate:"required,min=1"`
}

// stopVMReq 停止VM的请求参数
type stopVMReq struct {
	ID int64 `uri:"id" validate:"required,min=1"`
}

// restartVMReq 重启VM的请求参数
type restartVMReq struct {
	ID int64 `uri:"id" validate:"required,min=1"`
}

// recoverVMReq 恢复已标记删除的VM的请求参数
type recoverVMReq struct {
	ID int64 `uri:"id" validate:"required,min=1"`
}

// 创建虚拟机请求
type adminCreateVMReq struct {
	Name          string `json:"name" binding:"required"`
	CPU           int    `json:"cpu" binding:"required,min=1"`
	Memory        int    `json:"memory" binding:"required,min=1024"`
	Disk          int    `json:"disk" binding:"required,min=10"`
	ImageID       string `json:"image_id" binding:"required"`
	OwnerUsername string `json:"owner_username" binding:"required"`
}

