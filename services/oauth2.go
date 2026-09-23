package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"OauthGo/config"
	"OauthGo/models"
	"OauthGo/utils"
)

// OAuthSigningKeySetting 平台 OAuth2/OIDC id_token 签名私钥（RS256）对应的系统设置项
const OAuthSigningKeySetting = "oauth_jwt_private_key"

// OAuthKeyID id_token / JWKS 使用的密钥 ID
const OAuthKeyID = "oauthgo-rsa-1"

// OAuthAccessTokenTTL 访问令牌有效期
const OAuthAccessTokenTTL = 2 * time.Hour

// OAuthRefreshTokenTTL 刷新令牌有效期
const OAuthRefreshTokenTTL = 30 * 24 * time.Hour

// OAuthCodeTTL 授权码有效期
const OAuthCodeTTL = 10 * time.Minute

// OAuthCtx 一次 OAuth2/OIDC authorization code 流程的授权上下文。
// 目标站点跳转 /authorize 后，由 state（即 ID）定位；渠道授权完成后凭
// state 回到回调，随后据此签发授权码并 302 回目标站点。
type OAuthCtx struct {
	ID            string // 授权上下文 ID，同时作为回调 state
	ClientID      string // app.AppID
	RedirectURI   string
	Scope         string
	State         string // 客户端传入的原生 state（原样带回）
	Nonce         string
	PKCEChallenge string // code_challenge（可选）
	PKCEMethod    string // S256 / plain
	Provider      string // 用户选择的渠道（wechat/qq/oauth2/...）
	CreatedAt     time.Time
}

var oauthCtxStore = struct {
	sync.RWMutex
	m map[string]OAuthCtx
}{m: map[string]OAuthCtx{}}

const oauthCtxTTL = 10 * time.Minute

func startOAuthCtxJanitor() {
	go func() {
		ticker := time.NewTicker(oauthCtxTTL)
		defer ticker.Stop()
		for range ticker.C {
			oauthCtxStore.Lock()
			for s, ctx := range oauthCtxStore.m {
				if time.Since(ctx.CreatedAt) > oauthCtxTTL {
					delete(oauthCtxStore.m, s)
				}
			}
			oauthCtxStore.Unlock()
		}
	}()
}

func init() {
	startOAuthCtxJanitor()
}

// CreateOAuthCtx 创建 OAuth2 授权上下文并返回其 ID
func CreateOAuthCtx(ctx OAuthCtx) string {
	if ctx.ID == "" {
		ctx.ID = utils.RandomString(32)
	}
	ctx.CreatedAt = time.Now()
	oauthCtxStore.Lock()
	oauthCtxStore.m[ctx.ID] = ctx
	oauthCtxStore.Unlock()
	return ctx.ID
}

// ResolveOAuthCtx 按 ID 查找并消费授权上下文（一次性）
func ResolveOAuthCtx(id string) (OAuthCtx, bool) {
	oauthCtxStore.Lock()
	defer oauthCtxStore.Unlock()
	ctx, ok := oauthCtxStore.m[id]
	if !ok {
		return OAuthCtx{}, false
	}
	delete(oauthCtxStore.m, id)
	if time.Since(ctx.CreatedAt) > oauthCtxTTL {
		return OAuthCtx{}, false
	}
	return ctx, true
}

// oauthKeyCache 已加载的签名密钥缓存
var oauthKeyCache struct {
	sync.RWMutex
	key *rsa.PrivateKey
}

// EnsureOAuthSigningKey 获取平台 OIDC 签名私钥；不存在则生成并持久化到系统设置。
func EnsureOAuthSigningKey() (*rsa.PrivateKey, error) {
	oauthKeyCache.RLock()
	if oauthKeyCache.key != nil {
		k := oauthKeyCache.key
		oauthKeyCache.RUnlock()
		return k, nil
	}
	oauthKeyCache.RUnlock()

	oauthKeyCache.Lock()
	defer oauthKeyCache.Unlock()
	if oauthKeyCache.key != nil {
		return oauthKeyCache.key, nil
	}

	// 尝试从设置加载
	if raw := GetSetting(OAuthSigningKeySetting, ""); raw != "" {
		key, err := parseRSAPrivateKeyPEM(raw)
		if err == nil {
			oauthKeyCache.key = key
			return key, nil
		}
		// 私钥损坏则重新生成
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("生成 OIDC 签名密钥失败: %w", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	SetSetting(OAuthSigningKeySetting, pemStr)
	oauthKeyCache.key = key
	return key, nil
}

// parseRSAPrivateKeyPEM 解析 PKCS1/PKCS8 格式 RSA 私钥 PEM
func parseRSAPrivateKeyPEM(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("无效的 RSA 私钥 PEM")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
	}
	return nil, fmt.Errorf("无法解析 RSA 私钥")
}

// OAuthIssuer 返回本平台 OIDC issuer（= HOST）
func OAuthIssuer() string {
	host := strings.TrimSuffix(config.AppConfig.Host, "/")
	if host == "" {
		return "http://localhost:8080"
	}
	return host
}

// oauthRedirectKey 归一化回调地址为「scheme://host/path」，丢弃 query 与 fragment。
// 部分接入方（如 WPS 企业 SSO）会在回跳地址上追加每次请求都不同的 oauth_state / state，
// 逐字符比对必然失败；授权码的落点仍由白名单里的 host+path 决定，忽略 query 不放宽落点。
// 非法 / 相对地址返回空串，调用方应据此拒绝。
func oauthRedirectKey(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host) + u.Path
}

// SameOAuthRedirect 判断两个回调地址是否指向同一落点（scheme + host + path，忽略 query / fragment）
func SameOAuthRedirect(a, b string) bool {
	key := oauthRedirectKey(a)
	return key != "" && key == oauthRedirectKey(b)
}

// ValidOAuthRedirectURI 校验 redirect_uri 是否在应用 OAuth2 回调白名单内。
// 按 scheme + host + path 精确匹配（忽略 query，见 oauthRedirectKey）：未配置 redirect_uris 时
// 一律拒绝（不回退到域名白名单），避免开放重定向被用于授权码劫持。
func ValidOAuthRedirectURI(app *models.App, redirectURI string) bool {
	if redirectURI == "" {
		return false
	}
	var uris []string
	if err := json.Unmarshal([]byte(app.RedirectURIs), &uris); err != nil {
		return false
	}
	key := oauthRedirectKey(redirectURI)
	if key == "" {
		return false
	}
	for _, u := range uris {
		if oauthRedirectKey(u) == key {
			return true
		}
	}
	return false
}
