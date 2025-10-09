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

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
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

func (h *UserHandler) Login(c *gin.Context) {
	// 您的登录逻辑
	c.JSON(http.StatusOK, gin.H{"msg": "login success", "user": ""})
}

func (h *UserHandler) Logout(c *gin.Context) {
	// 您的登出逻辑
	c.JSON(http.StatusOK, gin.H{"msg": "logout success"})
}
