package controllers

import (
	"awesomeProject/internal/model"
	"awesomeProject/internal/service"
	"bytes"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	_ "image/png" // 支持 png
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// 控制灯
type LedControlRequest struct {
	DeviceID   string `json:"device_id" binding:"required"`
	State      string `json:"state" binding:"required,oneof=on off"`
	Color      string `json:"color,omitempty"`
	Brightness *int   `json:"brightness,omitempty"`
}

// 控制文字
type ScreenTextRequest struct {
	DeviceID        string `json:"device_id" binding:"required"`
	Text            string `json:"text" binding:"required,max=200"` // 文字，必填，限制长度
	Duration        int    `json:"duration,omitempty"`              // 显示秒数，可选
	Scroll          bool   `json:"scroll,omitempty"`                // 是否滚动，可选
	FontSize        int    `json:"font_size,omitempty"`             // 字体大小，可选
	TextColor       string `json:"text_color,omitempty"`            // 新增：文字颜色，可选 (e.g., "red" or "#FF0000")
	BackgroundColor string `json:"background_color,omitempty"`      // 新增：背景颜色，可选 (e.g., "white" or "#FFFFFF")
	Font            string `json:"font,omitempty"`                  // 新增：字体名称，可选 (e.g., "Arial")
}

// 上传图片
type ScreenImageRequest struct {
	DeviceID string `form:"device_id" binding:"required"`
	Duration int    `form:"duration"`
	Fit      string `form:"fit"` // cover, contain, fill
}

// 图片的url
type ScreenImageUrlRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Url      string `json:"url" binding:"required"`
	Duration int    `json:"duration"`
	Fit      string `json:"fit"` // cover, contain, fill
}

// 控制蜂鸣器
type BuzzerControlRequest struct {
	DeviceID  string `json:"device_id" binding:"required"`
	State     string `json:"state" binding:"required,oneof=on off"`
	Frequency *int   `json:"frequency,omitempty"` // 频率 Hz
	Duration  *int   `json:"duration,omitempty"`  // 每个蜂鸣 ms
	Interval  *int   `json:"interval,omitempty"`  // 间隔 ms
	Cycles    *int   `json:"cycles,omitempty"`    // 循环次数
}

func ControlLed(c *gin.Context) {
	var req LedControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误：" + err.Error(),
		})
		return
	}

	// 验证设备是否绑定到当前用户
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 检查设备是否绑定到当前用户
	var userDevice model.UserDevice
	if err := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).First(&userDevice).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "设备未绑定到当前用户",
		})
		return
	}

	payload := map[string]interface{}{
		"state": req.State,
	}

	if req.State == "on" {
		if req.Color != "" {
			payload["color"] = req.Color
		} else {
			payload["color"] = "white"
		}

		if req.Brightness != nil {
			if *req.Brightness < 0 || *req.Brightness > 100 {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": "brightness 必须在 0~100 之间",
				})
				return
			}
			payload["brightness"] = *req.Brightness
		} else {
			payload["brightness"] = 100
		}
	} else {
		payload["color"] = "black"
		payload["brightness"] = 0
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

	c.JSON(http.StatusOK, gin.H{
		"code":      200,
		"message":   "灯控制命令已发送",
		"sent":      payload,
		"device_id": req.DeviceID,
	})
}

