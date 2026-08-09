package models

import "time"

// LoginRecord 登录记录
type LoginRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	AppID     uint      `json:"app_id"`
	AppName   string    `gorm:"size:128" json:"app_name"`
	OpenID    string    `gorm:"size:128;index" json:"open_id"`
	Username  string    `gorm:"size:128" json:"username"`
	Nickname  string    `gorm:"size:128" json:"nickname"`
	Avatar    string    `gorm:"size:255" json:"avatar"`
	Platform  string    `gorm:"size:64" json:"platform"`
	IP        string    `gorm:"size:64" json:"ip"`
	Location  string    `gorm:"size:128" json:"location"`
	UserAgent string    `gorm:"size:255" json:"user_agent"`
	Status    int       `gorm:"default:1" json:"status"` // 1 成功 / 0 失败
	LoginTime time.Time `json:"login_time"`
	CreatedAt time.Time `json:"created_at"`
}
