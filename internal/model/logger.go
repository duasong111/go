package model

import (
	"gorm.io/gorm"
	"time"
)

// Log 日志表结构

type Log struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Level     string         `gorm:"type:varchar(20);not null" json:"level"`
	Message   string         `gorm:"type:text;not null" json:"message"`
	Detail    string         `gorm:"type:jsonb" json:"detail"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