func SendScreenText(c *gin.Context) {
	var req ScreenTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误：" + err.Error(),
		})
		return
	}

	// 验证设备是否绑定到当前用户
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 检查设备是否绑定到当前用户
	var userDevice model.UserDevice
	if err := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).First(&userDevice).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "设备未绑定到当前用户",
		})
		return
	}

	// 构造 MQTT payload
	payload := map[string]interface{}{
		"type":    "text",
		"content": req.Text,
	}

	// 显示时长（默认 10 秒）
	if req.Duration > 0 {
		payload["duration"] = req.Duration
	} else {
		payload["duration"] = 10
	}

	// 滚动开关（默认 false）
	payload["scroll"] = req.Scroll

	// 字体大小（如果传了就加）
	if req.FontSize > 0 {
		payload["font_size"] = req.FontSize
	}

	// 新增：文字颜色（默认 "black"）
	if req.TextColor != "" {
		payload["text_color"] = req.TextColor
	} else {
		payload["text_color"] = "black" // 或固件默认
	}

	// 新增：背景颜色（默认 "white"）
	if req.BackgroundColor != "" {
		payload["background_color"] = req.BackgroundColor
	} else {
		payload["background_color"] = "white" // 或固件默认
	}

	// 新增：字体（如果传了就加）
	if req.Font != "" {
		payload["font"] = req.Font
	}

	// 发送到 MQTT
	topic := "devices/" + req.DeviceID + "/control"
	err := service.Publish(topic, 0, false, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "MQTT 发送失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      200,
		"message":   "文字已发送到屏幕",
		"sent":      payload,
		"device_id": req.DeviceID,
	})
}

func UploadAndSendImage(c *gin.Context) {
	var req ScreenImageRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "参数错误"})
		return
	}

	// 验证设备是否绑定到当前用户
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 检查设备是否绑定到当前用户
	var userDevice model.UserDevice
	if err := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).First(&userDevice).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "设备未绑定到当前用户",
		})
		return
	}

	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少图片文件"})
		return
	}

	// 限制大小：最大 200KB（ESP32 接收缓冲建议小一点）
	if file.Size > 200*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "图片过大，最大 200KB"})
		return
	}

	// 检查文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "仅支持 jpg/png 格式"})
		return
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "打开文件失败"})
		return
	}
	defer src.Close()

	// 读取图片
	img, _, err := image.Decode(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效图片"})
		return
	}

	// 强制 resize 到 240×240（或根据 fit 模式调整）
	var resizedImg image.Image
	if req.Fit == "contain" {
		resizedImg = resize.Thumbnail(240, 240, img, resize.Lanczos3)
	} else {
		// 默认 cover 或 fill：直接缩放到 240×240（可能拉伸/裁剪）
		resizedImg = resize.Resize(240, 240, img, resize.Lanczos3)
	}

	// 编码成 JPEG（质量 60~75，平衡大小和质量）
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, resizedImg, &jpeg.Options{Quality: 70})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "图片编码失败"})
		return
	}

	imgData := buf.Bytes()

	// 构造 payload（建议用 base64 或直接二进制，但 MQTT 通常文本安全起见用 base64）
	payload := map[string]interface{}{
		"type":     "image",
		"format":   "jpeg",
		"width":    240,
		"height":   240,
		"data":     base64.StdEncoding.EncodeToString(imgData),
		"duration": req.Duration,
	}
	if req.Duration == 0 {
		payload["duration"] = 10
	}

	// 发送到 MQTT（建议用独立主题）
	topic := "devices/" + req.DeviceID + "/control"
	err = service.Publish(topic, 0, false, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "MQTT 发送失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "图片已发送到屏幕",
		"size_kb":   len(imgData) / 1024,
		"device_id": req.DeviceID,
	})
}

