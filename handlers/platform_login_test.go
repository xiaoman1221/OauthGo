package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestOAuth2PlatformLoginFlow：平台账号（密码）登录完成 OIDC 授权 → 换 token
func TestOAuth2PlatformLoginFlow(t *testing.T) {
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"
	user, _ := testUser(t)

	// 1. 发起授权（无 type → consent 页，带 ctx_id）
	w, body := doGet(t, fmt.Sprintf("/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid%%20profile&state=plogin1", app.AppID, cb))
	if w.Code != 200 {
		t.Fatalf("consent 页异常: %d", w.Code)
	}
	idx := strings.Index(body, `name="ctx_id" value="`)
	if idx < 0 {
		t.Fatalf("consent 页缺少 ctx_id: %s", body[:400])
	}
	rest := body[idx+len(`name="ctx_id" value="`):]
	end := strings.Index(rest, `"`)
	ctxID := rest[:end]
	if ctxID == "" {
		t.Fatalf("ctx_id 为空")
	}

	// 2. 平台账号密码登录 → 302 回 redirect_uri?code
	w2, _ := doPostForm(t, "/api/oauth2/platform-login", url.Values{
		"ctx_id":   {ctxID},
		"username": {user.Username},
		"password": {"secret123"},
	})
	if w2.Code != http.StatusFound {
		t.Fatalf("平台登录应 302: %d", w2.Code)
	}
	loc := w2.Header().Get("Location")
	if !strings.HasPrefix(loc, cb+"?code=") || !strings.Contains(loc, "state=plogin1") {
		t.Fatalf("回跳 URL 异常: %s", loc)
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}
	code := u.Query().Get("code")
	if code == "" {
		t.Fatal("缺少 code")
	}

	// 3. 错误密码 → 错误页
	w3, body3 := doGet(t, fmt.Sprintf("/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=plogin2", app.AppID, cb))
	if w3.Code != 200 {
		t.Fatalf("second consent 页异常: %d", w3.Code)
	}
	idx3 := strings.Index(body3, `name="ctx_id" value="`)
	ctx3 := body3[idx3+len(`name="ctx_id" value="`):]
	ctx3 = ctx3[:strings.Index(ctx3, `"`)]
	w4, _ := doPostForm(t, "/api/oauth2/platform-login", url.Values{
		"ctx_id":   {ctx3},
		"username": {user.Username},
		"password": {"wrong"},
	})
	if w4.Code != http.StatusBadRequest || !strings.Contains(w4.Body.String(), "用户名或密码错误") {
		t.Fatalf("错误密码应拒绝: %d %s", w4.Code, w4.Body.String())
	}

	// 4. code 换 token（校验 code 与平台用户绑定正确）
	w5, m5 := doPostForm(t, "/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {cb},
		"client_id":     {app.AppID},
		"client_secret": {app.AppKey},
	})
	if w5.Code != 200 {
		t.Fatalf("token 交换失败: %d %v", w5.Code, m5)
	}
	at, _ := m5["access_token"].(string)
	if at == "" {
		t.Fatalf("缺少 access_token: %v", m5)
	}
	uiW, uiBody := oauthBearerGet(t, "/userinfo", at)
	if uiW.Code != 200 || !strings.Contains(uiBody, fmt.Sprintf(`"sub":"%d"`, user.ID)) {
		t.Fatalf("userinfo 异常: %d %s", uiW.Code, uiBody)
	}
	// profile scope 应返回 username
	if !strings.Contains(uiBody, user.Username) {
		t.Fatalf("profile claims 缺少 preferred_username: %s", uiBody)
	}
}

// TestPasskeyAPIValidation：Passkey API 的鉴权与参数校验（真实 WebAuthn 需浏览器）
func TestPasskeyAPIValidation(t *testing.T) {
	user, _ := testUser(t)

	// 无凭据用户 login begin → 400 统一提示（不暴露账号状态，防枚举）
	w, _ := doGet(t, "/api/auth/passkey/login/begin?username="+user.Username)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "未注册") {
		t.Fatalf("无凭据用户 begin 应 400 统一提示: %d %s", w.Code, w.Body.String())
	}

	// 无 username
	w2, _ := doGet(t, "/api/auth/passkey/login/begin")
	if w2.Code != http.StatusBadRequest || !strings.Contains(w2.Body.String(), "用户名") {
		t.Fatalf("缺 username 应 400: %d %s", w2.Code, w2.Body.String())
	}

	// register begin 未登录 → 401（POST + 无 Authorization）
	w3, _ := doPostFormJSON(t, "/api/auth/passkey/register/begin", "{}")
	if w3.Code != http.StatusUnauthorized {
		t.Fatalf("register begin 未登录应 401: %d", w3.Code)
	}

	// login finish 缺 session_id → 400
	w4, body4 := doPostFormJSON(t, "/api/auth/passkey/login/finish", `{}`)
	if w4.Code != http.StatusBadRequest || !strings.Contains(body4, "session_id") {
		t.Fatalf("finish 缺 session_id 应 400: %d %s", w4.Code, body4)
	}
}

func doPostFormJSON(t *testing.T, rawURL, body string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, rawURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	testEngine.ServeHTTP(w, req)
	return w, w.Body.String()
}
