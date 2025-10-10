package model

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string         `gorm:"unique;not null" json:"username"`
	Password  string         `gorm:"not null" json:"-"`
	Email     string         `gorm:"unique" json:"email"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
