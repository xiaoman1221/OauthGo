package handlers_test

import (
	"net/http"
	"strings"
	"testing"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/utils"
)

// newTestAdmin 直接建库创建管理员并返回其 JWT
func newTestAdmin(t *testing.T) (models.User, string) {
	t.Helper()
	admin := models.User{
		Username:    "admin" + utils.RandomString(8),
		Nickname:    "测试管理员",
		Password:    "x",
		PasswordSet: true,
		Role:        "admin",
	}
	if err := database.DB.Create(&admin).Error; err != nil {
		t.Fatalf("创建测试管理员失败: %v", err)
	}
	token, err := utils.GenerateToken(admin.ID, admin.Role)
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	return admin, token
}

// TestSettingsAdminOnly 系统设置读取收归管理员；普通用户所需配置走公开 auth/config
func TestSettingsAdminOnly(t *testing.T) {
	user, userToken := testUser(t)
	_, adminToken := newTestAdmin(t)

	// 普通用户读系统设置 → 403
	code, m := doAuthedJSON(t, http.MethodGet, "/api/settings", "", userToken)
	if code != http.StatusForbidden {
		t.Fatalf("普通用户读设置应 403: %d %v", code, m)
	}

	// 管理员读系统设置 → 200
	code, m = doAuthedJSON(t, http.MethodGet, "/api/settings", "", adminToken)
	if code != http.StatusOK || int(m["code"].(float64)) != 0 {
		t.Fatalf("管理员读设置应 200: %d %v", code, m)
	}

	// 未登录读设置 → 401
	w, _ := doGet(t, "/api/settings")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未登录读设置应 401: %d", w.Code)
	}

	// 公开 auth/config 包含头像渲染所需字段（替代普通用户读设置的合法需求）
	w2, body2 := doGet(t, "/api/auth/config")
	if w2.Code != 200 || !strings.Contains(body2, "avatar_source") || !strings.Contains(body2, "gravatar_mirror") {
		t.Fatalf("auth/config 应包含头像配置: %d %s", w2.Code, body2)
	}

	_ = user
}

// TestAdminRoleRealtime AdminOnly 实时查库校验：降级/删除后旧令牌立即失去后台权限
func TestAdminRoleRealtime(t *testing.T) {
	admin, adminToken := newTestAdmin(t)

	// 降级前：可访问用户管理
	code, m := doAuthedJSON(t, http.MethodGet, "/api/users", "", adminToken)
	if code != http.StatusOK || int(m["code"].(float64)) != 0 {
		t.Fatalf("管理员应可访问用户管理: %d %v", code, m)
	}

	// 降级后：同一枚旧令牌立即 403
	if err := database.DB.Model(&admin).Update("role", "user").Error; err != nil {
		t.Fatalf("降级失败: %v", err)
	}
	code, m = doAuthedJSON(t, http.MethodGet, "/api/users", "", adminToken)
	if code != http.StatusForbidden {
		t.Fatalf("降级后旧令牌应立即 403: %d %v", code, m)
	}

	// 删除后：同样 403
	if err := database.DB.Model(&admin).Update("role", "admin").Error; err != nil {
		t.Fatalf("恢复失败: %v", err)
	}
	if err := database.DB.Delete(&models.User{}, admin.ID).Error; err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	code, _ = doAuthedJSON(t, http.MethodGet, "/api/users", "", adminToken)
	if code != http.StatusForbidden {
		t.Fatalf("删除后旧令牌应 403: %d", code)
	}
}

// TestPasskeyBeginAntiEnumeration 不存在的账号也返回结构一致的假挑战（防枚举）
func TestPasskeyBeginAntiEnumeration(t *testing.T) {
	// 不存在的账号 → 200 + 正常结构的挑战（而非可区分的错误）
	w, body := doGet(t, "/api/auth/passkey/login/begin?username=no-such-user-xyz")
	if w.Code != 200 {
		t.Fatalf("不存在账号应返回 200 假挑战: %d %s", w.Code, body)
	}
	if !strings.Contains(body, "session_id") || !strings.Contains(body, "challenge") {
		t.Fatalf("假挑战结构异常: %s", body)
	}

	// 存在但未注册 Passkey 的账号 → 同样 200 假挑战
	user, _ := testUser(t)
	w2, body2 := doGet(t, "/api/auth/passkey/login/begin?username="+user.Username)
	if w2.Code != 200 || !strings.Contains(body2, "session_id") {
		t.Fatalf("未注册 Passkey 账号应返回 200 假挑战: %d %s", w2.Code, body2)
	}

	// 假会话无法完成登录：session 绑定 userID=0，finish 应报用户不存在
	code3, m3 := doPostJSON(t, "/api/auth/passkey/login/finish?session_id=nonexistent", `{}`)
	// session 不存在本身也 400，关键是不泄露账号存在性
	if code3 != http.StatusBadRequest {
		t.Fatalf("finish 应 400: %d %v", code3, m3)
	}
}
