package middleware

import (
	"awesomeProject/internal/redis"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未提供 token"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			secret = "K9f4zB2qX8vL7nA1pR6sT5wM3cN9xY2hV7jQ4mE6oI5uP8tW1rS3eD7yH6kL9vC4n"
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "无效 token: " + err.Error()})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "无效 token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID := claims["user_id"]
			c.Set("user_id", userID)
			c.Set("username", claims["username"])

			// 检查 token 是否在 Redis 中存在（单点登录验证）
			ctx := context.Background()
			tokenKey := fmt.Sprintf("user:%v:token", userID)
			storedToken, err := redis.Client.Get(ctx, tokenKey).Result()
			if err != nil || storedToken != tokenStr {
				// token 不存在或不匹配，说明已被其他登录覆盖
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "登录已过期，请重新登录"})
				c.Abort()
				return
			}
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "无效 token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
