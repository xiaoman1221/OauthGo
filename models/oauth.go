package models

import "time"

// OAuthCode 授权码（authorization code flow，一次性）。
// 对应 RFC 6749 §4.1.2：目标站点（OAuth2/OIDC 客户端）用其换取访问令牌。
type OAuthCode struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Code          string    `gorm:"uniqueIndex;size:128;not null" json:"-"`
	ClientID      string    `gorm:"size:64;index;not null" json:"client_id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	Scope         string    `gorm:"size:255" json:"scope"`
	RedirectURI   string    `gorm:"size:512" json:"redirect_uri"`
	PKCEChallenge string    `gorm:"size:255" json:"-"` // code_challenge（可选 PKCE）
	PKCEMethod    string    `gorm:"size:16" json:"-"`  // S256 / plain
	ExpiresAt     time.Time `json:"expires_at"`
	Used          bool      `json:"-"`
	CreatedAt     time.Time `json:"created_at"`
}

// OAuthAccessToken OAuth2 访问令牌（opaque token，服务端查库校验）
type OAuthAccessToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Token     string    `gorm:"uniqueIndex;size:128;not null" json:"-"`
	ClientID  string    `gorm:"size:64;index;not null" json:"client_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Scope     string    `gorm:"size:255" json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// OAuthRefreshToken OAuth2 刷新令牌（轮换制）
type OAuthRefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Token     string    `gorm:"uniqueIndex;size:128;not null" json:"-"`
	ClientID  string    `gorm:"size:64;index;not null" json:"client_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Scope     string    `gorm:"size:255" json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
