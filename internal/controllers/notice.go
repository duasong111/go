package controllers

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/elasticsearch"
	"awesomeProject/internal/model"
	"awesomeProject/internal/redis"
	"awesomeProject/internal/service"
	"awesomeProject/internal/service/rabbitmq"
	"awesomeProject/pkg"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func AcceptBarkToken(c *gin.Context) {
	var req pkg.BarkTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 获取完整用户信息
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户不存在",
		})
		return
	}

	// 保存或更新 Token（一个用户通常覆盖旧的）
	tokenRecord := model.BarkToken{
		UserID:   user.ID,
		Token:    req.Token,
		Device:   req.Device,
		IsActive: true,
	}

	result := model.DB.Where("user_id = ?", user.ID).
		Assign(&tokenRecord).
		FirstOrCreate(&model.BarkToken{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "保存 Bark Token 失败: " + result.Error.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Bark Token 已保存",
	})
}

// 2. 接收并保存阈值配置
func AcceptThreshold(c *gin.Context) {
	var req pkg.ThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 获取完整用户信息
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "用户不存在",
		})
		return
	}

	// 必须先绑定 Token
	var tokenCount int64
	model.DB.Model(&model.BarkToken{}).
		Where("user_id = ? AND is_active = true", user.ID).
		Count(&tokenCount)

	if tokenCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请先绑定有效的 Bark Token",
		})
		return
	}

	// 默认报警间隔 5 分钟
	if req.AlertSeconds <= 0 {
		req.AlertSeconds = 300
	}

	var threshold model.Threshold
	result := model.DB.Where("user_id = ? AND device_id = ?", user.ID, req.DeviceID).
		First(&threshold)

	if result.Error != nil {
		// 创建新记录
		threshold = model.Threshold{
			UserID:        user.ID,
			DeviceID:      req.DeviceID,
			TempMax:       req.TempMax,
			TempMin:       req.TempMin,
			HumidityMax:   req.HumidityMax,
			HumidityMin:   req.HumidityMin,
			AlertInterval: req.AlertSeconds,
			IsActive:      req.IsActive,
		}
		if err := model.DB.Create(&threshold).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "保存阈值失败: " + err.Error(),
			})
			return
		}
	} else {
		// 更新现有记录
		updates := map[string]interface{}{
			"temp_max":       req.TempMax,
			"temp_min":       req.TempMin,
			"humidity_max":   req.HumidityMax,
			"humidity_min":   req.HumidityMin,
			"alert_interval": req.AlertSeconds,
			"is_active":      req.IsActive,
		}
		if err := model.DB.Model(&threshold).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新阈值失败: " + err.Error(),
			})
			return
		}
	}

	// 清除 Redis 缓存
	ctx := context.Background()
	cacheKey := fmt.Sprintf("threshold:%d:%s", userID, req.DeviceID)
	redis.Client.Del(ctx, cacheKey)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "阈值配置已保存",
	})
}

// 实践中给设备更新唯一的device_id 并在出厂编号中进行注明

func BarkAlert(c *gin.Context) {
	var req pkg.DeviceAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		//log.Printf("[BarkAlert] 参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}
	//log.Printf("[BarkAlert] 收到设备上报数据: device_id=%s, temperature=%.2f, humidity=%.2f", req.DeviceID, req.Temperature, req.Humidity)

	var threshold model.Threshold
	if err := model.DB.Where("device_id = ?", req.DeviceID).First(&threshold).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "未配置阈值，跳过报警",
		})
		return
	}

	// 检查是否启用
	if !threshold.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "报警已禁用，跳过报警",
		})
		return
	}

	// 收集报警信息
	var alertItems []string
	if threshold.TempMax != nil && req.Temperature > *threshold.TempMax {
		alertItems = append(alertItems, fmt.Sprintf("温度过高: %.1f > %.1f", req.Temperature, *threshold.TempMax))
	}
	if threshold.TempMin != nil && req.Temperature < *threshold.TempMin {
		alertItems = append(alertItems, fmt.Sprintf("温度过低: %.1f < %.1f", req.Temperature, *threshold.TempMin))
	}
	if threshold.HumidityMax != nil && req.Humidity > *threshold.HumidityMax {
		alertItems = append(alertItems, fmt.Sprintf("湿度过高: %.1f > %.1f", req.Humidity, *threshold.HumidityMax))
	}
	if threshold.HumidityMin != nil && req.Humidity < *threshold.HumidityMin {
		alertItems = append(alertItems, fmt.Sprintf("湿度过低: %.1f < %.1f", req.Humidity, *threshold.HumidityMin))
	}

	if len(alertItems) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "指标正常",
		})
		return
	}

	alertMessage := strings.Join(alertItems, "；")

	cooldownKey := fmt.Sprintf("alert:cooldown:%d:%s:%s", threshold.UserID, req.DeviceID, alertMessage)
	ctx := context.Background()
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(threshold.AlertInterval)*time.Second).Result()
	if err != nil {
		log.Printf("Redis SetNX 错误: %v", err)
		// Redis 失败时仍允许推送（避免漏报）
	} else if !setOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "报警冷却中",
		})
		return
	}

	// 获取用户 Bark Token（取第一个活跃的）
	var barkToken model.BarkToken
	if err := model.DB.Where("user_id = ? AND is_active = true", threshold.UserID).
		First(&barkToken).Error; err != nil {
		//log.Printf("未找到活跃 Bark Token: user_id=%d, err=%v", threshold.UserID, err)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "无可用 Bark Token",
		})
		return
	}

	// 异步推送 Bark
	go func() {
		title := fmt.Sprintf("设备[%s] 预警", req.DeviceID)
		body := alertMessage
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(body),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			log.Printf("Bark 请求失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Bark 返回非 200: %d", resp.StatusCode)
		} else {
			log.Printf("Bark 推送成功: device=%s, alert=%s", req.DeviceID, alertMessage)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "报警已触发并推送",
	})
}

