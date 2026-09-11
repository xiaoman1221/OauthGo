package models

import "time"

// PlatformLoginCode 平台登录一次性码：前端以 code 兑换平台 JWT。
// 用于主站第三方登录 / Passkey 登录回跳，避免 JWT 直接出现在跳转 URL（浏览器历史）中。
type PlatformLoginCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"uniqueIndex;size:64;not null" json:"-"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}
