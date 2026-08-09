package models

import "time"

// Provider 第三方登录渠道配置
type Provider struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:64;uniqueIndex;not null" json:"name"`
	DisplayName  string    `gorm:"size:128" json:"display_name"`
	Category     string    `gorm:"size:32" json:"category"` // social / enterprise
	ClientID     string    `gorm:"size:255" json:"client_id"`
	ClientSecret string    `gorm:"size:512" json:"client_secret"`
	RedirectURL  string    `gorm:"size:512" json:"redirect_url"`
	Config       string    `gorm:"type:text" json:"config"` // 扩展配置（JSON）
	Enabled      bool      `gorm:"default:false" json:"enabled"`
	MainSite     bool      `gorm:"default:true" json:"main_site"` // 是否用于主站登录页
	Sort         int       `json:"sort"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