// 4. 设置距离监测阈值（独立接口）
func AcceptDistanceThreshold(c *gin.Context) {
	var req pkg.DistanceThresholdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 必须先绑定 Token
	var tokenCount int64
	model.DB.Model(&model.BarkToken{}).
		Where("user_id = ? AND is_active = true", userID).
		Count(&tokenCount)

	if tokenCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请先绑定有效的 Bark Token",
		})
		return
	}

	// 默认报警间隔 5 分钟
	if req.AlertSeconds <= 0 {
		req.AlertSeconds = 300
	}

	var threshold model.DistanceThreshold
	result := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).
		First(&threshold)

	if result.Error != nil {
		// 创建新记录
		threshold = model.DistanceThreshold{
			UserID:        userID,
			DeviceID:      req.DeviceID,
			DistanceMin:   req.DistanceMin,
			AlertInterval: req.AlertSeconds,
			IsActive:      req.IsActive,
		}
		if err := model.DB.Create(&threshold).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "保存距离阈值失败: " + err.Error(),
			})
			return
		}
	} else {
		// 更新现有记录
		updates := map[string]interface{}{
			"distance_min":   req.DistanceMin,
			"alert_interval": req.AlertSeconds,
			"is_active":      req.IsActive,
		}
		if err := model.DB.Model(&threshold).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新距离阈值失败: " + err.Error(),
			})
			return
		}
	}
	// 清除缓存
	ctx := context.Background()
	cacheKey := fmt.Sprintf("distance_threshold:%d:%s", userID, req.DeviceID)
	redis.Client.Del(ctx, cacheKey)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "距离监测阈值已保存",
	})
}

func DistanceAlert(c *gin.Context) {
	var req pkg.DistanceAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}
	//log.Printf("[DistanceAlert] 收到距离上报: device_id=%s, distance=%.2f", req.DeviceID, req.Distance)

	// 查找该设备的距离阈值配置
	var threshold model.DistanceThreshold
	if err := model.DB.Where("device_id = ?", req.DeviceID).First(&threshold).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "未配置距离阈值，跳过报警",
		})
		return
	}

	// 检查是否启用
	if !threshold.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "距离报警已禁用，跳过报警",
		})
		return
	}

	if threshold.DistanceMin == nil || req.Distance >= *threshold.DistanceMin {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "距离正常",
		})
		return
	}

	alertMessage := fmt.Sprintf("距离过近: %.1f < %.1f", req.Distance, *threshold.DistanceMin)
	//log.Printf("[DistanceAlert] 触发报警: %s", alertMessage)

	cooldownKey := fmt.Sprintf("distance_alert:cooldown:%d:%s:%s", threshold.UserID, req.DeviceID, alertMessage)
	ctx := context.Background()
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(threshold.AlertInterval)*time.Second).Result()
	if err != nil {
		log.Printf("[DistanceAlert] Redis SetNX 错误: %v", err)
	} else if !setOK {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "报警冷却中",
		})
		return
	}

	var barkToken model.BarkToken
	if err := model.DB.Where("user_id = ? AND is_active = true", threshold.UserID).
		First(&barkToken).Error; err != nil {
		log.Printf("[DistanceAlert] 未找到活跃 Bark Token: user_id=%d, err=%v", threshold.UserID, err)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "无可用 Bark Token",
		})
		return
	}

	go func() {
		title := fmt.Sprintf("设备[%s] 距离预警", req.DeviceID)
		body := alertMessage
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(body),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			log.Printf("[DistanceAlert] Bark 请求失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("[DistanceAlert] Bark 返回非 200: %d", resp.StatusCode)
		} else {
			log.Printf("[DistanceAlert] Bark 推送成功: device=%s, alert=%s", req.DeviceID, alertMessage)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "距离报警已触发并推送",
	})
}

