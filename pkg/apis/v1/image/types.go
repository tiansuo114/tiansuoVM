package image

import (
	"tiansuoVM/pkg/server/request"
)

// 镜像信息结构体
type imageInfo struct {
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

type (
	// 镜像列表请求
	listImageReq struct {
		request.Pagination
		Type   string `form:"type"`   // 按类型过滤
		Public bool   `form:"public"` // 仅显示公共镜像
	}

	// 镜像列表响应
	listImageResp struct {
		Total  int64       `json:"total"`
		Images []imageInfo `json:"images"`
	}

	// 创建镜像请求
	createImageReq struct {
		Name         string `json:"name" binding:"required,max=64"`
		DisplayName  string `json:"display_name" binding:"required,max=128"`
		OSType       string `json:"os_type" binding:"required,max=32"`
		OSVersion    string `json:"os_version" binding:"required,max=32"`
		Architecture string `json:"architecture" binding:"required,max=16"`
		ImageURL     string `json:"image_url" binding:"required,url,max=256"`
		Public       bool   `json:"public"`
		DefaultUser  string `json:"default_user" binding:"required,max=32"`
		Description  string `json:"description" binding:"max=1024"`
	}

	// 更新镜像请求
	updateImageReq struct {
		DisplayName string `json:"display_name" binding:"omitempty,max=128"`
		Description string `json:"description" binding:"omitempty,max=1024"`
		Status      string `json:"status" binding:"omitempty,oneof=available unavailable"`
		Public      *bool  `json:"public"`
	}
)
