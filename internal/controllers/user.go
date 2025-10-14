package controllers

import (
	"awesomeProject/internal/service"
	"awesomeProject/pkg"
	"fmt"
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
		pkg.ErrorResponse(c, http.StatusBadRequest, "参数错误")
		return
	}
	user, err := h.userService.Register(req.Username, req.Password, req.Email)
	if err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err.Error())
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
		pkg.ErrorResponse(c, http.StatusBadRequest, "参数错误")
		return
	}
	fmt.Print("Username", req.Username, req.Password)
	user, token, err := h.userService.Login(req.Username, req.Password)
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}
	pkg.RightResponse(c, gin.H{
		"token":    token,
		"user_id":  user.ID,
		"username": user.Username,
	}, "登录成功")
}

func (h *UserHandler) Update(c *gin.Context) {
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		pkg.ErrorResponse(c, http.StatusUnauthorized, "未认证用户")
		return
	}
	userID := uint(userIDFloat.(float64))

	var req struct {
		Email     *string `json:"email"`
		Address   *string `json:"address"`
		Phone     *string `json:"phone"`
		Education *string `json:"education"`
		Avatar    *string `json:"avatar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, "参数错误")
		return
	}

	updates := make(map[string]interface{})
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Education != nil {
		updates["education"] = *req.Education
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}

	if len(updates) == 0 {
		pkg.ErrorResponse(c, http.StatusUnauthorized, "无更新内容")
		return
	}

	// 调用 Service 更新
	updatedUser, err := h.userService.UpdateUser(userID, updates)
	if err != nil {
		pkg.ErrorResponse(c, http.StatusUnauthorized, err.Error())
		return
	}

	pkg.RightResponse(c, gin.H{
		"user_id":   updatedUser.ID,
		"username":  updatedUser.Username,
		"email":     updatedUser.Email,
		"address":   updatedUser.Address,
		"phone":     updatedUser.Phone,
		"education": updatedUser.Education,
		"avatar":    updatedUser.Avatar,
	}, "更新成功")
}

func (h *UserHandler) ModifyPW(c *gin.Context) {
	userIDFloat, exists := c.Get("user_id")
	if !exists {
		pkg.ErrorResponse(c, http.StatusMethodNotAllowed, "未认证用户")
		return
	}
	userID := uint(userIDFloat.(float64))

	var req struct {
		OldPassword     string `json:"old_password" binding:"required,min=6"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
		ConfirmPassword string `json:"confirm_password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusMethodNotAllowed, "参数错误")
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		pkg.ErrorResponse(c, http.StatusBadRequest, "新密码与确认密码不匹配")
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// 调用 Service 修改密码
	if err := h.userService.ModifyPassword(userID, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}

	// 响应成功（无需返回用户数据，避免敏感信息）
	pkg.RightResponse(c, gin.H{}, "密码修改成功")
}

func (h *UserHandler) Logout(c *gin.Context) {
	method := c.Request.Method
	if method != http.MethodPost {
		pkg.ErrorResponse(c, http.StatusMethodNotAllowed, "仅支持 POST 方法")
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		pkg.ErrorResponse(c, http.StatusUnauthorized, "未认证用户")
		return
	}
	if err := h.userService.Logout(uint(userID.(float64))); err != nil {
		pkg.ErrorResponse(c, http.StatusInternalServerError, "登出失败")
		return
	}
	pkg.RightResponse(c, gin.H{
		"user_id": userID,
	}, "登出成功")
}
