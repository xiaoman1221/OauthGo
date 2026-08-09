package providers

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AppleProvider Apple（Sign in with Apple）登录
// 客户端密钥（client_secret）为 ES256 签名的 JWT，由 Team ID、Key ID 与私钥在服务端实时生成
type AppleProvider struct {
	cfg    Config
	client *http.Client
}

func (p *AppleProvider) Name() string { return "apple" }

func (p *AppleProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://appleid.apple.com/auth/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&response_mode=form_post&state=%s",
		url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.RedirectURL),
		url.QueryEscape("name email"), url.QueryEscape(state))
}

func (p *AppleProvider) GetUserInfo(code string) (*UserInfo, error) {
	clientSecret, err := p.clientSecret()
	if err != nil {
		return nil, err
	}

	var token struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := postFormClient(p.client, "https://appleid.apple.com/auth/token", nil, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {p.cfg.RedirectURL},
	}, &token); err != nil {
		return nil, err
	}

	claims, err := decodeIDTokenClaims(token.IDToken)
	if err != nil {
		return nil, err
	}
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)

	nickname := "Apple 用户"
	if email != "" {
		nickname = strings.Split(email, "@")[0]
	}

	return &UserInfo{
		OpenID:      sub,
		Nickname:    nickname,
		Email:       email,
		AccessToken: token.AccessToken,
	}, nil
}

// clientSecret 生成 Apple OAuth client_secret（ES256 JWT，有效期 1 小时）
func (p *AppleProvider) clientSecret() (string, error) {
	teamID := strings.TrimSpace(p.cfg.ExtraString("team_id"))
	keyID := strings.TrimSpace(p.cfg.ExtraString("key_id"))
	privKey := strings.TrimSpace(p.cfg.ExtraString("client_secret_key"))
	if teamID == "" || keyID == "" || privKey == "" {
		return "", fmt.Errorf("Apple 渠道缺少 Team ID / Key ID / 私钥配置")
	}

	key, err := parseECPrivateKey(privKey)
	if err != nil {
		return "", fmt.Errorf("解析 Apple 私钥失败: %w", err)
	}

	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": keyID})
	now := time.Now().Unix()
	payload, _ := json.Marshal(map[string]interface{}{
		"iss": teamID,
		"iat": now,
		"exp": now + 3600,
		"aud": "https://appleid.apple.com",
		"sub": p.cfg.ClientID,
	})

	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)

	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		return "", err
	}

	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// parseECPrivateKey 解析 Apple 私钥（P-256 曲线，PKCS#8 或 SEC1 PEM 格式）
func parseECPrivateKey(pemStr string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("无效的 PEM 私钥")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if ecKey, ok := key.(*ecdsa.PrivateKey); ok {
			return ecKey, nil
		}
		return nil, fmt.Errorf("私钥不是 EC 密钥")
	}
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("仅支持 P-256（ES256）私钥，请确认已导入 Apple 生成的 .p8 密钥")
}

// decodeIDTokenClaims 解码 JWT 的 payload 部分（不校验签名，仅用于提取字段）
func decodeIDTokenClaims(idToken string) (map[string]interface{}, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("无效的 id_token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}
