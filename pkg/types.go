package pkg

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// 温度阈值请求
type ThresholdRequest struct {
	DeviceID       string   `json:"device_id" binding:"required"`
	TemperatureMin *float64 `json:"temperature_min,omitempty"`
	TemperatureMax *float64 `json:"temperature_max,omitempty"`
	HumidityMin    *float64 `json:"humidity_min,omitempty"`
	HumidityMax    *float64 `json:"humidity_max,omitempty"`
	AlertInterval  int      `json:"alert_interval" binding:"required"`
	IsActive       bool     `json:"is_active"`
	AlertAction    string   `json:"alert_action"`
	LedColor       string   `json:"led_color"`
}

type DistanceThresholdManageRequest struct {
	DeviceID     string   `json:"device_id" binding:"required"`
	DistanceMin  *float64 `json:"distance_min,omitempty"`
	AlertSeconds int      `json:"alert_seconds,omitempty"`
	IsActive     bool     `json:"is_active"`
	AlertAction  string   `json:"alert_action,omitempty"`
	LedColor     string   `json:"led_color,omitempty"`
}

// 距离阈值请求
type DistanceThresholdRequest struct {
	DeviceID      string   `json:"device_id" binding:"required"`
	DistanceMin   *float64 `json:"distance_min,omitempty"`
	AlertInterval int      `json:"alert_interval" binding:"required"`
	IsActive      bool     `json:"is_active"`
	AlertAction   string   `json:"alert_action"`
	LedColor      string   `json:"led_color"`
}

// 设备离线监测配置请求
type DeviceOfflineConfigRequest struct {
	DeviceID          string `json:"device_id" binding:"required"`
	OfflineThreshold  int    `json:"offline_threshold" binding:"required"`
	AlertInterval     int    `json:"alert_interval" binding:"required"`
	MaxAlertCount     int    `json:"max_alert_count" binding:"required"`
	IsActive          bool   `json:"is_active"`
	EnableOnlineAlert bool   `json:"enable_online_alert"`
}

// 统一传感器数据上报请求（包含温湿度和距离）
type SensorDataRequest struct {
	DeviceID    string      `json:"device_id" binding:"required"`
	SecretKey   string      `json:"secret_key" binding:"required"` // 设备密钥
	Temperature interface{} `json:"temperature"`
	Humidity    interface{} `json:"humidity"`
	Distance    interface{} `json:"distance"`
	Time        string      `json:"time,omitempty"` // 可选：设备时间
}

// parseValue 解析值，支持数字和字符串类型
func (r *SensorDataRequest) parseValue(value interface{}) *float64 {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		if v == "" || v == "--" {
			return nil
		}
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil
		}
		return &val
	case float64:
		return &v
	case float32:
		val := float64(v)
		return &val
	case int:
		val := float64(v)
		return &val
	case int64:
		val := float64(v)
		return &val
	case int32:
		val := float64(v)
		return &val
	default:
		return nil
	}
}

// GetTemperature 获取温度值，处理"--"等特殊值
func (r *SensorDataRequest) GetTemperature() *float64 {
	return r.parseValue(r.Temperature)
}

// GetHumidity 获取湿度值，处理"--"等特殊值
func (r *SensorDataRequest) GetHumidity() *float64 {
	return r.parseValue(r.Humidity)
}

// GetDistance 获取距离值，处理"--"等特殊值
func (r *SensorDataRequest) GetDistance() *float64 {
	return r.parseValue(r.Distance)
}

// IsSensorDataValid 检查传感器数据是否有效
func (r *SensorDataRequest) IsSensorDataValid() bool {
	return r.GetTemperature() != nil || r.GetHumidity() != nil || r.GetDistance() != nil
}

// GetSensorStatus 获取传感器状态
func (r *SensorDataRequest) GetSensorStatus() map[string]string {
	status := make(map[string]string)

	if r.GetTemperature() == nil {
		status["temperature"] = "error"
	} else {
		status["temperature"] = "normal"
	}

	if r.GetHumidity() == nil {
		status["humidity"] = "error"
	} else {
		status["humidity"] = "normal"
	}

	if r.GetDistance() == nil {
		status["distance"] = "error"
	} else {
		status["distance"] = "normal"
	}

	return status
}

// ToJSON 转换为JSON字符串
func (r *SensorDataRequest) ToJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// 响应结构
// APIResponse API响应结构体
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 错误响应
// func ErrorResponse(c interface{}, code int, message string) interface{} {
// 	// 这里需要根据具体的上下文类型实现
// 	fmt.Printf("Error: %d - %s\n", code, message)
// }

// 成功响应
func SuccessResponse(c interface{}, data interface{}, message string) {
	// 这里需要根据具体的上下文类型实现
	fmt.Printf("Success: %s\n", message)
}
