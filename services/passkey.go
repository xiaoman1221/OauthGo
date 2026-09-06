package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"OauthGo/config"
	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/utils"

	"github.com/go-webauthn/webauthn/webauthn"
)

// PasskeyUser 实现 webauthn.User 接口的平台用户包装。
// 用户句柄 = "oauthgo-user-{id}"（<=64 字节）。
type PasskeyUser struct {
	User        models.User
	Credentials []webauthn.Credential
}

func (u *PasskeyUser) WebAuthnID() []byte {
	return []byte(fmt.Sprintf("oauthgo-user-%d", u.User.ID))
}

func (u *PasskeyUser) WebAuthnName() string { return u.User.Username }

func (u *PasskeyUser) WebAuthnDisplayName() string {
	if u.User.Nickname != "" {
		return u.User.Nickname
	}
	return u.User.Username
}

func (u *PasskeyUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// PasskeyRPConfig 解析 HOST 得到 WebAuthn Relying Party 配置
type PasskeyRPConfig struct {
	RPID        string
	RPOrigin    string
	RPName      string
	IsLocalhost bool
}

// PasskeyRPFromHost 根据配置的 HOST 推导 RP 参数
func PasskeyRPFromHost() PasskeyRPConfig {
	host := strings.TrimSuffix(config.AppConfig.Host, "/")
	if host == "" {
		host = "http://localhost:8080"
	}
	u, err := url.Parse(host)
	if err != nil {
		return PasskeyRPConfig{RPID: "localhost", RPOrigin: "http://localhost:8080", RPName: "OauthGo", IsLocalhost: true}
	}
	rp := PasskeyRPConfig{
		RPID:        u.Hostname(),
		RPOrigin:    host,
		RPName:      GetSetting("site_name", "OauthGo"),
		IsLocalhost: u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1",
	}
	return rp
}

// NewPasskeyWebAuthn 构建 WebAuthn 实例（RP 参数取自 HOST 配置）
func NewPasskeyWebAuthn() (*webauthn.WebAuthn, error) {
	rp := PasskeyRPFromHost()
	return webauthn.New(&webauthn.Config{
		RPDisplayName: rp.RPName,
		RPID:          rp.RPID,
		RPOrigins:     []string{rp.RPOrigin},
	})
}

// ---------- 凭据持久化 ----------

func passkeyIDKey(rawID []byte) string {
	return base64.RawURLEncoding.EncodeToString(rawID)
}

// SavePasskeyCredential 保存/更新一条 WebAuthn 凭据
func SavePasskeyCredential(userID uint, name string, cred *webauthn.Credential) error {
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	idKey := passkeyIDKey(cred.ID)
	var existing models.PasskeyCredential
	if err := database.DB.Where("user_id = ? AND credential_id = ?", userID, idKey).First(&existing).Error; err == nil {
		return database.DB.Model(&existing).Updates(map[string]interface{}{
			"credential_json": string(raw),
			"last_used_at":    existing.LastUsedAt,
		}).Error
	}
	if name == "" {
		name = "Passkey"
	}
	row := models.PasskeyCredential{
		UserID:         userID,
		Name:           name,
		CredentialID:   idKey,
		CredentialJSON: string(raw),
	}
	return database.DB.Create(&row).Error
}

// ListPasskeyCredentials 列出用户凭据
func ListPasskeyCredentials(userID uint) ([]models.PasskeyCredential, error) {
	var rows []models.PasskeyCredential
	err := database.DB.Where("user_id = ?", userID).Order("id asc").Find(&rows).Error
	return rows, err
}

// LoadPasskeyUser 加载用户及其 WebAuthn 凭据
func LoadPasskeyUser(userID uint) (*PasskeyUser, error) {
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return nil, err
	}
	rows, err := ListPasskeyCredentials(userID)
	if err != nil {
		return nil, err
	}
	creds := make([]webauthn.Credential, 0, len(rows))
	for _, r := range rows {
		var c webauthn.Credential
		if err := json.Unmarshal([]byte(r.CredentialJSON), &c); err == nil {
			creds = append(creds, c)
		}
	}
	return &PasskeyUser{User: user, Credentials: creds}, nil
}

// FindPasskeyUserByRawID 按凭据 rawID（base64url 解码后）查找用户（登录 finish 映射）
func FindPasskeyUserByRawID(rawID []byte) (*PasskeyUser, error) {
	idKey := passkeyIDKey(rawID)
	var row models.PasskeyCredential
	if err := database.DB.Where("credential_id = ?", idKey).First(&row).Error; err != nil {
		return nil, fmt.Errorf("凭据不存在")
	}
	return LoadPasskeyUser(row.UserID)
}

// UpdatePasskeyLastUsed 更新凭据使用时间
func UpdatePasskeyLastUsed(rawID []byte) {
	idKey := passkeyIDKey(rawID)
	database.DB.Model(&models.PasskeyCredential{}).Where("credential_id = ?", idKey).
		Update("last_used_at", time.Now())
}

// ---------- WebAuthn 会话（内存，5 分钟） ----------

// PasskeySessionKind 会话类型
const (
	PasskeySessionRegister = "register"
	PasskeySessionLogin    = "login"
)

// PasskeySession 一次 WebAuthn 仪式的暂存数据
type PasskeySession struct {
	Kind      string
	UserID    uint
	Data      *webauthn.SessionData
	CreatedAt time.Time
}

var passkeySessionStore = struct {
	sync.RWMutex
	m map[string]PasskeySession
}{m: map[string]PasskeySession{}}

const passkeySessionTTL = 5 * time.Minute

func startPasskeySessionJanitor() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			passkeySessionStore.Lock()
			for s, sess := range passkeySessionStore.m {
				if time.Since(sess.CreatedAt) > passkeySessionTTL {
					delete(passkeySessionStore.m, s)
				}
			}
			passkeySessionStore.Unlock()
		}
	}()
}

func init() {
	startPasskeySessionJanitor()
}

// StorePasskeySession 保存会话并返回 ID
func StorePasskeySession(kind string, userID uint, data *webauthn.SessionData) string {
	id := utils.RandomString(32)
	passkeySessionStore.Lock()
	passkeySessionStore.m[id] = PasskeySession{
		Kind:      kind,
		UserID:    userID,
		Data:      data,
		CreatedAt: time.Now(),
	}
	passkeySessionStore.Unlock()
	return id
}

// GetPasskeySession 取出会话（一次性）
func GetPasskeySession(id string) (PasskeySession, bool) {
	passkeySessionStore.Lock()
	defer passkeySessionStore.Unlock()
	s, ok := passkeySessionStore.m[id]
	if !ok {
		return PasskeySession{}, false
	}
	delete(passkeySessionStore.m, id)
	if time.Since(s.CreatedAt) > passkeySessionTTL {
		return PasskeySession{}, false
	}
	return s, true
}
