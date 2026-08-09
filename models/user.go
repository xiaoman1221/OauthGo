package models

import "time"

// User 用户
type User struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Username    string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Nickname    string    `gorm:"size:64" json:"nickname"`
	Avatar      string    `gorm:"size:512" json:"avatar"`
	Password    string    `gorm:"size:255;not null" json:"-"`
	PasswordSet bool      `json:"password_set"` // 是否设置了真实密码（第三方注册用户为 false）
	Email       string    `gorm:"size:128" json:"email"`
	Phone       string    `gorm:"size:32" json:"phone"`
	Role        string    `gorm:"size:32;default:user" json:"role"` // admin / user
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	ProviderAccounts []ProviderAccount `gorm:"foreignKey:UserID" json:"provider_accounts,omitempty"`
}
