package middleware

import (
	"strings"

	"OauthGo/database"
	"OauthGo/models"
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

// AdminOnly 管理员权限校验中间件。
// 角色实时查库校验而非信任 JWT 内嵌声明：管理员被降级 / 删除后立即失去后台权限，
// 不必等待旧令牌过期（JWT 有效期 24 小时）。
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		uid, _ := userID.(uint)
		var user models.User
		if err := database.DB.Select("role").First(&user, uid).Error; err != nil || user.Role != "admin" {
			utils.FailForbidden(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
