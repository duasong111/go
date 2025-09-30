package handler

import (
	"awesomeProject/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

func UserHandler(c *gin.Context) {
	id := c.Param("id")
	user, err := service.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)

}
