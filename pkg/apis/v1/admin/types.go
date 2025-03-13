package admin

import (
	"tiansuoVM/pkg/model"
	"tiansuoVM/pkg/server/request"
)

type (
	// 用户信息结构体
	UserInfo struct {
		ID        int64            `json:"id"`
		UID       string           `json:"uid"`
		Username  string           `json:"username"`
		Role      model.UserRole   `json:"role"`
		Primary   bool             `json:"primary"`
		Tel       string           `json:"tel"`
		Email     string           `json:"email"`
		Desc      string           `json:"desc"`
		Status    model.UserStatus `json:"status"`
		GidNumber string           `json:"gid_number,omitempty"`
		CreatedAt int64            `json:"created_at"`
		Creator   string           `json:"creator"`
		UpdatedAt int64            `json:"updated_at"`
		Updater   string           `json:"updater"`
	}

	// 虚拟机信息结构体
	VMInfo struct {
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

	// 镜像信息结构体
	ImageInfo struct {
		ID           int64  `json:"id"`
		Name         string `json:"name"`
		DisplayName  string `json:"display_name"`
		OSType       string `json:"os_type"`
		OSVersion    string `json:"os_version"`
		Architecture string `json:"architecture"`
		ImageURL     string `json:"image_url"`
		Public       bool   `json:"public"`
		DefaultUser  string `json:"default_user"`
		Description  string `json:"description"`
		Status       string `json:"status"`
		CreatedAt    int64  `json:"created_at"`
		Creator      string `json:"creator"`
		UpdatedAt    int64  `json:"updated_at"`
		Updater      string `json:"updater"`
	}

	// 用户列表请求和响应
	userListReq struct {
		Page  int `form:"page" json:"page"`
		Limit int `form:"limit" json:"limit"`

		request.Pagination
	}

	userListResp struct {
		Total int64      `json:"total"`
		Users []UserInfo `json:"users"`
	}

	// 更新用户请求
	updateUserReq struct {
		Tel    string           `json:"tel" binding:"omitempty"`
		Email  string           `json:"email" binding:"omitempty,email"`
		Desc   string           `json:"desc" binding:"omitempty"`
		Role   model.UserRole   `json:"role" binding:"omitempty"`
		Status model.UserStatus `json:"status" binding:"omitempty"`
	}

	// 设置用户角色请求
	setUserRoleReq struct {
		UID  string         `json:"uid" binding:"required"`
		Role model.UserRole `json:"role" binding:"required"`
	}

	// 虚拟机列表请求和响应
	vmListReq struct {
		Page  int    `form:"page" json:"page"`
		Limit int    `form:"limit" json:"limit"`
		UID   string `form:"uid" json:"uid"`

		request.Pagination
	}

	vmListResp struct {
		Total int64    `json:"total"`
		VMs   []VMInfo `json:"vms"`
	}

	// 更新虚拟机请求
	updateVMReq struct {
		CPU    int    `json:"cpu" binding:"omitempty,min=1"`
		Memory int    `json:"memory" binding:"omitempty,min=1024"`
		Disk   int    `json:"disk" binding:"omitempty,min=10"`
		Status string `json:"status" binding:"omitempty"`
	}

	// 镜像列表请求和响应
	imageListReq struct {
		request.Pagination
	}

	imageListResp struct {
		Total  int64       `json:"total"`
		Images []ImageInfo `json:"images"`
	}

	// 创建镜像请求
	createImageReq struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description" binding:"required"`
		URL         string `json:"url" binding:"required,url"`
		OSType      string `json:"os_type" binding:"required"`
		Version     string `json:"version" binding:"required"`
	}

	// 更新镜像请求
	updateImageReq struct {
		Name        string `json:"name" binding:"omitempty"`
		Description string `json:"description" binding:"omitempty"`
		URL         string `json:"url" binding:"omitempty,url"`
		OSType      string `json:"os_type" binding:"omitempty"`
		Version     string `json:"version" binding:"omitempty"`
		Status      string `json:"status" binding:"omitempty"`
	}
)