func SendScreenImageFromUrl(c *gin.Context) {
	var req ScreenImageUrlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误：" + err.Error(),
		})
		return
	}

	// 验证设备是否绑定到当前用户
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 检查设备是否绑定到当前用户
	var userDevice model.UserDevice
	if err := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).First(&userDevice).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "设备未绑定到当前用户",
		})
		return
	}

	// 校验 URL 格式
	parsedUrl, err := url.Parse(req.Url)
	if err != nil || (parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的图片 URL"})
		return
	}

	// 下载图片
	resp, err := http.Get(req.Url)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无法下载图片"})
		return
	}
	defer resp.Body.Close()

	// 限制大小（防止下载超大文件）
	if resp.ContentLength > 500*1024 { // 500KB
		c.JSON(http.StatusBadRequest, gin.H{"message": "图片过大"})
		return
	}

	// 读取图片 bytes
	imgBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "读取图片失败"})
		return
	}

	// 解码图片
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的图片格式"})
		return
	}

	// resize 到 240×240（根据 fit 模式）
	var resized image.Image
	if req.Fit == "contain" {
		resized = resize.Thumbnail(240, 240, img, resize.Lanczos3)
	} else {
		// 默认 cover/fill：强制缩放（可能裁剪/拉伸）
		resized = resize.Resize(240, 240, img, resize.Lanczos3)
	}

	// 转 JPEG（质量 60~70，控制大小）
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 70})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "图片处理失败"})
		return
	}

	finalData := buf.Bytes()

	// 构造 payload
	payload := map[string]interface{}{
		"type":     "image",
		"format":   "jpeg",
		"width":    240,
		"height":   240,
		"data":     base64.StdEncoding.EncodeToString(finalData),
		"duration": req.Duration,
	}
	if req.Duration == 0 {
		payload["duration"] = 10
	}

	// 发送 MQTT
	topic := "devices/" + req.DeviceID + "/control"
	err = service.Publish(topic, 0, false, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "MQTT 发送失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":      200,
		"message":   "图片已从 URL 下载并发送到屏幕",
		"size_kb":   len(finalData) / 1024,
		"device_id": req.DeviceID,
	})
}

func ControlBuzzer(c *gin.Context) {
	var req BuzzerControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数格式错误：" + err.Error(),
		})
		return
	}

	// 验证设备是否绑定到当前用户
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "请先登录",
		})
		return
	}
	userID := uint(userIDFloat.(float64))

	// 检查设备是否绑定到当前用户
	var userDevice model.UserDevice
	if err := model.DB.Where("user_id = ? AND device_id = ?", userID, req.DeviceID).First(&userDevice).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "设备未绑定到当前用户",
		})
		return
	}

	// 构造 payload
	payload := map[string]interface{}{
		"type":  "buzzer",
		"state": req.State,
	}

	if req.State == "on" {
		// 频率（默认 2000 Hz）
		if req.Frequency != nil {
			if *req.Frequency < 100 || *req.Frequency > 5000 {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": "frequency 必须在 100~5000 Hz 之间",
				})
				return
			}
			payload["frequency"] = *req.Frequency
		} else {
			payload["frequency"] = 2000
		}

		// 持续时间（默认 500 ms）
		if req.Duration != nil {
			if *req.Duration < 50 || *req.Duration > 5000 {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": "duration 必须在 50~5000 ms 之间",
				})
				return
			}
			payload["duration"] = *req.Duration
		} else {
			payload["duration"] = 500
		}

		// 间隔时间（默认 200 ms）
		if req.Interval != nil {
			if *req.Interval < 0 || *req.Interval > 5000 {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": "interval 必须在 0~5000 ms 之间",
				})
				return
			}
			payload["interval"] = *req.Interval
		} else {
			payload["interval"] = 200
		}

		// 循环次数（默认 1，-1 表示无限）
		if req.Cycles != nil {
			if *req.Cycles < -1 || *req.Cycles == 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    400,
					"message": "cycles 必须 >=1 或 -1（无限）",
				})
				return
			}
			payload["cycles"] = *req.Cycles
		} else {
			payload["cycles"] = 1
		}
	} else {
		// state == "off" 时，忽略其他参数，只发关闭指令
		payload["frequency"] = 0
		payload["duration"] = 0
		payload["interval"] = 0
		payload["cycles"] = 0
	}

	// 发送到 MQTT
	topic := "devices/" + req.DeviceID + "/control"
	err := service.Publish(topic, 0, false, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "MQTT 发送失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      200,
		"message":   "蜂鸣器命令已发送",
		"sent":      payload,
		"device_id": req.DeviceID,
	})
}
