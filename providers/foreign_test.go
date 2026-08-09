package providers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
)

func TestGoogleAuthURL(t *testing.T) {
	p := &GoogleProvider{cfg: Config{ClientID: "gid", RedirectURL: "https://x.com/cb"}}
	u := p.GetAuthURL("abc")
	for _, want := range []string{"accounts.google.com", "client_id=gid", "redirect_uri=https%3A%2F%2Fx.com%2Fcb", "scope=openid+email+profile", "state=abc"} {
		if !strings.Contains(u, want) {
			t.Fatalf("授权地址缺少 %q: %s", want, u)
		}
	}
}

func TestGitHubAuthURL(t *testing.T) {
	p := &GitHubProvider{cfg: Config{ClientID: "ghid", RedirectURL: "https://x.com/cb"}}
	u := p.GetAuthURL("s1")
	if !strings.Contains(u, "github.com/login/oauth/authorize") || !strings.Contains(u, "state=s1") {
		t.Fatalf("GitHub 授权地址异常: %s", u)
	}
}

func TestAppleClientSecret(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))

	cfg := Config{
		ClientID: "com.example.login",
		Extra: map[string]interface{}{
			"team_id":           "TEAM12345",
			"key_id":            "KID789",
			"client_secret_key": pemStr,
		},
	}
	p := &AppleProvider{cfg: cfg}

	secret, err := p.clientSecret()
	if err != nil {
		t.Fatalf("生成 client_secret 失败: %v", err)
	}

	parts := strings.Split(secret, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT 结构异常: %s", secret)
	}

	var header map[string]string
	if err := json.Unmarshal(mustB64(parts[0]), &header); err != nil {
		t.Fatal(err)
	}
	if header["alg"] != "ES256" || header["kid"] != "KID789" {
		t.Fatalf("JWT header 异常: %v", header)
	}

	claims, err := decodeIDTokenClaims(secret)
	if err != nil {
		t.Fatal(err)
	}
	if claims["iss"] != "TEAM12345" || claims["sub"] != "com.example.login" || claims["aud"] != "https://appleid.apple.com" {
		t.Fatalf("JWT payload 异常: %v", claims)
	}
	if exp, ok := claims["exp"].(float64); !ok || exp <= 0 {
		t.Fatalf("JWT 缺少有效 exp: %v", claims)
	}
}

func TestAppleClientSecretMissingConfig(t *testing.T) {
	p := &AppleProvider{cfg: Config{ClientID: "x", Extra: map[string]interface{}{}}}
	if _, err := p.clientSecret(); err == nil {
		t.Fatal("缺少 Team ID / Key ID / 私钥时应报错")
	}
}

func mustB64(s string) []byte {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
