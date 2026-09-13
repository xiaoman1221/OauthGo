package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/services"
)

// perform 登录测试用户并返回会话 Cookie（模拟控制台登录建立 OIDC 会话）
func loginSessionCookie(t *testing.T, username, password string) string {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login",
		strings.NewReader(fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)))
	req.Header.Set("Content-Type", "application/json")
	testEngine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("登录失败: %d %s", w.Code, w.Body.String())
	}
	for _, ck := range w.Result().Cookies() {
		if ck.Name == services.SessionCookieName && ck.Value != "" {
			return ck.Value
		}
	}
	t.Fatalf("登录响应未下发会话 Cookie")
	return ""
}

// doGetWithCookie 携带会话 Cookie 的 GET
func doGetWithCookie(t *testing.T, rawURL, cookie string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	req.AddCookie(&http.Cookie{Name: services.SessionCookieName, Value: cookie})
	testEngine.ServeHTTP(w, req)
	return w, w.Body.String()
}

// TestOIDCSessionFlow 覆盖「记录当前登录用户」的完整链路：
// 控制台登录建立会话 → 授权页一键继续 → consent 发码 → 切换账号 → 退出同步 → 会话管理
func TestOIDCSessionFlow(t *testing.T) {
	user, jwtToken := testUser(t)
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"

	// 1. 控制台登录 → 下发会话 Cookie
	sessionCookie := loginSessionCookie(t, user.Username, testUserPassword)
	if len(sessionCookie) != 64 {
		t.Fatalf("会话 Cookie 长度异常: %d", len(sessionCookie))
	}

	newCtx := func(state string) string {
		return services.CreateOAuthCtx(services.OAuthCtx{
			ClientID:    app.AppID,
			RedirectURI: cb,
			Scope:       "openid profile",
			State:       state,
		})
	}

	// 2. 有会话的授权页：展示「一键继续」而非登录表单
	ctxID := newCtx("s1")
	w, body := doGetWithCookie(t, fmt.Sprintf(
		"/api/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid%%20profile&state=s1",
		app.AppID, url.QueryEscape(cb)), sessionCookie)
	if w.Code != 200 || !strings.Contains(body, "授权并继续") || strings.Contains(body, "pwdForm") {
		t.Fatalf("有会话应展示一键继续: %d", w.Code)
	}
	if !strings.Contains(body, "prompt=login") {
		t.Fatalf("应包含切换账号入口")
	}

	// 3. 一键确认 → 302 回 redirect_uri 携带 code + state
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/oauth2/consent",
		strings.NewReader("ctx_id="+ctxID))
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req3.AddCookie(&http.Cookie{Name: services.SessionCookieName, Value: sessionCookie})
	testEngine.ServeHTTP(w3, req3)
	if w3.Code != http.StatusFound {
		t.Fatalf("consent 应 302: %d %s", w3.Code, w3.Body.String())
	}
	loc := w3.Header().Get("Location")
	if !strings.HasPrefix(loc, cb+"?code=") || !strings.Contains(loc, "state=s1") {
		t.Fatalf("consent 回跳异常: %s", loc)
	}

	// 4. 无会话（或会话无效）时 consent 应拒绝
	w4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodPost, "/api/oauth2/consent",
		strings.NewReader("ctx_id="+newCtx("s2")))
	req4.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	testEngine.ServeHTTP(w4, req4)
	if w4.Code != http.StatusBadRequest {
		t.Fatalf("无会话 consent 应 400: %d", w4.Code)
	}

	// 5. prompt=login：有会话也强制展示登录表单（切换账号）
	w5, body5 := doGetWithCookie(t, fmt.Sprintf(
		"/api/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=s3&prompt=login",
		app.AppID, url.QueryEscape(cb)), sessionCookie)
	if w5.Code != 200 || !strings.Contains(body5, "pwdForm") || strings.Contains(body5, "授权并继续") {
		t.Fatalf("prompt=login 应展示登录表单: %d", w5.Code)
	}

	// 6. 在线会话列表：包含当前会话
	_, m6 := doAuthedJSON(t, http.MethodGet, "/api/auth/sessions", "", jwtToken)
	if int(m6["code"].(float64)) != 0 {
		t.Fatalf("sessions 查询失败: %v", m6)
	}
	list, _ := m6["data"].(map[string]interface{})["list"].([]interface{})
	if len(list) == 0 {
		t.Fatalf("应至少有一条会话")
	}
	found := false
	var someID string
	for _, item := range list {
		s := item.(map[string]interface{})
		if s["id"] == sessionCookie {
			found = true
		}
		someID, _ = s["id"].(string)
	}
	if !found {
		t.Fatalf("会话列表应包含当前会话")
	}

	// 7. 吊销会话后立即失效
	_, m7 := doAuthedJSON(t, http.MethodDelete, "/api/auth/sessions/"+someID, "", jwtToken)
	if int(m7["code"].(float64)) != 0 {
		t.Fatalf("吊销失败: %v", m7)
	}
	w8, body8 := doGetWithCookie(t, fmt.Sprintf(
		"/api/oauth2/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=s4",
		app.AppID, url.QueryEscape(cb)), sessionCookie)
	if w8.Code != 200 || !strings.Contains(body8, "pwdForm") {
		t.Fatalf("吊销后应回退登录表单: %d", w8.Code)
	}

	// 8. 退出登录清除会话
	sessionCookie2 := loginSessionCookie(t, user.Username, testUserPassword)
	w9 := httptest.NewRecorder()
	req9 := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req9.AddCookie(&http.Cookie{Name: services.SessionCookieName, Value: sessionCookie2})
	testEngine.ServeHTTP(w9, req9)
	if w9.Code != http.StatusOK {
		t.Fatalf("logout 失败: %d", w9.Code)
	}
	var count int64
	database.DB.Model(&models.UserSession{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 0 {
		t.Fatalf("退出后会话应被删除，剩余 %d", count)
	}

	// 10. 修改密码吊销全部会话
	loginSessionCookie(t, user.Username, testUserPassword)
	if err := services.ChangePassword(user.ID, testUserPassword+"x"); err != nil {
		t.Fatalf("改密失败: %v", err)
	}
	var count2 int64
	database.DB.Model(&models.UserSession{}).Where("user_id = ?", user.ID).Count(&count2)
	if count2 != 0 {
		t.Fatalf("改密后会话应全部吊销，剩余 %d", count2)
	}
}
