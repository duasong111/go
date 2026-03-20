package service

import (
	"awesomeProject/internal/model"
	"log"
)

// 提醒类型常量
const (
	NotificationTypeAlert   = "alert"   // 警报提醒
	NotificationTypeOffline = "offline" // 设备离线提醒
	NotificationTypeOnline  = "online"  // 设备上线提醒
	NotificationTypeSystem  = "system"  // 系统提醒
)

// CreateNotification 创建提醒记录
func CreateNotification(userID uint, deviceID string, notificationType string, message string) error {
	notification := model.NotificationLog{
		UserID:   userID,
		DeviceID: deviceID,
		Type:     notificationType,
		Message:  message,
		IsRead:   false,
	}

	if err := model.DB.Create(&notification).Error; err != nil {
		log.Printf("创建提醒记录失败: %v", err)
		return err
	}

	return nil
}

// GetUserNotifications 获取用户的提醒列表
func GetUserNotifications(userID uint, limit int) ([]model.NotificationLog, error) {
	var notifications []model.NotificationLog

	query := model.DB.Where("user_id = ?", userID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&notifications).Error; err != nil {
		log.Printf("获取用户提醒列表失败: %v", err)
		return nil, err
	}

	return notifications, nil
}

// MarkNotificationAsRead 标记提醒为已读
func MarkNotificationAsRead(notificationID uint, userID uint) error {
	result := model.DB.Model(&model.NotificationLog{}).Where("id = ? AND user_id = ?", notificationID, userID).Update("is_read", true)
	if result.Error != nil {
		log.Printf("标记提醒为已读失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		log.Printf("提醒不存在或不属于当前用户: notificationID=%d, userID=%d", notificationID, userID)
	}

	return nil
}

// GetUnreadNotificationCount 获取用户未读提醒数量
func GetUnreadNotificationCount(userID uint) (int64, error) {
	var count int64

	if err := model.DB.Model(&model.NotificationLog{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count).Error; err != nil {
		log.Printf("获取未读提醒数量失败: %v", err)
		return 0, err
	}

	return count, nil
}