func SensorData(c *gin.Context) {
	var req pkg.SensorDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[SensorData] 参数绑定失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 验证设备密钥
	var factoryConfig model.DeviceFactoryConfig
	if err := model.DB.Where("device_id = ? AND secret_key = ?", req.DeviceID, req.SecretKey).First(&factoryConfig).Error; err != nil {
		// 尝试使用设备别名查找
		var userDevice model.UserDevice
		if err := model.DB.Where("device_name = ?", req.DeviceID).First(&userDevice).Error; err == nil {
			if err := model.DB.Where("device_id = ? AND secret_key = ?", userDevice.DeviceID, req.SecretKey).First(&factoryConfig).Error; err == nil {
				// 使用实际的设备ID
				req.DeviceID = userDevice.DeviceID
			} else {
				log.Printf("[SensorData] 设备密钥验证失败: device_id=%s", req.DeviceID)
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":    401,
					"message": "设备密钥错误",
				})
				return
			}
		} else {
			log.Printf("[SensorData] 设备密钥验证失败: device_id=%s", req.DeviceID)
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "设备密钥错误",
			})
			return
		}
	}

	// 检查设备是否已激活
	if !factoryConfig.IsActivated {
		log.Printf("[SensorData] 设备未激活: device_id=%s", req.DeviceID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "设备未激活",
		})
		return
	}

	// 查找设备对应的用户
	var userDevice model.UserDevice
	if err := model.DB.Where("device_id = ?", req.DeviceID).First(&userDevice).Error; err != nil {
		log.Printf("[SensorData] 设备未绑定到用户: device_id=%s", req.DeviceID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "设备未绑定到用户",
		})
		return
	}

	// 获取设备对应的用户ID
	userID := userDevice.UserID

	// 更新设备最后活跃时间
	go func() {
		err := service.UpdateDeviceLastActive(userID, req.DeviceID)
		if err != nil {
			log.Printf("[SensorData] 更新设备最后活跃时间失败: %v", err)
		}
	}()

	// 发送数据到 RabbitMQ
	go func() {
		err := rabbitmq.Publish(
			config.AppConfig.RabbitMQ.Exchange,
			config.AppConfig.RabbitMQ.RoutingKey,
			req,
		)
		if err != nil {
			log.Printf("[SensorData] 发送到 RabbitMQ 失败: %v", err)
		}
	}()

	// 索引到 Elasticsearch
	go func() {
		err := elasticsearch.IndexSensorData(req.DeviceID, req.Temperature, req.Humidity, req.Distance)
		if err != nil {
			log.Printf("[SensorData] 索引到 Elasticsearch 失败: %v", err)
		}
	}()

	// 创建通道来接收两个处理结果
	tempHumidityDone := make(chan bool, 1)
	distanceDone := make(chan bool, 1)

	// 异步处理温湿度报警
	go func() {
		ProcessTempHumidityAlert(userID, req.DeviceID, req.Temperature, req.Humidity)
		tempHumidityDone <- true
	}()

	// 异步处理距离报警
	go func() {
		ProcessDistanceAlert(userID, req.DeviceID, req.Distance)
		distanceDone <- true
	}()

	// 等待两个处理完成（带超时）
	select {
	case <-tempHumidityDone:
	case <-time.After(5 * time.Second):
	}

	select {
	case <-distanceDone:
	case <-time.After(5 * time.Second):
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "传感器数据已接收并处理",
	})
}

