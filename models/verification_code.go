package models

import "time"

// VerificationCode 验证码（注册、找回密码等）
type VerificationCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Scope     string    `gorm:"size:32;index;not null" json:"scope"` // register / reset
	Account   string    `gorm:"size:255;index;not null" json:"account"`
	Code      string    `gorm:"size:16;not null" json:"-"`
	Used      bool      `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
