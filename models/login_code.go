package models

import "time"

// LoginCode 目标站点登录授权码（code 换用户信息，一次性）
type LoginCode struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"uniqueIndex;size:64;not null" json:"-"`
	AppID       string    `gorm:"size:64;index;not null" json:"appid"`
	Type        string    `gorm:"size:64" json:"type"`     // 请求中的登录类型（如 qq / wx）
	Provider    string    `gorm:"size:64" json:"provider"` // 实际 provider name
	OpenID      string    `gorm:"size:128;index" json:"open_id"`
	UnionID     string    `gorm:"size:128" json:"union_id"`
	Nickname    string    `gorm:"size:128" json:"nickname"`
	Avatar      string    `gorm:"size:255" json:"avatar"`
	Email       string    `gorm:"size:128" json:"email"`
	Gender      string    `gorm:"size:32" json:"gender"`
	Location    string    `gorm:"size:128" json:"location"`
	AccessToken string    `gorm:"size:255" json:"access_token"`
	IP          string    `gorm:"size:64" json:"ip"`
	ExpiresAt   time.Time `json:"expires_at"`
	Used        bool      `gorm:"default:false" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}