// 处理温湿度报警（内部函数）
func ProcessTempHumidityAlert(userID uint, deviceID string, temperature, humidity float64) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("threshold:%d:%s", userID, deviceID)

	// 1. 先从 Redis 缓存获取阈值配置
	var threshold model.Threshold
	cached, err := redis.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，反序列化
		if err := json.Unmarshal([]byte(cached), &threshold); err == nil {
			// 检查是否启用
			if !threshold.IsActive {
				return
			}
		} else {
			// 反序列化失败，从数据库查询
			if err := findThreshold(userID, deviceID, &threshold); err != nil {
				// 数据库中没有记录，静默返回
				return
			}
			// 写入缓存
			if data, err := json.Marshal(threshold); err == nil {
				redis.Client.Set(ctx, cacheKey, data, 1*time.Hour)
			}
		}
	} else {
		// 缓存未命中，从数据库查询
		if err := findThreshold(userID, deviceID, &threshold); err != nil {
			// 数据库中没有记录，静默返回
			return
		}
		// 检查是否启用
		if !threshold.IsActive {
			return
		}
		// 写入缓存
		if data, err := json.Marshal(threshold); err == nil {
			redis.Client.Set(ctx, cacheKey, data, 1*time.Hour)
		}
	}

	// 收集报警信息
	var alertItems []string
	if threshold.TempMax != nil && temperature > *threshold.TempMax {
		alertItems = append(alertItems, fmt.Sprintf("温度过高: %.1f > %.1f", temperature, *threshold.TempMax))
	}
	if threshold.TempMin != nil && temperature < *threshold.TempMin {
		alertItems = append(alertItems, fmt.Sprintf("温度过低: %.1f < %.1f", temperature, *threshold.TempMin))
	}
	if threshold.HumidityMax != nil && humidity > *threshold.HumidityMax {
		alertItems = append(alertItems, fmt.Sprintf("湿度过高: %.1f > %.1f", humidity, *threshold.HumidityMax))
	}
	if threshold.HumidityMin != nil && humidity < *threshold.HumidityMin {
		alertItems = append(alertItems, fmt.Sprintf("湿度过低: %.1f < %.1f", humidity, *threshold.HumidityMin))
	}

	if len(alertItems) == 0 {
		//log.Printf("[processTempHumidityAlert] 指标正常: device_id=%s", deviceID)
		return
	}

	alertMessage := strings.Join(alertItems, "；")
	//log.Printf("[processTempHumidityAlert] 触发报警: %s", alertMessage)

	// 防抖
	cooldownKey := fmt.Sprintf("alert:cooldown:%d:%s:%s", userID, deviceID, alertMessage)
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(threshold.AlertInterval)*time.Second).Result()
	if err != nil {
		//log.Printf("[processTempHumidityAlert] Redis错误: %v", err)
		return
	}
	if !setOK {
		//log.Printf("[processTempHumidityAlert] 冷却中: device_id=%s", deviceID)
		return
	}

	// 获取Bark Token并推送
	barkTokenCacheKey := fmt.Sprintf("bark_token:%d", userID)
	var barkToken model.BarkToken
	cachedToken, err := redis.Client.Get(ctx, barkTokenCacheKey).Result()
	if err == nil {
		// 缓存命中，反序列化
		if err := json.Unmarshal([]byte(cachedToken), &barkToken); err != nil {
			// 反序列化失败，从数据库查询
			if err := model.DB.Where("user_id = ? AND is_active = true", userID).First(&barkToken).Error; err != nil {
				return
			}
			// 写入缓存
			if data, err := json.Marshal(barkToken); err == nil {
				redis.Client.Set(ctx, barkTokenCacheKey, data, 1*time.Hour)
			}
		}
	} else {
		// 缓存未命中，从数据库查询
		if err := model.DB.Where("user_id = ? AND is_active = true", userID).First(&barkToken).Error; err != nil {
			return
		}
		// 写入缓存
		if data, err := json.Marshal(barkToken); err == nil {
			redis.Client.Set(ctx, barkTokenCacheKey, data, 1*time.Hour)
		}
	}

	// 记录提醒
	service.CreateNotification(userID, deviceID, service.NotificationTypeAlert, alertMessage)

	// 异步推送
	go func() {
		title := fmt.Sprintf("设备[%s] 温湿度预警", deviceID)
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(alertMessage),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			//log.Printf("[processTempHumidityAlert] Bark推送失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			//log.Printf("[processTempHumidityAlert] Bark推送成功: device_id=%s", deviceID)
		}

		// 根据配置的报警行为触发设备
		triggerDeviceByAlertAction(deviceID, threshold.AlertAction, threshold.LedColor)
	}()
}

// 查找用户设备映射
func findUserDevice(userID uint, deviceIdentifier string) (model.UserDevice, error) {
	var userDevice model.UserDevice

	// 1. 先尝试使用device_id查找
	if err := model.DB.Where("user_id = ? AND device_id = ?", userID, deviceIdentifier).First(&userDevice).Error; err == nil {
		return userDevice, nil
	}

	// 2. 如果找不到，尝试使用设备别名查找
	if err := model.DB.Where("user_id = ? AND device_name = ?", userID, deviceIdentifier).First(&userDevice).Error; err == nil {
		return userDevice, nil
	}

	// 3. 如果还找不到，返回错误
	return userDevice, errors.New("device not found")
}

// 查找温湿度阈值配置
func findThreshold(userID uint, deviceID string, threshold *model.Threshold) error {
	// 尝试获取用户设备映射
	userDevice, err := findUserDevice(userID, deviceID)
	if err == nil {
		// 使用实际的device_id查找
		if err := model.DB.Where("user_id = ? AND device_id = ?", userID, userDevice.DeviceID).First(threshold).Error; err == nil {
			return nil
		}
	}

	// 如果找不到，尝试获取用户的第一个设备配置
	return model.DB.Where("user_id = ?", userID).First(threshold).Error
}

