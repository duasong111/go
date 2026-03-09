package pkg

type BarkTokenRequest struct {
	Token  string `json:"token" binding:"required"` // Bark 的 key，例如 "xxxxxx" (api.day.app/xxxxxx/)
	Device string `json:"device,omitempty"`         // 可选：用户备注这个 token 属于哪个手机
}

type ThresholdRequest struct {
	DeviceID     string   `json:"device_id" binding:"required"` // 设备唯一标识
	TempMax      *float64 `json:"temp_max"`                     // 温度上限，null = 不监控
	TempMin      *float64 `json:"temp_min"`                     // 温度下限
	HumidityMax  *float64 `json:"humidity_max"`                 // 湿度上限
	HumidityMin  *float64 `json:"humidity_min"`                 // 湿度下限
	AlertSeconds int      `json:"alert_seconds,omitempty"`      // 报警最小间隔秒数，默认 300（5分钟）
	IsActive     bool     `json:"is_active,omitempty"`          // 是否启用，默认 true
}

type DeviceAlertRequest struct {
	DeviceID    string  `json:"device_id" binding:"required"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

// 距离阈值设置请求
type DistanceThresholdRequest struct {
	DeviceID     string   `json:"device_id" binding:"required"` // 设备唯一标识
	DistanceMin  *float64 `json:"distance_min"`                 // 距离下限（小于此值报警），null = 不监控
	AlertSeconds int      `json:"alert_seconds,omitempty"`      // 报警最小间隔秒数，默认 300（5分钟）
	IsActive     bool     `json:"is_active,omitempty"`          // 是否启用，默认 true
}

// 距离上报请求
type DistanceAlertRequest struct {
	DeviceID string  `json:"device_id" binding:"required"`
	Distance float64 `json:"distance"` // 当前距离值
}

// 统一传感器数据上报请求（包含温湿度和距离）
type SensorDataRequest struct {
	DeviceID    string  `json:"device_id" binding:"required"`
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Distance    float64 `json:"distance"`
	Time        string  `json:"time,omitempty"` // 可选：设备时间
}
