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
	Address   string         `gorm:"unique" json:"address"`
	Phone     string         `gorm:"unique" json:"phone"`
	Education string         `gorm:"unique" json:"education"`
	Avatar    string         `gorm:"default:''" json:"avatar"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