// 查找距离阈值配置
func findDistanceThreshold(userID uint, deviceID string, threshold *model.DistanceThreshold) error {
	// 尝试获取用户设备映射
	userDevice, err := findUserDevice(userID, deviceID)
	if err == nil {
		// 使用实际的device_id查找
		if err := model.DB.Where("user_id = ? AND device_id = ?", userID, userDevice.DeviceID).First(threshold).Error; err == nil {
			return nil
		}
	}

	// 如果找不到，尝试获取用户的第一个设备配置
	return model.DB.Where("user_id = ?", userID).First(threshold).Error
}

// 处理距离报警（内部函数）
func ProcessDistanceAlert(userID uint, deviceID string, distance float64) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("distance_threshold:%d:%s", userID, deviceID)

	// 1. 先从 Redis 缓存获取距离阈值配置
	var threshold model.DistanceThreshold
	cached, err := redis.Client.Get(ctx, cacheKey).Result()
	if err == nil {
		// 缓存命中，反序列化
		if err := json.Unmarshal([]byte(cached), &threshold); err == nil {
			// 检查是否启用
			if !threshold.IsActive {
				return
			}
		} else {
			// 反序列化失败，从数据库查询
			if err := findDistanceThreshold(userID, deviceID, &threshold); err != nil {
				// 数据库中没有记录，静默返回
				return
			}
			// 写入缓存
			if data, err := json.Marshal(threshold); err == nil {
				redis.Client.Set(ctx, cacheKey, data, 1*time.Hour)
			}
		}
	} else {
		// 缓存未命中，从数据库查询
		if err := findDistanceThreshold(userID, deviceID, &threshold); err != nil {
			// 数据库中没有记录，静默返回
			return
		}
		// 检查是否启用
		if !threshold.IsActive {
			return
		}
		// 写入缓存
		if data, err := json.Marshal(threshold); err == nil {
			redis.Client.Set(ctx, cacheKey, data, 1*time.Hour)
		}
	}

	// 判断是否触发报警
	if threshold.DistanceMin == nil || distance >= *threshold.DistanceMin {
		//log.Printf("[processDistanceAlert] 距离正常: device_id=%s, distance=%.2f", deviceID, distance)
		return
	}

	alertMessage := fmt.Sprintf("距离过近: %.1f < %.1f", distance, *threshold.DistanceMin)
	//log.Printf("[processDistanceAlert] 触发报警: %s", alertMessage)

	// 防抖
	cooldownKey := fmt.Sprintf("distance_alert:cooldown:%d:%s:%s", userID, deviceID, alertMessage)
	setOK, err := redis.Client.SetNX(ctx, cooldownKey, "1", time.Duration(threshold.AlertInterval)*time.Second).Result()
	if err != nil {
		//log.Printf("[processDistanceAlert] Redis错误: %v", err)
		return
	}
	if !setOK {
		//log.Printf("[processDistanceAlert] 冷却中: device_id=%s", deviceID)
		return
	}

	// 获取Bark Token并推送
	barkTokenCacheKey := fmt.Sprintf("bark_token:%d", userID)
	var barkToken model.BarkToken
	cachedToken, err := redis.Client.Get(ctx, barkTokenCacheKey).Result()
	if err == nil {
		// 缓存命中，反序列化
		if err := json.Unmarshal([]byte(cachedToken), &barkToken); err != nil {
			// 反序列化失败，从数据库查询
			if err := model.DB.Where("user_id = ? AND is_active = true", userID).First(&barkToken).Error; err != nil {
				return
			}
			// 写入缓存
			if data, err := json.Marshal(barkToken); err == nil {
				redis.Client.Set(ctx, barkTokenCacheKey, data, 1*time.Hour)
			}
		}
	} else {
		// 缓存未命中，从数据库查询
		if err := model.DB.Where("user_id = ? AND is_active = true", userID).First(&barkToken).Error; err != nil {
			return
		}
		// 写入缓存
		if data, err := json.Marshal(barkToken); err == nil {
			redis.Client.Set(ctx, barkTokenCacheKey, data, 1*time.Hour)
		}
	}

	// 记录提醒
	service.CreateNotification(userID, deviceID, service.NotificationTypeAlert, alertMessage)

	// 异步推送
	go func() {
		title := fmt.Sprintf("设备[%s] 距离预警", deviceID)
		barkURL := fmt.Sprintf("https://api.day.app/%s/%s/%s?sound=alarm",
			barkToken.Token,
			url.QueryEscape(title),
			url.QueryEscape(alertMessage),
		)

		resp, err := http.Get(barkURL)
		if err != nil {
			//log.Printf("[processDistanceAlert] Bark推送失败: %v", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			//log.Printf("[processDistanceAlert] Bark推送成功: device_id=%s", deviceID)
		}

		// 根据配置的报警行为触发设备
		triggerDeviceByAlertAction(deviceID, threshold.AlertAction, threshold.LedColor)
	}()
}

