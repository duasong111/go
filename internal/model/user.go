package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
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
	Email     *string        `gorm:"uniqueIndex" json:"email"`
	Address   *string        `gorm:"uniqueIndex" json:"address"`
	Phone     *string        `gorm:"uniqueIndex" json:"phone"`
	Education *string        `gorm:"uniqueIndex" json:"education"`
	Avatar    string         `gorm:"default:''" json:"avatar"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

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
	AlertInterval int            `gorm:"default:300" json:"alert_interval"`    // 报警间隔秒数，默认5分钟防抖
	IsActive      bool           `gorm:"default:true" json:"is_active"`        // 是否启用
	AlertAction   string         `gorm:"default:'buzzer'" json:"alert_action"` // 报警行为：buzzer(蜂鸣器), led(灯), both(两者)
	LedColor      string         `gorm:"default:'yellow'" json:"led_color"`    // LED灯颜色：red, blue, yellow, white
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// 距离阈值配置表
type DistanceThreshold struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        uint           `gorm:"index:idx_user_device;not null" json:"user_id"`
	DeviceID      string         `gorm:"size:64;index:idx_user_device" json:"device_id"`       // 设备标识
	DistanceMin   *float64       `json:"distance_min,omitempty"`               // 距离下限（小于此值报警），null 表示不监控
	AlertInterval int            `gorm:"default:300" json:"alert_interval"`    // 报警间隔秒数，默认5分钟
	IsActive      bool           `gorm:"default:true" json:"is_active"`        // 是否启用
	AlertAction   string         `gorm:"default:'buzzer'" json:"alert_action"` // 报警行为：buzzer(蜂鸣器), led(灯), both(两者)
	LedColor      string         `gorm:"default:'yellow'" json:"led_color"`    // LED灯颜色：red, blue, yellow, white
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// 设备离线监测配置表
type DeviceOfflineConfig struct {
	ID                uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            uint           `gorm:"index:idx_user_device;not null" json:"user_id"`
	DeviceID          string         `gorm:"size:64;index:idx_user_device" json:"device_id"`           // 设备标识
	IsActive          bool           `gorm:"default:false" json:"is_active"`           // 是否启用离线监测
	EnableOnlineAlert bool           `gorm:"default:false" json:"enable_online_alert"` // 是否启用上线提醒
	MaxAlertCount     int            `gorm:"default:3" json:"max_alert_count"`         // 最大通知次数，默认3次
	OfflineThreshold  int            `gorm:"default:300" json:"offline_threshold"`     // 离线判断阈值（秒），默认5分钟
	AlertInterval     int            `gorm:"default:300" json:"alert_interval"`        // 报警间隔秒数，默认5分钟
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// 设置唯一索引，确保每个用户的设备ID唯一
func (DeviceOfflineConfig) TableName() string {
	return "device_offline_configs"
}

// BeforeCreate 创建前的钩子，确保用户设备ID唯一
func (d *DeviceOfflineConfig) BeforeCreate(tx *gorm.DB) error {
	var count int64
	tx.Model(&DeviceOfflineConfig{}).Where("user_id = ? AND device_id = ?", d.UserID, d.DeviceID).Count(&count)
	if count > 0 {
		return errors.New("设备离线配置已存在")
	}
	return nil
}

// 设备工厂配置表
type DeviceFactoryConfig struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	DeviceID      string         `gorm:"size:64;unique;not null;index:idx_device_activation" json:"device_id"` // 设备唯一标识
	SecretKey     string         `gorm:"size:128;not null" json:"secret_key"`             // 设备密钥
	ActivationCode string        `gorm:"size:64;unique;not null;index:idx_device_activation" json:"activation_code"` // 激活码
	IsActivated   bool           `gorm:"default:false" json:"is_activated"`               // 是否已激活
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// 用户设备关联表
type UserDevice struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     uint           `gorm:"index:idx_user_device;not null" json:"user_id"`                   // 用户ID
	DeviceID   string         `gorm:"size:64;index:idx_user_device;not null" json:"device_id"`         // 设备ID
	DeviceName string         `gorm:"size:64;not null" json:"device_name"`             // 设备别名
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// 设置唯一索引，确保每个用户的设备ID唯一
func (UserDevice) TableName() string {
	return "user_devices"
}

// BeforeCreate 创建前的钩子，确保用户设备ID唯一
func (u *UserDevice) BeforeCreate(tx *gorm.DB) error {
	var count int64
	tx.Model(&UserDevice{}).Where("user_id = ? AND device_id = ?", u.UserID, u.DeviceID).Count(&count)
	if count > 0 {
		return errors.New("设备已绑定")
	}
	return nil
}

// 通知记录表
type NotificationLog struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`    // 关联 User
	DeviceID  string         `gorm:"size:64;index" json:"device_id"`  // 设备标识
	Type      string         `gorm:"size:16" json:"type"`            // 通知类型：alert(警报), offline(离线), online(上线), system(系统)
	Message   string         `gorm:"size:512" json:"message"`         // 通知内容
	IsRead    bool           `gorm:"default:false" json:"is_read"`     // 是否已读
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// 设备绑定请求
type DeviceBindRequest struct {
	DeviceID      string `json:"device_id" binding:"required"`      // 设备ID
	DeviceName    string `json:"device_name" binding:"required"`    // 设备别名
	ActivationCode string `json:"activation_code" binding:"required"` // 激活码
}

// 设备解绑请求
type DeviceUnbindRequest struct {
	DeviceID string `json:"device_id" binding:"required"` // 设备ID
}
