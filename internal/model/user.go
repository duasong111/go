package model

import (
	"gorm.io/gorm"
	"time"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB(db *gorm.DB) {
	DB = db
}

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

// 用户的 Bark Token 表（一个用户可能绑定多个设备/多个 Bark key，但通常一个够用）
type BarkToken struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`         // 关联 User
	Token     string         `gorm:"size:128;unique;not null" json:"token"` // Bark 的 key，通常 22 位或 64 位 device token
	Device    string         `gorm:"size:64" json:"device,omitempty"`       // 可选：设备备注，如 "iPhone 主设备"
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// 阈值配置表（支持多设备、多指标，未来可扩展）
type Threshold struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint           `gorm:"index;not null" json:"user_id"`
	DeviceID      string         `gorm:"size:64;index" json:"device_id"` // 设备标识，必填或空表示全局
	TempMax       *float64       `json:"temp_max,omitempty"`             // 温度上限，null 表示不监控
	TempMin       *float64       `json:"temp_min,omitempty"`
	HumidityMax   *float64       `json:"humidity_max,omitempty"`
	HumidityMin   *float64       `json:"humidity_min,omitempty"`
	AlertInterval int            `gorm:"default:100" json:"alert_interval"` // 报警间隔秒数，默认5分钟防抖
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// 距离监测阈值表（独立表，用于近距离报警）
type DistanceThreshold struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint           `gorm:"index;not null" json:"user_id"`
	DeviceID      string         `gorm:"size:64;index" json:"device_id"` // 设备标识
	DistanceMin   *float64       `json:"distance_min,omitempty"`         // 距离下限（小于此值报警），null 表示不监控
	AlertInterval int            `gorm:"default:300" json:"alert_interval"` // 报警间隔秒数，默认5分钟
	IsActive      bool           `gorm:"default:true" json:"is_active"` // 是否启用
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