// 7. 升级版阈值管理接口（查询、修改、配置报警行为）
func ManageThreshold(c *gin.Context) {
	var req pkg.ThresholdManageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误：" + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 查询现有阈值
	var threshold model.Threshold
	result := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).First(&threshold)

	if result.Error != nil {
		// 创建新记录
		if req.AlertSeconds <= 0 {
			req.AlertSeconds = 300
		}
		if req.AlertAction == "" {
			req.AlertAction = "buzzer"
		}
		if req.LedColor == "" {
			req.LedColor = "red"
		}

		threshold = model.Threshold{
			UserID:        userID,
			DeviceID:      req.DeviceID,
			TempMax:       req.TempMax,
			TempMin:       req.TempMin,
			HumidityMax:   req.HumidityMax,
			HumidityMin:   req.HumidityMin,
			AlertInterval: req.AlertSeconds,
			IsActive:      req.IsActive,
			AlertAction:   req.AlertAction,
			LedColor:      req.LedColor,
		}
		if err := model.DB.Create(&threshold).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "创建阈值失败：" + err.Error(),
			})
			return
		}
	} else {
		// 更新现有记录
		updates := map[string]interface{}{}
		if req.TempMax != nil {
			updates["temp_max"] = req.TempMax
		}
		if req.TempMin != nil {
			updates["temp_min"] = req.TempMin
		}
		if req.HumidityMax != nil {
			updates["humidity_max"] = req.HumidityMax
		}
		if req.HumidityMin != nil {
			updates["humidity_min"] = req.HumidityMin
		}
		if req.AlertSeconds > 0 {
			updates["alert_interval"] = req.AlertSeconds
		}
		updates["is_active"] = req.IsActive
		if req.AlertAction != "" {
			updates["alert_action"] = req.AlertAction
		}
		if req.LedColor != "" {
			updates["led_color"] = req.LedColor
		}

		if err := model.DB.Model(&threshold).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新阈值失败：" + err.Error(),
			})
			return
		}
	}

	// 清除 Redis 缓存
	ctx := context.Background()
	cacheKey := fmt.Sprintf("threshold:%d:%s", userID, req.DeviceID)
	redis.Client.Del(ctx, cacheKey)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "阈值配置已保存",
		"data": gin.H{
			"device_id":      req.DeviceID,
			"temp_max":       threshold.TempMax,
			"temp_min":       threshold.TempMin,
			"humidity_max":   threshold.HumidityMax,
			"humidity_min":   threshold.HumidityMin,
			"alert_interval": threshold.AlertInterval,
			"is_active":      threshold.IsActive,
			"alert_action":   threshold.AlertAction,
			"led_color":      threshold.LedColor,
		},
	})
}

// 8. 升级版距离阈值管理接口（查询、修改、配置报警行为）
func ManageDistanceThreshold(c *gin.Context) {
	var req pkg.DistanceThresholdManageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误：" + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 查询现有阈值
	var threshold model.DistanceThreshold
	result := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).First(&threshold)

	if result.Error != nil {
		// 创建新记录
		if req.AlertSeconds <= 0 {
			req.AlertSeconds = 300
		}
		if req.AlertAction == "" {
			req.AlertAction = "buzzer"
		}
		if req.LedColor == "" {
			req.LedColor = "yellow"
		}

		threshold = model.DistanceThreshold{
			UserID:        userID,
			DeviceID:      req.DeviceID,
			DistanceMin:   req.DistanceMin,
			AlertInterval: req.AlertSeconds,
			IsActive:      req.IsActive,
			AlertAction:   req.AlertAction,
			LedColor:      req.LedColor,
		}
		if err := model.DB.Create(&threshold).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "创建距离阈值失败：" + err.Error(),
			})
			return
		}
	} else {
		// 更新现有记录
		updates := map[string]interface{}{}
		if req.DistanceMin != nil {
			updates["distance_min"] = req.DistanceMin
		}
		if req.AlertSeconds > 0 {
			updates["alert_interval"] = req.AlertSeconds
		}
		updates["is_active"] = req.IsActive
		if req.AlertAction != "" {
			updates["alert_action"] = req.AlertAction
		}
		if req.LedColor != "" {
			updates["led_color"] = req.LedColor
		}

		if err := model.DB.Model(&threshold).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "更新距离阈值失败：" + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "距离阈值配置已保存",
		"data": gin.H{
			"device_id":      req.DeviceID,
			"distance_min":   threshold.DistanceMin,
			"alert_interval": threshold.AlertInterval,
			"is_active":      threshold.IsActive,
			"alert_action":   threshold.AlertAction,
			"led_color":      threshold.LedColor,
		},
	})
}

