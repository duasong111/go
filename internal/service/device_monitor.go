package service

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/redis"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
	//"github.com/redis/go-redis/v9"
)

// 设备离线监测服务

// UpdateDeviceLastActive 更新设备最后活跃时间
func UpdateDeviceLastActive(userID uint, deviceID string) error {
	ctx := context.Background()
	// 使用 user_id + device_id 作为键，确保不同用户的设备不会混淆
	key := fmt.Sprintf("device:%d:%s:last_active", userID, deviceID)

	// 检查设备之前的状态
	lastActiveStr, err := redis.Client.Get(ctx, key).Result()

	if err == nil {
		// 解析时间戳
		var lastActive int64
		_, err = fmt.Sscanf(lastActiveStr, "%d", &lastActive)
		if err == nil {
			// 检查是否离线
			currentTime := time.Now().Unix()

			// 获取设备的离线配置
			var config model.DeviceOfflineConfig
			result := model.DB.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&config)
			if result.Error == nil && config.IsActive {
				// 如果超过离线阈值，则认为设备之前是离线状态
				if currentTime-lastActive > int64(config.OfflineThreshold) {
					// 发送上线提醒
					if config.EnableOnlineAlert {
						sendOnlineAlert(config)
					}
					// 重置离线通知次数计数器
					alertCountKey := fmt.Sprintf("alert:count:offline:%d:%s", config.UserID, config.DeviceID)
					redis.Client.Del(ctx, alertCountKey)
					log.Printf("设备 %s 上线，重置离线通知次数计数器", config.DeviceID)
				}
			}
		}
	}

	// 设置设备最后活跃时间（当前时间戳）
	err = redis.Client.Set(ctx, key, time.Now().Unix(), 24*time.Hour).Err()
	if err != nil {
		log.Printf("更新设备最后活跃时间失败: %v", err)
		return err
	}
	return nil
}

// sendOnlineAlert 发送设备上线提醒
func sendOnlineAlert(config model.DeviceOfflineConfig) {
	// 检查是否在报警冷却期
	ctx := context.Background()
	// 使用 user_id + device_id 作为键，确保不同用户的设备不会混淆
	cooldownKey := fmt.Sprintf("alert:cooldown:online:%d:%s", config.UserID, config.DeviceID)
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(config.AlertInterval)*time.Second).Result()
	if err != nil {
		log.Printf("Redis错误: %v", err)
		return
	}
	if !setOK {
		// 冷却中，不发送报警
		return
	}

	// 获取用户的 Bark Token
	var barkToken model.BarkToken
	if err := model.DB.Where("user_id = ? AND is_active = true", config.UserID).First(&barkToken).Error; err != nil {
		log.Printf("无可用Bark Token: user_id=%d", config.UserID)
		return
	}

	// 异步发送 Bark 通知
	go func() {
		title := fmt.Sprintf("设备[%s] 上线通知", config.DeviceID)
		message := "设备已重新上线，恢复正常通信"
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(message),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			log.Printf("Bark推送失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("设备上线通知推送成功: device_id=%s", config.DeviceID)
		}
	}()
}

// StartDeviceOfflineMonitor 启动设备离线监测
func StartDeviceOfflineMonitor() {
	go func() {
		for {
			checkDeviceOffline()
			time.Sleep(30 * time.Second) // 每30秒检查一次
		}
	}()
	log.Println("设备离线监测服务已启动")
}

// checkDeviceOffline 检查设备是否离线
func checkDeviceOffline() {
	// 获取所有设备离线监测配置
	var configs []model.DeviceOfflineConfig
	result := model.DB.Where("is_active = ?", true).Find(&configs)
	if result.Error != nil {
		log.Printf("获取设备离线配置失败: %v", result.Error)
		return
	}

	for _, config := range configs {
		// 检查设备最后活跃时间
		ctx := context.Background()
		// 使用 user_id + device_id 作为键，确保不同用户的设备不会混淆
		key := fmt.Sprintf("device:%d:%s:last_active", config.UserID, config.DeviceID)
		lastActiveStr, err := redis.Client.Get(ctx, key).Result()
		if err != nil {
			// 没有记录，可能是设备从未上线
			continue
		}

		// 解析时间戳
		var lastActive int64
		_, err = fmt.Sscanf(lastActiveStr, "%d", &lastActive)
		if err != nil {
			log.Printf("解析时间戳失败: %v", err)
			continue
		}

		// 检查是否离线
		currentTime := time.Now().Unix()
		if currentTime-lastActive > int64(config.OfflineThreshold) {
			// 设备离线，发送报警
			sendOfflineAlert(config)
		}
	}
}

// sendOfflineAlert 发送设备离线报警
func sendOfflineAlert(config model.DeviceOfflineConfig) {
	// 检查是否在报警冷却期
	ctx := context.Background()
	// 使用 user_id + device_id 作为键，确保不同用户的设备不会混淆
	cooldownKey := fmt.Sprintf("alert:cooldown:offline:%d:%s", config.UserID, config.DeviceID)
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(config.AlertInterval)*time.Second).Result()
	if err != nil {
		log.Printf("Redis错误: %v", err)
		return
	}
	if !setOK {
		// 冷却中，不发送报警
		return
	}

	// 检查通知次数
	// 使用 user_id + device_id 作为键，确保不同用户的设备不会混淆
	alertCountKey := fmt.Sprintf("alert:count:offline:%d:%s", config.UserID, config.DeviceID)
	count, err := redis.Client.Get(ctx, alertCountKey).Int()
	if err != nil && err.Error() != "redis: nil" {
		log.Printf("获取通知次数失败: %v", err)
		return
	}

	// 如果达到最大通知次数，不再发送
	if count >= config.MaxAlertCount {
		//log.Printf("设备 %s 已达到最大通知次数 (%d次)", config.DeviceID, config.MaxAlertCount)
		return
	}

	// 增加通知次数
	redis.Client.Incr(ctx, alertCountKey)
	// 设置过期时间（24小时）
	redis.Client.Expire(ctx, alertCountKey, 24*time.Hour)

	// 获取用户的 Bark Token
	var barkToken model.BarkToken
	if err := model.DB.Where("user_id = ? AND is_active = true", config.UserID).First(&barkToken).Error; err != nil {
		log.Printf("无可用Bark Token: user_id=%d", config.UserID)
		return
	}

	// 异步发送 Bark 通知
	go func() {
		title := fmt.Sprintf("设备[%s] 离线警告", config.DeviceID)
		message := "设备长时间未上报数据，可能已离线"
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(message),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			log.Printf("Bark推送失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("设备离线报警推送成功: device_id=%s", config.DeviceID)
		}
	}()
}
