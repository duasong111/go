package model

import (
	"gorm.io/gorm"
	"time"
)

// User 用户模型（对应数据库表 users）
type User struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string         `gorm:"unique;not null" json:"username"` // 用户名唯一
	Password  string         `gorm:"not null" json:"-"`               // 存储加密后的密码
	Email     string         `gorm:"unique" json:"email"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
