package controllers

import (
	"awesomeProject/internal/service"
	"awesomeProject/pkg"
	"github.com/gin-gonic/gin"
	"net/http"
)

type UserHandler struct {
	userService *service.UserService
}

// 用户注册 --
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}

	user, err := h.userService.Register(req.Username, req.Password, req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	pkg.RightResponse(c, gin.H{
		"user_id":  user.ID,
		"username": user.Username,
	}, "注册成功")
}

// 用户登录，登出 控制层
func Login(c *gin.Context) {
	//http.Handle()UserHandler
	c.JSON(http.StatusOK, gin.H{"msg": "login success", "user": ""})
}

func Logout(c *gin.Context) {
	c.JSON(200, gin.H{"msg": "logout success"})
}
