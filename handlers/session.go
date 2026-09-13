package handlers

import (
	"net/http"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// sessionFromCookie 从请求读取并校验 OIDC 登录会话；无效返回 nil
func sessionFromCookie(c *gin.Context) *models.UserSession {
	ck, err := c.Cookie(services.SessionCookieName)
	if err != nil || ck == "" {
		return nil
	}
	return services.ValidateSession(ck)
}

// setSessionCookie 下发会话 Cookie（HttpOnly + SameSite=Lax；HTTPS 部署时附加 Secure）
func setSessionCookie(c *gin.Context, sessionID string, expires time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     services.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   services.SessionCookieSecure() || c.Request.TLS != nil,
	})
}

// clearSessionCookie 清除会话 Cookie
func clearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     services.SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   services.SessionCookieSecure() || c.Request.TLS != nil,
	})
}

// establishSession 建立新会话并下发 Cookie；旧会话轮换删除（登录时调用）
func establishSession(c *gin.Context, userID uint) {
	if old, err := c.Cookie(services.SessionCookieName); err == nil && old != "" {
		services.DeleteSession(old)
	}
	id, err := services.CreateSession(userID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		return // 会话建立失败不阻断登录主流程
	}
	setSessionCookie(c, id, time.Now().Add(services.SessionTTL))
}

// Logout 退出登录：吊销 OIDC 会话并清除 Cookie（会话 Cookie 即凭据，无需 JWT）
// POST /api/auth/logout
func Logout(c *gin.Context) {
	if old, err := c.Cookie(services.SessionCookieName); err == nil && old != "" {
		services.DeleteSession(old)
	}
	clearSessionCookie(c)
	utils.SuccessMsg(c, "已退出登录")
}

// ListSessions 当前用户的有效会话列表（用户中心「在线会话」）
// GET /api/auth/sessions
func ListSessions(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	list, err := services.ListUserSessions(uid)
	if err != nil {
		utils.FailInternal(c, "查询失败")
		return
	}
	current, _ := c.Cookie(services.SessionCookieName)
	items := make([]gin.H, 0, len(list))
	for _, s := range list {
		items = append(items, gin.H{
			"id":           s.ID,
			"ip":           s.IP,
			"user_agent":   s.UserAgent,
			"created_at":   s.CreatedAt,
			"last_used_at": s.LastUsedAt,
			"expires_at":   s.ExpiresAt,
			"current":      s.ID == current,
		})
	}
	utils.Success(c, gin.H{"list": items})
}

// RevokeSession 吊销当前用户的一条会话
// DELETE /api/auth/sessions/:id
func RevokeSession(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	id := c.Param("id")
	var row models.UserSession
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&row).Error; err != nil {
		utils.FailNotFound(c, "会话不存在")
		return
	}
	services.DeleteSession(id)
	// 吊销的是当前会话时同步清 Cookie
	if cur, err := c.Cookie(services.SessionCookieName); err == nil && cur == id {
		clearSessionCookie(c)
	}
	utils.SuccessMsg(c, "已吊销")
}
