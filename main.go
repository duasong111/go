package main

import (
	"awesomeProject/internal/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	routes.RegisterRoutes(r)
	r.Run("0.0.0.0:8080")
}
