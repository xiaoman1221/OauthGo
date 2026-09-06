package handlers_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/services"
	"OauthGo/utils"
)

// seedOAuth2App 创建开启了 OAuth2/OIDC 协议并配置了 redirect_uris 的应用
func seedOAuth2App(t *testing.T) models.App {
	t.Helper()
	user, _ := testUser(t)
	app := models.App{
		OwnerID:       user.ID,
		Name:          "OIDC 测试站点",
		Platform:      "web",
		AppID:         "t" + utils.RandomString(15),
		AppKey:        utils.RandomString(32),
		Mode:          services.ModeOAuth2,
		Types:         `["gitee","wechat"]`,
		Domains:       "target.example.com",
		RedirectURIs:  `["https://target.example.com/callback"]`,
		EnableRefresh: true,
		Status:        1,
	}
	if err := database.DB.Create(&app).Error; err != nil {
		t.Fatalf("创建 OAuth2 应用失败: %v", err)
	}
	enableTestProviders()
	return app
}

func doPostForm(t *testing.T, rawURL string, form url.Values) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testEngine.ServeHTTP(w, req)
	var m map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w, m
}

func TestOAuth2DiscoveryAndJWKS(t *testing.T) {
	w, body := doGet(t, "/api/oauth2/.well-known/openid-configuration")
	if w.Code != 200 || !strings.Contains(body, "authorization_endpoint") {
		t.Fatalf("discovery 异常: %d %s", w.Code, body)
	}
	var d map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &d)
	if !strings.Contains(d["issuer"].(string), "localhost") {
		t.Fatalf("issuer 异常: %v", d["issuer"])
	}
	if !strings.Contains(d["authorization_endpoint"].(string), "/authorize") {
		t.Fatalf("authorization_endpoint 异常: %v", d["authorization_endpoint"])
	}
	// 旧前缀别名 discovery 仍可用（向后兼容）
	walias, balias := doGet(t, "/api/oauth2/.well-known/openid-configuration")
	if walias.Code != 200 || !strings.Contains(balias, "authorization_endpoint") {
		t.Fatalf("alias discovery 异常: %d %s", walias.Code, balias)
	}
	// RFC 8414 别名：oauth-authorization-server
	w8414, _ := doGet(t, "/.well-known/oauth-authorization-server")
	if w8414.Code != 200 {
		t.Fatalf("oauth-authorization-server discovery 异常: %d", w8414.Code)
	}

	w2, body2 := doGet(t, "/api/oauth2/jwks")
	if w2.Code != 200 {
		t.Fatalf("jwks 异常: %d %s", w2.Code, body2)
	}
	var jwks struct {
		Keys []map[string]interface{} `json:"keys"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &jwks); err != nil || len(jwks.Keys) != 1 {
		t.Fatalf("jwks keys 异常: %s", body2)
	}
	k := jwks.Keys[0]
	if k["kid"] != services.OAuthKeyID || k["alg"] != "RS256" || k["n"] == "" || k["e"] == "" {
		t.Fatalf("jwks 内容异常: %v", k)
	}
}

func TestOAuth2AuthorizeConsentPage(t *testing.T) {
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"

	// 不带 type → 返回渠道选择页（HTML）
	w, body := doGet(t, fmt.Sprintf(
		"/api/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid%%20profile&state=st1",
		app.AppID, cb))
	if w.Code != 200 || !strings.Contains(body, "授权登录") {
		t.Fatalf("consent 页异常: %d %s", w.Code, body)
	}
	if !strings.Contains(body, "gitee") || !strings.Contains(body, "wechat") {
		t.Fatalf("consent 页缺少渠道: %s", body)
	}

	// 带 type=gitee → 302 到 Gitee 授权地址（state=OAuth2 ctx id）
	w2, _ := doGet(t, fmt.Sprintf(
		"/api/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=st2&type=gitee",
		app.AppID, cb))
	if w2.Code != http.StatusFound {
		t.Fatalf("应 302: %d", w2.Code)
	}
	loc := w2.Header().Get("Location")
	if !strings.Contains(loc, "gitee.com/oauth/authorize") || !strings.Contains(loc, "state=") {
		t.Fatalf("授权跳转地址异常: %s", loc)
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	ctxID := u.Query().Get("state")
	if ctxID == "" || len(ctxID) < 16 {
		t.Fatalf("state(oauth ctx) 异常: %q", ctxID)
	}

	// 回调消费该 ctx（模拟渠道拒绝/失败）：因 gitee 无法真实授权，
	// 应 302 回 redirect_uri 并携带 error
	w3, _ := doGet(t, fmt.Sprintf("/api/oauth/gitee/callback?code=bad-code&state=%s", ctxID))
	if w3.Code != http.StatusFound {
		t.Fatalf("回调应 302: %d", w3.Code)
	}
	loc3 := w3.Header().Get("Location")
	if !strings.Contains(loc3, cb) || !strings.Contains(loc3, "error=access_denied") {
		t.Fatalf("回调错误跳转异常: %s", loc3)
	}

	// ctx 已消费：再次使用应回退到普通 state 校验（state 不存在 → 400）
	w4, body4 := doGet(t, fmt.Sprintf("/api/oauth/gitee/callback?code=x&state=%s", ctxID))
	if w4.Code != http.StatusBadRequest || !strings.Contains(body4, "state 校验失败") {
		t.Fatalf("ctx 消费后应 state 校验失败: %d %s", w4.Code, body4)
	}
}

func TestOAuth2AuthorizeInvalid(t *testing.T) {
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"

	// 非法 redirect_uri（不在白名单）→ 错误页
	w, body := doGet(t, fmt.Sprintf("/api/oauth2/authorize?client_id=%s&redirect_uri=https://evil.example.com/x&state=s", app.AppID))
	if w.Code != http.StatusBadRequest || !strings.Contains(body, "授权失败") {
		t.Fatalf("非法 redirect_uri 应返回错误页: %d %s", w.Code, body)
	}

	// 错误 client → 错误页
	w2, _ := doGet(t, fmt.Sprintf("/api/oauth2/authorize?client_id=bad&redirect_uri=%s&state=s", cb))
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("非法 client 应返回错误页: %d", w2.Code)
	}

	// 不支持的 response_type
	w3, _ := doGet(t, fmt.Sprintf("/api/oauth2/authorize?response_type=token&client_id=%s&redirect_uri=%s&state=s", app.AppID, cb))
	if w3.Code != http.StatusBadRequest {
		t.Fatalf("不支持 response_type 应报错: %d", w3.Code)
	}

	// rainbow-only 应用不允许走 OAuth2
	app2 := seedApp(t) // ModeCompat
	database.DB.Model(&models.App{}).Where("id = ?", app2.ID).Update("mode", services.ModeRainbow)
	w4, _ := doGet(t, fmt.Sprintf("/api/oauth2/authorize?client_id=%s&redirect_uri=https://target.example.com/callback&state=s", app2.AppID))
	if w4.Code != http.StatusBadRequest {
		t.Fatalf("rainbow 模式应用不应允许 OAuth2: %d", w4.Code)
	}
}

func TestOAuth2TokenUserinfoFlow(t *testing.T) {
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"
	user, _ := testUser(t)

	// PKCE S256 challenge
	verifier := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEF"
	sum := sha256sum(verifier)
	challenge := base64RawURL(sum)

	code := models.OAuthCode{
		Code:          strings.ToUpper(utils.RandomString(32)),
		ClientID:      app.AppID,
		UserID:        user.ID,
		Scope:         "openid profile",
		RedirectURI:   cb,
		PKCEChallenge: challenge,
		PKCEMethod:    "S256",
		ExpiresAt:     time.Now().Add(time.Minute),
	}
	if err := database.DB.Create(&code).Error; err != nil {
		t.Fatalf("插入授权码失败: %v", err)
	}

	// 错误 verifier → invalid_grant
	wErr, mErr := doPostForm(t, "/api/oauth2/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code.Code},
		"redirect_uri":  {cb},
		"client_id":     {app.AppID},
		"client_secret": {app.AppKey},
		"code_verifier": {"wrong-verifier"},
	})
	if wErr.Code != http.StatusBadRequest || mErr["error"] != "invalid_grant" {
		t.Fatalf("错误 verifier 应 invalid_grant: %d %v", wErr.Code, mErr)
	}

	// 正确 verifier → 成功换取令牌
	w, m := doPostForm(t, "/api/oauth2/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code.Code},
		"redirect_uri":  {cb},
		"client_id":     {app.AppID},
		"client_secret": {app.AppKey},
		"code_verifier": {verifier},
	})
	if w.Code != 200 {
		t.Fatalf("token 交换失败: %d %v", w.Code, m)
	}
	accessToken, _ := m["access_token"].(string)
	refreshToken, _ := m["refresh_token"].(string)
	idToken, _ := m["id_token"].(string)
	if accessToken == "" || refreshToken == "" {
		t.Fatalf("token 交换缺字段: %v", m)
	}
	if idToken == "" || !strings.Contains(idToken, ".") {
		t.Fatalf("缺 id_token（scope 含 openid 应签发）: %v", m)
	}

	// code 一次性：再次交换 → invalid_grant
	w2, m2 := doPostForm(t, "/api/oauth2/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code.Code},
		"redirect_uri":  {cb},
		"client_id":     {app.AppID},
		"client_secret": {app.AppKey},
		"code_verifier": {verifier},
	})
	if w2.Code != http.StatusBadRequest || m2["error"] != "invalid_grant" {
		t.Fatalf("code 应一次性: %d %v", w2.Code, m2)
	}

	// userinfo
	uiW, uiBody := oauthBearerGet(t, "/api/oauth2/userinfo", accessToken)
	if uiW.Code != 200 || !strings.Contains(uiBody, fmt.Sprintf(`"sub":"%d"`, user.ID)) {
		t.Fatalf("userinfo 异常: %d %s", uiW.Code, uiBody)
	}

	// refresh_token 轮换
	w3, m3 := doPostForm(t, "/api/oauth2/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {app.AppID},
		"client_secret": {app.AppKey},
	})
	if w3.Code != 200 || m3["refresh_token"] == nil {
		t.Fatalf("refresh 轮换失败: %d %v", w3.Code, m3)
	}
	if m3["refresh_token"] == refreshToken {
		t.Fatalf("refresh token 应轮换")
	}
	newRefresh, _ := m3["refresh_token"].(string)

	// 旧 refresh 已失效
	w4, m4 := doPostForm(t, "/api/oauth2/token", url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {app.AppID},
		"client_secret": {app.AppKey},
	})
	if w4.Code != http.StatusBadRequest || m4["error"] != "invalid_grant" {
		t.Fatalf("旧 refresh 应失效: %d %v", w4.Code, m4)
	}

	// introspect：旧 access_token 仍有效
	wi, mi := doPostForm(t, "/api/oauth2/introspect", url.Values{
		"token":           {accessToken},
		"token_type_hint": {"access_token"},
		"client_id":       {app.AppID},
		"client_secret":   {app.AppKey},
	})
	if wi.Code != 200 || mi["active"] != true {
		t.Fatalf("introspect 应 active=true: %d %v", wi.Code, mi)
	}
	if fmt.Sprintf("%v", mi["sub"]) != fmt.Sprintf("%d", user.ID) {
		t.Fatalf("introspect sub 异常: %v", mi["sub"])
	}
	// introspect：伪造 token → active=false
	wi2, mi2 := doPostForm(t, "/api/oauth2/introspect", url.Values{
		"token":         {"no-such-token"},
		"client_id":     {app.AppID},
		"client_secret": {app.AppKey},
	})
	if wi2.Code != 200 || mi2["active"] != false {
		t.Fatalf("伪造 token 应 active=false: %d %v", wi2.Code, mi2)
	}

	// revoke 新 refresh
	w5, _ := doPostForm(t, "/api/oauth2/revoke", url.Values{
		"token":           {newRefresh},
		"token_type_hint": {"refresh_token"},
		"client_id":       {app.AppID},
		"client_secret":   {app.AppKey},
	})
	if w5.Code != 200 {
		t.Fatalf("revoke 失败: %d", w5.Code)
	}
}

func TestOAuth2UserinfoUnauthorized(t *testing.T) {
	w, _ := doGet(t, "/api/oauth2/userinfo")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无 token 应 401: %d", w.Code)
	}
	w2, _ := oauthBearerGet(t, "/api/oauth2/userinfo", "not-exist-token")
	if w2.Code != http.StatusUnauthorized {
		t.Fatalf("非法 token 应 401: %d", w2.Code)
	}
}

func TestOAuth2ClientAuthFail(t *testing.T) {
	app := seedOAuth2App(t)
	w, m := doPostForm(t, "/api/oauth2/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"X"},
		"client_id":     {app.AppID},
		"client_secret": {"wrong-secret"},
	})
	if w.Code != http.StatusUnauthorized || m["error"] != "invalid_client" {
		t.Fatalf("client 认证失败应 401 invalid_client: %d %v", w.Code, m)
	}
}

// ---- helpers ----

func sha256sum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

func base64RawURL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func oauthBearerGet(t *testing.T, rawURL, token string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	testEngine.ServeHTTP(w, req)
	return w, w.Body.String()
}