// 9. 根据配置的报警行为触发设备
func triggerDeviceByAlertAction(deviceID string, alertAction string, ledColor string) {
	topic := "devices/" + deviceID + "/control"
	switch alertAction {
	case "buzzer":
		// 只触发蜂鸣器
		buzzerPayload := map[string]interface{}{
			"type":      "buzzer",
			"state":     "on",
			"frequency": 2000,
			"duration":  1000,
			"cycles":    3,
			"interval":  500,
		}
		service.Publish(topic, 0, false, buzzerPayload)

	case "led":
		// 只触发LED灯
		ledPayload := map[string]interface{}{
			"state":      "on",
			"color":      ledColor,
			"brightness": 100,
			"mode":       "blink",
			"interval":   500,
			"duration":   5000,
		}
		service.Publish(topic, 0, false, ledPayload)

	case "both":
		// 同时触发蜂鸣器和LED灯
		buzzerPayload := map[string]interface{}{
			"type":      "buzzer",
			"state":     "on",
			"frequency": 2000,
			"duration":  1000,
			"cycles":    3,
			"interval":  500,
		}
		service.Publish(topic, 0, false, buzzerPayload)

		ledPayload := map[string]interface{}{
			"state":      "on",
			"color":      ledColor,
			"brightness": 100,
			"mode":       "blink",
			"interval":   500,
			"duration":   5000,
		}
		service.Publish(topic, 0, false, ledPayload)
	}
}

