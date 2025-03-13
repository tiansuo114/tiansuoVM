package model

type AuditLog struct {
	ID        int64  `gorm:"primary_key;AUTO_INCREMENT"`
	UID       string `gorm:"column:uid;not null;index:uid;type:varchar(32)"`
	Username  string `gorm:"column:username;not null;index:username;type:varchar(32)"`
	Module    string `gorm:"column:module;not null;type:varchar(32)"` // 操作模块
	Method    string `gorm:"column:method;not null;type:varchar(32)"` // HTTP方法
	Duration  int64  `gorm:"column:duration;not null"`                // 请求耗时(ms)
	IP        string `gorm:"column:source_ip;not null;type:varchar(32)"`
	Status    int    `gorm:"column:status;not null"`                // HTTP状态码
	URI       string `gorm:"column:uri;not null;type:varchar(255)"` // 请求路径
	Operation string `gorm:"column:operation;type:varchar(32)"`     // 操作类型
	Detail    string `gorm:"column:detail;type:varchar(255)"`       // 操作详情
	CreatedAt int64  `gorm:"column:created_at;not null"`            // 创建时间
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
