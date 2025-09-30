package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Login(c *gin.Context) {
	//http.Handle()UserHandler
	c.JSON(http.StatusOK, gin.H{"msg": "login success", "user": user})
}

func Logout(c *gin.Context) {
	c.JSON(200, gin.H{"msg": "logout success"})
}
