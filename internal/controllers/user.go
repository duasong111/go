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
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	user, token, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": err.Error()})
		return
	}
	pkg.RightResponse(c, gin.H{
		"token":    token,
		"user_id":  user.ID,
		"username": user.Username,
	}, "登录成功")
}

func (h *UserHandler) Logout(c *gin.Context) {
	method := c.Request.Method
	if method != http.MethodPost {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"success": false, "message": "仅支持 POST 方法"})
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未认证用户"})
		return
	}
	if err := h.userService.Logout(uint(userID.(float64))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "登出失败"})
		return
	}
	pkg.RightResponse(c, gin.H{
		"user_id": userID,
	}, "登出成功")
}
