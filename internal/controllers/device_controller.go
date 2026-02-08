package controllers

import (
	"net/http"

	"awesomeProject/internal/service"

	"github.com/gin-gonic/gin"
)

// 控制灯
type LedControlRequest struct {
	State      string `json:"state" binding:"required,oneof=on off"`
	Color      string `json:"color,omitempty"`
	Brightness *int   `json:"brightness,omitempty"`
}

// 控制文字

type ScreenTextRequest struct {
	Text            string `json:"text" binding:"required,max=200"` // 文字，必填，限制长度
	Duration        int    `json:"duration,omitempty"`              // 显示秒数，可选
	Scroll          bool   `json:"scroll,omitempty"`                // 是否滚动，可选
	FontSize        int    `json:"font_size,omitempty"`             // 字体大小，可选
	TextColor       string `json:"text_color,omitempty"`            // 新增：文字颜色，可选 (e.g., "red" or "#FF0000")
	BackgroundColor string `json:"background_color,omitempty"`      // 新增：背景颜色，可选 (e.g., "white" or "#FFFFFF")
	Font            string `json:"font,omitempty"`                  // 新增：字体名称，可选 (e.g., "Arial")
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
	err := service.Publish("control/esp32", 0, false, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "MQTT 命令发送失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "灯控制命令已发送",
		"sent":    payload,
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
	err := service.Publish("control/esp32", 0, false, payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "MQTT 发送失败：" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "文字已发送到屏幕",
		"sent":    payload,
	})
}
