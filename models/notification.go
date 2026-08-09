package models

import "time"

// NotificationChannel 通知渠道
type NotificationChannel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Type      string    `gorm:"size:32;not null" json:"type"` // email / webhook / bark
	Config    string    `gorm:"type:text" json:"config"`      // 渠道配置（JSON）
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NotificationLog 通知日志
type NotificationLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ChannelID   uint      `json:"channel_id"`
	ChannelName string    `gorm:"size:128" json:"channel_name"`
	Target      string    `gorm:"size:255" json:"target"`
	Subject     string    `gorm:"size:255" json:"subject"`
	Content     string    `gorm:"type:text" json:"content"`
	Status      int       `gorm:"default:0" json:"status"` // 0 待发送 / 1 成功 / 2 失败
	Error       string    `gorm:"type:text" json:"error"`
	CreatedAt   time.Time `json:"created_at"`
}
