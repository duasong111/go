package pkg

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Response struct {
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
}

// 数据返回成功
func RightResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, Response{
		StatusCode: http.StatusOK,
		Message:    message,
		Success:    true,
		Data:       data,
	})
}

// 数据返回失败
func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, Response{
		StatusCode: statusCode,
		Message:    message,
		Success:    false,
		Data:       nil,
	})
}
