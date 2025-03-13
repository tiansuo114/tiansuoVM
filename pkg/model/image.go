package model

import "gorm.io/gorm"

// VMImage 虚拟机操作系统镜像模型
type VMImage struct {
	ID              int64       `gorm:"primary_key;AUTO_INCREMENT"`
	Name            string      `gorm:"not null;index:name,unique;type:varchar(64)"` // 镜像名称，用于在系统中标识
	DisplayName     string      `gorm:"type:varchar(128)"`                           // 显示名称
	OSType          string      `gorm:"not null;type:varchar(32)"`                   // 操作系统类型：Linux, Windows等
	OSVersion       string      `gorm:"not null;type:varchar(32)"`                   // 操作系统版本：Ubuntu 20.04, CentOS 7等
	Architecture    string      `gorm:"not null;type:varchar(16)"`                   // 架构：x86_64, arm64等
	ImageURL        string      `gorm:"not null;type:varchar(256)"`                  // 容器镜像地址
	Status          ImageStatus `gorm:"not null;type:varchar(32);default:'available'"`
	Public          bool        `gorm:"not null;default:false"` // 是否为公共镜像
	DefaultUser     string      `gorm:"type:varchar(32)"`       // 默认用户名
	DefaultPassword string      `gorm:"type:varchar(128)"`
	DefaultSSHKey   string      `gorm:"type:varchar(2048)"` // 默认SSH密钥
	Description     string      `gorm:"type:text"`          // 镜像描述
	PictureUrl      string      `gorm:"type:varchar(256)"`
	CreatedAt       int64       `gorm:"autoCreateTime:milli;not null;index:idx_created_at"`
	Creator         string      `gorm:"not null;type:varchar(32)"`
	UpdatedAt       int64       `gorm:"autoUpdateTime:milli;not null"`
	Updater         string      `gorm:"not null;type:varchar(32)"`
	gorm.DeletedAt
}

type ImageType string

const (
	ImageTypeSystem   ImageType = "system"   // 系统镜像
	ImageTypeCustom   ImageType = "custom"   // 自定义镜像
	ImageTypeUploaded ImageType = "uploaded" // 用户上传的镜像
)

type ImageStatus string

const (
	ImageStatusAvailable   ImageStatus = "available"   // 可用
	ImageStatusUnavailable ImageStatus = "unavailable" // 不可用
)

func (VMImage) TableName() string {
	return "vm_images"
}

// ImageOperationLog 镜像操作日志
type ImageOperationLog struct {
	ID        int64              `gorm:"primary_key;AUTO_INCREMENT"`
	ImageID   int64              `gorm:"not null;index:image_id"`
	UserID    int64              `gorm:"not null;index:user_id"`
	Operation ImageOperationType `gorm:"not null;type:varchar(32)"`
	Status    string             `gorm:"not null;type:varchar(32)"`
	Message   string             `gorm:"type:text"`
	CreatedAt int64              `gorm:"autoCreateTime:milli;not null;index:idx_created_at"`
	Creator   string             `gorm:"not null;type:varchar(32)"`
}

type ImageOperationType string

const (
	ImageOperationCreate ImageOperationType = "create" // 创建镜像
	ImageOperationDelete ImageOperationType = "delete" // 删除镜像
	ImageOperationUpdate ImageOperationType = "update" // 更新镜像
)

func (ImageOperationLog) TableName() string {
	return "image_operation_logs"
}
