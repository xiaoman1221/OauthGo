package models

import "time"

// UserSession 平台登录会话：记录当前登录用户，实现控制台与 OIDC 授权页的单点登录。
// ID 即会话 Cookie 值（HttpOnly + SameSite=Lax），7 天滑动续期。
type UserSession struct {
	ID         string    `gorm:"primaryKey;size:64;not null" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	IP         string    `gorm:"size:64" json:"ip"`
	UserAgent  string    `gorm:"size:255" json:"user_agent"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}
