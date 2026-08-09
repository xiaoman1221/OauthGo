package middleware

import (
	"strings"

	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// JWT JWT 认证中间件，校验 Authorization: Bearer <token>
func JWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			utils.FailUnauthorized(c)
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			utils.FailUnauthorized(c)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// AdminOnly 管理员权限校验中间件
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			utils.FailForbidden(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
