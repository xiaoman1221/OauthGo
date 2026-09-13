package services

import (
	"errors"
	"strings"
	"time"

	"OauthGo/config"
	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/utils"
)

const (
	// SessionCookieName OIDC 登录会话 Cookie 名
	SessionCookieName = "oauthgo_session"
	// SessionTTL 会话有效期（滑动续期：每次使用自动顺延）
	SessionTTL = 7 * 24 * time.Hour
	// sessionRenewInterval 距上次使用超过该间隔才回写续期，避免每个请求都写库
	sessionRenewInterval = time.Hour
	// sessionIDLen 会话 ID 长度（256 位随机）
	sessionIDLen = 64
)

// CreateSession 为用户建立登录会话，返回会话 ID（作为 Cookie 值）
func CreateSession(userID uint, ip, userAgent string) (string, error) {
	if userID == 0 {
		return "", errors.New("用户不存在")
	}
	s := models.UserSession{
		ID:         utils.RandomString(sessionIDLen),
		UserID:     userID,
		IP:         ip,
		UserAgent:  userAgent,
		LastUsedAt: time.Now(),
		ExpiresAt:  time.Now().Add(SessionTTL),
	}
	if err := database.DB.Create(&s).Error; err != nil {
		return "", err
	}
	return s.ID, nil
}

// ValidateSession 校验会话有效性并滑动续期；无效返回 nil
func ValidateSession(id string) *models.UserSession {
	if len(id) != sessionIDLen {
		return nil
	}
	var s models.UserSession
	if err := database.DB.Where("id = ?", id).First(&s).Error; err != nil {
		return nil
	}
	now := time.Now()
	if now.After(s.ExpiresAt) {
		database.DB.Delete(&s)
		return nil
	}
	// 滑动续期：距上次使用超过 1 小时才回写，避免高频请求写库
	if now.Sub(s.LastUsedAt) >= sessionRenewInterval {
		updates := map[string]interface{}{"last_used_at": now, "expires_at": now.Add(SessionTTL)}
		_ = database.DB.Model(&s).Updates(updates).Error
		s.LastUsedAt = now
		s.ExpiresAt = now.Add(SessionTTL)
	}
	return &s
}

// DeleteSession 删除单个会话
func DeleteSession(id string) {
	if id != "" {
		database.DB.Where("id = ?", id).Delete(&models.UserSession{})
	}
}

// DeleteUserSessions 吊销用户全部会话（改密 / 重置密码 / 删除用户时调用）
func DeleteUserSessions(userID uint) {
	database.DB.Where("user_id = ?", userID).Delete(&models.UserSession{})
}

// ListUserSessions 列出用户全部有效会话（按最近使用排序）
func ListUserSessions(userID uint) ([]models.UserSession, error) {
	var list []models.UserSession
	err := database.DB.Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Order("last_used_at desc").Find(&list).Error
	return list, err
}

// SessionCookieSecure 依据对外 HOST 判断会话 Cookie 是否附加 Secure 属性
func SessionCookieSecure() bool {
	return strings.HasPrefix(strings.ToLower(config.AppConfig.Host), "https://")
}
