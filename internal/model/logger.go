package model

import (
	"gorm.io/gorm"
	"time"
)

// Log 日志表结构
type Log struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Level     string         `gorm:"type:varchar(20);not null" json:"level"` // 日志级别 INFO / ERROR / WARN ...
	Message   string         `gorm:"type:text;not null" json:"message"`      // 日志信息
	Detail    string         `gorm:"type:jsonb" json:"detail"`               // 详情，存 JSON 结构更灵活
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`       // 创建时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                         // 软删除
}
