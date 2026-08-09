package models

import "time"

// App 目标站点（使用本平台第三方登录服务的应用）
type App struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Platform  string    `gorm:"size:64" json:"platform"` // web / ios / android / pc
	AppID     string    `gorm:"uniqueIndex;size:64;not null" json:"appid"`
	AppKey    string    `gorm:"size:128" json:"app_key"`
	Mode      string    `gorm:"size:16;default:compat" json:"mode"` // rainbow 仅彩虹协议 / rest 仅REST / compat 兼容
	Types     string    `gorm:"type:text" json:"-"`                 // JSON 数组：该目标站点支持的登录类型（provider name）
	Domains   string    `gorm:"type:text" json:"domains"`           // 回调白名单域名（每行一个，区分子域名）
	Status    int       `gorm:"default:1" json:"status"`            // 1 启用 / 0 禁用
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