// 统一设备控制接口
func ControlDevices(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
		Controls []struct {
			Type       string `json:"type" binding:"required,oneof=buzzer led"`
			State      string `json:"state" binding:"required,oneof=on off"`
			Frequency  *int   `json:"frequency,omitempty"`
			Duration   *int   `json:"duration,omitempty"`
			Cycles     *int   `json:"cycles,omitempty"`
			Interval   *int   `json:"interval,omitempty"`
			Color      string `json:"color,omitempty"`
			Brightness *int   `json:"brightness,omitempty"`
			Mode       string `json:"mode,omitempty"`
		} `json:"controls" binding:"required,dive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误：" + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 验证设备是否属于当前用户
	var deviceCount int64
	model.DB.Model(&model.Threshold{}).Where("user_id = ? AND device_id = ?", userID, req.DeviceID).Count(&deviceCount)
	if deviceCount == 0 {
		model.DB.Model(&model.DistanceThreshold{}).Where("user_id = ? AND device_id = ?", userID, req.DeviceID).Count(&deviceCount)
	}
	if deviceCount == 0 {
		model.DB.Model(&model.DeviceOfflineConfig{}).Where("user_id = ? AND device_id = ?", userID, req.DeviceID).Count(&deviceCount)
	}
	if deviceCount == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "无权控制该设备",
		})
		return
	}

	// 处理每个控制命令
	for _, control := range req.Controls {
		var payload map[string]interface{}

		switch control.Type {
		case "buzzer":
			payload = map[string]interface{}{
				"type":  "buzzer",
				"state": control.State,
			}
			if control.State == "on" {
				if control.Frequency != nil {
					payload["frequency"] = *control.Frequency
				} else {
					payload["frequency"] = 2000
				}
				if control.Duration != nil {
					payload["duration"] = *control.Duration
				} else {
					payload["duration"] = 500
				}
				if control.Cycles != nil {
					payload["cycles"] = *control.Cycles
				}
				if control.Interval != nil {
					payload["interval"] = *control.Interval
				}
			}

		case "led":
			payload = map[string]interface{}{
				"state": control.State,
			}
			if control.State == "on" {
				if control.Color != "" {
					payload["color"] = control.Color
				} else {
					payload["color"] = "white"
				}
				if control.Brightness != nil {
					payload["brightness"] = *control.Brightness
				} else {
					payload["brightness"] = 100
				}
				if control.Mode != "" {
					payload["mode"] = control.Mode
				}
				if control.Interval != nil {
					payload["interval"] = *control.Interval
				}
				if control.Duration != nil {
					payload["duration"] = *control.Duration
				}
			} else {
				payload["color"] = "black"
				payload["brightness"] = 0
			}
		}

		// 发送到 MQTT
		topic := "devices/" + req.DeviceID + "/control"
		err := service.Publish(topic, 0, false, payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "MQTT 命令发送失败：" + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          200,
		"message":       "设备控制命令已发送",
		"device_id":     req.DeviceID,
		"control_count": len(req.Controls),
	})
}

// 设备绑定接口
func BindDevice(c *gin.Context) {
	var req model.DeviceBindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 激活码限流检查
	ctx := context.Background()
	activationCodeKey := fmt.Sprintf("activation_code:limit:%s", req.ActivationCode)
	
	// 检查激活码使用次数
	count, err := redis.Client.Get(ctx, activationCodeKey).Int()
	if err == nil && count >= 3 { // 每个激活码最多使用3次
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "激活码已达到使用上限",
		})
		return
	}

	// 查找设备出厂配置
	var factoryConfig model.DeviceFactoryConfig
	if err := model.DB.Where("device_id = ? AND activation_code = ?", req.DeviceID, req.ActivationCode).First(&factoryConfig).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "设备ID或激活码错误",
		})
		return
	}

	// 检查设备是否已激活
	if factoryConfig.IsActivated {
		// 检查设备是否已绑定到其他用户
		var existingUserDevice model.UserDevice
		if err := model.DB.Where("device_id = ?", req.DeviceID).First(&existingUserDevice).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "该设备已被其他用户绑定",
			})
			return
		}
	}

	// 开始事务
	tx := model.DB.Begin()

	// 激活设备
	factoryConfig.IsActivated = true
	if err := tx.Save(&factoryConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "设备激活失败: " + err.Error(),
		})
		return
	}

	// 创建用户设备关联
	userDevice := model.UserDevice{
		UserID:     userID,
		DeviceID:   req.DeviceID,
		DeviceName: req.DeviceName,
	}

	if err := tx.Create(&userDevice).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "设备绑定失败: " + err.Error(),
		})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "事务提交失败: " + err.Error(),
		})
		return
	}

	// 增加激活码使用次数
	ctx = context.Background()
	activationCodeKey = fmt.Sprintf("activation_code:limit:%s", req.ActivationCode)
	redis.Client.Incr(ctx, activationCodeKey)
	// 设置过期时间为24小时
	redis.Client.Expire(ctx, activationCodeKey, 24*time.Hour)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "设备绑定成功",
		"data": gin.H{
			"device_id":     factoryConfig.DeviceID,
			"device_name":   userDevice.DeviceName,
			"secret_key":    factoryConfig.SecretKey,
			"is_activated":  factoryConfig.IsActivated,
			"bound_at":      userDevice.CreatedAt,
			"info": "请将此device_id和secret_key配置到设备中，设备上传数据时需要使用此device_id",
		},
	})
}

// 设备解绑接口
func UnbindDevice(c *gin.Context) {
	var req model.DeviceUnbindRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 查找用户设备关联
	var userDevice model.UserDevice
	if err := model.DB.Where("device_id = ? AND user_id = ?", req.DeviceID, userID).First(&userDevice).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "设备不存在或未绑定到当前用户",
		})
		return
	}

	// 开始事务
	tx := model.DB.Begin()

	// 解绑设备（删除用户设备关联）
	if err := tx.Delete(&userDevice).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "设备解绑失败: " + err.Error(),
		})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "事务提交失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "设备解绑成功",
		"data": gin.H{
			"device_id":  req.DeviceID,
			"unbound_at": time.Now(),
		},
	})
}

// 获取用户绑定的设备列表
func GetUserDevices(c *gin.Context) {
	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 查询用户绑定的设备
	var userDevices []model.UserDevice
	if err := model.DB.Where("user_id = ?", userID).Find(&userDevices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取设备列表失败: " + err.Error(),
		})
		return
	}

	// 构建设备信息列表
	type DeviceInfo struct {
		DeviceID   string    `json:"device_id"`
		DeviceName string    `json:"device_name"`
		BoundAt    time.Time `json:"bound_at"`
		Info       string    `json:"info"`
	}

	var deviceInfos []DeviceInfo
	for _, ud := range userDevices {
		deviceInfos = append(deviceInfos, DeviceInfo{
			DeviceID:   ud.DeviceID,
			DeviceName: ud.DeviceName,
			BoundAt:    ud.CreatedAt,
			Info:       "配置阈值时请使用此device_id",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取设备列表成功",
		"data":    deviceInfos,
	})
}

// 获取用户的提醒列表
func GetUserNotifications(c *gin.Context) {
	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 获取查询参数
	limit := 20 // 默认获取20条
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || l != 1 {
			limit = 20
		}
	}

	// 获取提醒列表
	notifications, err := service.GetUserNotifications(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取提醒列表失败: " + err.Error(),
		})
		return
	}

	// 获取未读提醒数量
	unreadCount, err := service.GetUnreadNotificationCount(userID)
	if err != nil {
		unreadCount = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code":          200,
		"message":       "获取提醒列表成功",
		"data":          notifications,
		"unread_count":  unreadCount,
		"total_count":   len(notifications),
	})
}

// 标记提醒为已读
func MarkNotificationAsRead(c *gin.Context) {
	// 获取当前用户ID
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 获取提醒ID
	notificationIDStr := c.Param("id")
	var notificationID uint
	if _, err := fmt.Sscanf(notificationIDStr, "%d", &notificationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的提醒ID",
		})
		return
	}

	// 标记为已读
	if err := service.MarkNotificationAsRead(notificationID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "标记提醒为已读失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "提醒已标记为已读",
		"data": gin.H{
			"notification_id": notificationID,
			"is_read":        true,
		},
	})
}
