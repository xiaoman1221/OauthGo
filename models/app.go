package models

import "time"

// App 目标站点（使用本平台第三方登录服务的应用）
type App struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	OwnerID       uint      `gorm:"index" json:"owner_id"`
	Name          string    `gorm:"size:128;not null" json:"name"`
	Platform      string    `gorm:"size:64" json:"platform"` // web / ios / android / pc
	AppID         string    `gorm:"uniqueIndex;size:64;not null" json:"appid"`
	AppKey        string    `gorm:"size:128" json:"app_key"`
	Mode          string    `gorm:"size:16;default:compat" json:"mode"`  // rainbow 仅彩虹协议 / rest 仅 REST / oauth2 仅 OAuth2/OIDC / compat 兼容
	Types         string    `gorm:"type:text" json:"-"`                  // JSON 数组：该目标站点支持的登录类型（provider name）
	Domains       string    `gorm:"type:text" json:"domains"`            // 回调白名单域名（每行一个，区分子域名）
	RedirectURIs  string    `gorm:"type:text" json:"redirect_uris"`      // OAuth2/OIDC 回调地址白名单（JSON 数组，按 scheme+host+path 匹配，忽略 query）
	SampleParams  string    `gorm:"type:text" json:"sample_params"`      // 接入示例参数（JSON 对象）：接入方自带参数的样例值，仅用于生成接入文档示例，不参与校验
	EnableRefresh bool      `gorm:"default:false" json:"enable_refresh"` // OAuth2 是否签发 refresh_token
	Status        int       `gorm:"default:1" json:"status"`             // 1 启用 / 0 禁用
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
