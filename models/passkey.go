package models

import "time"

// PasskeyCredential 用户注册的 WebAuthn 通行密钥（Passkey）
// CredentialJSON 保存序列化的 webauthn.Credential（含公钥、签名计数、传输方式等）
type PasskeyCredential struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"index;not null" json:"user_id"`
	Name           string     `gorm:"size:128" json:"name"`
	CredentialID   string     `gorm:"size:512;index;not null" json:"credential_id"` // base64url(rawID)
	CredentialJSON string     `gorm:"type:text;not null" json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	LastUsedAt     *time.Time `json:"last_used_at"`
}
