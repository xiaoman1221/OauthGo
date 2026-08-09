package models

import "time"

// ProviderAccount 用户与第三方渠道账号的绑定关系
type ProviderAccount struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Provider  string    `gorm:"size:64;index;not null" json:"provider"`
	OpenID    string    `gorm:"size:255;index;not null" json:"open_id"`
	UnionID   string    `gorm:"size:255" json:"union_id"`
	Nickname  string    `gorm:"size:128" json:"nickname"`
	Avatar    string    `gorm:"size:512" json:"avatar"`
	Email     string    `gorm:"size:128" json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
