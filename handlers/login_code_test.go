package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/services"
	"OauthGo/utils"
)

// TestLoginCodeExchange 覆盖一次性登录码的签发与兑换（/oauth-callback?code= 自动适配流程）
func TestLoginCodeExchange(t *testing.T) {
	user, _ := testUser(t)

	code, err := services.IssuePlatformLoginCode(user.ID)
	if err != nil {
		t.Fatalf("签发登录码失败: %v", err)
	}
	if code == "" || len(code) < 16 {
		t.Fatalf("登录码异常: %q", code)
	}

	postExchange := func(c string) (*httptest.ResponseRecorder, map[string]interface{}) {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/code-exchange", strings.NewReader(fmt.Sprintf(`{"code":%q}`, c)))
		req.Header.Set("Content-Type", "application/json")
		testEngine.ServeHTTP(w, req)
		var m map[string]interface{}
		_ = json.Unmarshal(w.Body.Bytes(), &m)
		return w, m
	}

	t.Run("exchange ok and returns usable jwt", func(t *testing.T) {
		w, m := postExchange(code)
		if w.Code != http.StatusOK || int(m["code"].(float64)) != 0 {
			t.Fatalf("兑换失败: %d %v", w.Code, m)
		}
		data, _ := m["data"].(map[string]interface{})
		token, _ := data["token"].(string)
		if token == "" {
			t.Fatalf("应返回 token: %v", m)
		}
		if data["user"] == nil {
			t.Fatalf("应返回 user: %v", m)
		}
		claims, err := utils.ParseToken(token)
		if err != nil || claims.UserID != user.ID {
			t.Fatalf("token 应为该用户的有效 JWT: %v err=%v", claims, err)
		}
		// 兑换出的 JWT 可访问需要认证的接口
		wme := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		testEngine.ServeHTTP(wme, req)
		if wme.Code != http.StatusOK {
			t.Fatalf("兑换出的 JWT 应可用: %d", wme.Code)
		}
	})

	t.Run("code is one-time", func(t *testing.T) {
		w, _ := postExchange(code)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("code 应一次性: %d", w.Code)
		}
	})

	t.Run("invalid code", func(t *testing.T) {
		w, _ := postExchange("DEADBEEFDEADBEEFDEADBEEFDEADBEEF")
		if w.Code != http.StatusBadRequest {
			t.Fatalf("无效 code 应 400: %d", w.Code)
		}
		w2, _ := postExchange("")
		if w2.Code != http.StatusBadRequest {
			t.Fatalf("空 code 应 400: %d", w2.Code)
		}
	})

	t.Run("expired code", func(t *testing.T) {
		expired := models.PlatformLoginCode{
			Code:      strings.ToUpper(utils.RandomString(32)),
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(-services.PlatformLoginCodeTTL),
		}
		if err := database.DB.Create(&expired).Error; err != nil {
			t.Fatalf("插入过期登录码失败: %v", err)
		}
		w, _ := postExchange(expired.Code)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("过期 code 应 400: %d", w.Code)
		}
	})
}
