package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"OauthGo/config"
	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/providers"
	"OauthGo/router"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

var testEngine *gin.Engine

func TestMain(m *testing.M) {
	db := filepath.Join(os.TempDir(), "oauthgo-aggregate-test.db")
	os.Remove(db)
	os.Setenv("DB_PATH", db)
	os.Setenv("PORT", "18081")
	os.Setenv("HOST", "http://localhost:18081")
	os.Setenv("JWT_KEY", "test")
	config.Load()
	database.Init()
	services.InitSettings()
	testEngine = router.Setup()
	code := m.Run()
	os.Remove(db)
	os.Exit(code)
}

const testTarget = "https://target.example.com/callback"

func seedApp(t *testing.T) models.App {
	t.Helper()
	app := models.App{
		Name:     "测试站点",
		Platform: "web",
		AppID:    "t" + utils.RandomString(15),
		AppKey:   utils.RandomString(32),
		Mode:     services.ModeCompat,
		Types:    `["gitee","wechat"]`,
		Domains:  "target.example.com",
		Status:   1,
	}
	if err := database.DB.Create(&app).Error; err != nil {
		t.Fatalf("创建应用失败: %v", err)
	}
	for _, p := range []string{"gitee", "wechat"} {
		database.DB.Model(&models.Provider{}).Where("name = ?", p).Updates(map[string]interface{}{
			"enabled":       true,
			"client_id":     "test-client",
			"client_secret": "test-secret",
		})
	}
	return app
}

func doGet(t *testing.T, rawURL string) (*httptest.ResponseRecorder, string) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	testEngine.ServeHTTP(w, req)
	return w, w.Body.String()
}

func doPostJSON(t *testing.T, rawURL, body string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, rawURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	testEngine.ServeHTTP(w, req)
	var m map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w.Code, m
}

func TestRainbowConnect(t *testing.T) {
	app := seedApp(t)
	appID, appKey := app.AppID, app.AppKey

	t.Run("login gitee", func(t *testing.T) {
		_, body := doGet(t, fmt.Sprintf("/api/connect.php?act=login&appid=%s&appkey=%s&type=gitee&redirect_uri=%s/abc", appID, appKey, testTarget))
		if !strings.Contains(body, `"code":0`) {
			t.Fatalf("返回异常: %s", body)
		}
		if !strings.Contains(body, "gitee.com/oauth/authorize") || !strings.Contains(body, "state=") {
			t.Fatalf("url 不正确: %s", body)
		}
	})

	t.Run("login rainbow type wx -> wechat", func(t *testing.T) {
		_, body := doGet(t, fmt.Sprintf("/api/connect.php?act=login&appid=%s&appkey=%s&type=wx&redirect_uri=%s", appID, appKey, testTarget))
		if !strings.Contains(body, `"code":0`) || !strings.Contains(body, "open.weixin.qq.com") {
			t.Fatalf("返回异常: %s", body)
		}
	})

	t.Run("login unsupported type", func(t *testing.T) {
		_, body := doGet(t, fmt.Sprintf("/api/connect.php?act=login&appid=%s&appkey=%s&type=baidu&redirect_uri=%s", appID, appKey, testTarget))
		if !strings.Contains(body, "未开启此登录类型") {
			t.Fatalf("应拒绝未开启类型: %s", body)
		}
	})

	t.Run("login redirect not in whitelist", func(t *testing.T) {
		_, body := doGet(t, fmt.Sprintf("/api/connect.php?act=login&appid=%s&appkey=%s&type=gitee&redirect_uri=%s", appID, appKey, "https://evil.example.com/x"))
		if !strings.Contains(body, "回跳地址不在允许范围内") {
			t.Fatalf("应拒绝非法回跳: %s", body)
		}
	})

	t.Run("login wrong appkey", func(t *testing.T) {
		_, body := doGet(t, fmt.Sprintf("/api/connect.php?act=login&appid=%s&appkey=bad&type=gitee&redirect_uri=%s", appID, testTarget))
		if !strings.Contains(body, "appkey 校验失败") {
			t.Fatalf("应拒绝错误 appkey: %s", body)
		}
	})

	t.Run("mode rest blocks rainbow", func(t *testing.T) {
		database.DB.Model(&models.App{}).Where("id = ?", app.ID).Update("mode", services.ModeREST)
		_, body := doGet(t, fmt.Sprintf("/api/connect.php?act=login&appid=%s&appkey=%s&type=gitee&redirect_uri=%s", appID, appKey, testTarget))
		if !strings.Contains(body, "未开启此协议") {
			t.Fatalf("rest 模式应拒绝彩虹协议: %s", body)
		}
		database.DB.Model(&models.App{}).Where("id = ?", app.ID).Update("mode", services.ModeCompat)
	})

	t.Run("root path and POST method", func(t *testing.T) {
		// 根路径 /connect.php（兼容彩虹官方调用约定）
		_, body := doGet(t, fmt.Sprintf("/connect.php?act=login&appid=%s&appkey=%s&type=gitee&redirect_uri=%s", appID, appKey, testTarget))
		if !strings.Contains(body, `"code":0`) {
			t.Fatalf("根路径返回异常: %s", body)
		}

		// POST + 查询串参数（WordPress 彩虹插件调用方式）
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost,
			fmt.Sprintf("/connect.php?act=login&appid=%s&appkey=%s&type=gitee&redirect_uri=%s", appID, appKey, testTarget), nil)
		testEngine.ServeHTTP(w, req)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"code":0`) {
			t.Fatalf("POST 返回异常: %d %s", w.Code, w.Body.String())
		}

		// POST + 表单参数（application/x-www-form-urlencoded）
		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodPost, "/connect.php", strings.NewReader(
			fmt.Sprintf("act=login&appid=%s&appkey=%s&type=gitee&redirect_uri=%s", appID, appKey, testTarget)))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		testEngine.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK || !strings.Contains(w2.Body.String(), `"code":0`) {
			t.Fatalf("POST 表单返回异常: %d %s", w2.Code, w2.Body.String())
		}
	})

	t.Run("callback exchange one-time", func(t *testing.T) {
		record, err := services.IssueLoginCode(appID, "gitee", "gitee", &providers.UserInfo{
			OpenID: "12345", Nickname: "小明", Avatar: "https://a/1.png",
		}, "1.2.3.4")
		if err != nil {
			t.Fatalf("签发 code 失败: %v", err)
		}
		_, body := doGet(t, fmt.Sprintf("/api/connect.php?act=callback&appid=%s&appkey=%s&type=gitee&code=%s", appID, appKey, record.Code))
		for _, want := range []string{`"code":0`, `"social_uid":"12345"`, `"nickname":"小明"`, `"faceimg":"https://a/1.png"`} {
			if !strings.Contains(body, want) {
				t.Fatalf("响应缺少 %s: %s", want, body)
			}
		}
		_, body2 := doGet(t, fmt.Sprintf("/api/connect.php?act=callback&appid=%s&appkey=%s&type=gitee&code=%s", appID, appKey, record.Code))
		if !strings.Contains(body2, `"code":2`) {
			t.Fatalf("重复使用 code 应失败: %s", body2)
		}
	})

	t.Run("query by social_uid", func(t *testing.T) {
		database.DB.Create(&models.LoginRecord{
			AppID: app.ID, AppName: app.Name, OpenID: "54321", Nickname: "小红",
			Avatar: "https://a/2.png", Platform: "gitee", Status: 1,
		})
		_, body := doGet(t, fmt.Sprintf("/api/connect.php?act=query&appid=%s&appkey=%s&type=gitee&social_uid=54321", appID, appKey))
		for _, want := range []string{`"code":0`, `"social_uid":"54321"`, `"nickname":"小红"`} {
			if !strings.Contains(body, want) {
				t.Fatalf("响应缺少 %s: %s", want, body)
			}
		}
	})
}

func TestRESTAPI(t *testing.T) {
	app := seedApp(t)
	appID, appKey := app.AppID, app.AppKey

	t.Run("login", func(t *testing.T) {
		_, m := doPostJSON(t, "/api/v1/oauth/login", fmt.Sprintf(`{"appid":"%s","appkey":"%s","type":"gitee","redirect_uri":"%s/x"}`, appID, appKey, testTarget))
		if int(m["code"].(float64)) != 0 {
			t.Fatalf("login 失败: %v", m)
		}
		data := m["data"].(map[string]interface{})
		if !strings.Contains(data["url"].(string), "gitee.com/oauth/authorize") {
			t.Fatalf("url 不正确: %v", data)
		}
	})

	t.Run("userinfo wrong sign", func(t *testing.T) {
		_, m := doPostJSON(t, "/api/v1/oauth/userinfo", fmt.Sprintf(`{"appid":"%s","code":"X","type":"gitee","sign":"bad"}`, appID))
		if !strings.Contains(m["message"].(string), "签名校验失败") {
			t.Fatalf("应拒绝错误签名: %v", m)
		}
	})

	t.Run("userinfo ok", func(t *testing.T) {
		record, err := services.IssueLoginCode(appID, "gitee", "gitee", &providers.UserInfo{
			OpenID: "88888", Nickname: "老王", Avatar: "https://a/3.png",
			Extra: map[string]interface{}{"gender": "男"},
		}, "9.9.9.9")
		if err != nil {
			t.Fatalf("签发 code 失败: %v", err)
		}
		sign := services.ComputeSign(map[string]string{"appid": appID, "code": record.Code, "type": "gitee"}, appKey)
		_, m := doPostJSON(t, "/api/v1/oauth/userinfo", fmt.Sprintf(`{"appid":"%s","code":"%s","type":"gitee","sign":"%s"}`, appID, record.Code, sign))
		if int(m["code"].(float64)) != 0 {
			t.Fatalf("userinfo 失败: %v", m)
		}
		data := m["data"].(map[string]interface{})
		if data["openid"] != "88888" || data["nickname"] != "老王" || data["gender"] != "男" {
			t.Fatalf("用户信息不正确: %v", data)
		}
	})

	t.Run("query ok", func(t *testing.T) {
		database.DB.Create(&models.LoginRecord{
			AppID: app.ID, AppName: app.Name, OpenID: "99999", Nickname: "小红",
			Avatar: "https://a/9.png", Platform: "gitee", Status: 1,
		})
		sign := services.ComputeSign(map[string]string{"appid": appID, "type": "gitee", "social_uid": "99999"}, appKey)
		_, m := doPostJSON(t, "/api/v1/oauth/query", fmt.Sprintf(`{"appid":"%s","type":"gitee","social_uid":"99999","sign":"%s"}`, appID, sign))
		if int(m["code"].(float64)) != 0 {
			t.Fatalf("query 失败: %v", m)
		}
		data := m["data"].(map[string]interface{})
		if data["openid"] != "99999" || data["nickname"] != "小红" {
			t.Fatalf("query 结果不正确: %v", data)
		}
	})

	t.Run("sign algorithm matches docs", func(t *testing.T) {
		got := services.ComputeSign(map[string]string{"appid": "a", "code": "b", "type": "qq"}, "abc123")
		want := utils.MD5("appid=a&code=b&type=qq&key=abc123")
		if got != want {
			t.Fatalf("签名算法不符: got=%s want=%s", got, want)
		}
	})
}

func TestAppCallbackRedirect(t *testing.T) {
	app := seedApp(t)
	state := services.CreateAppSession(app.AppID, "gitee", "gitee", testTarget+"/landing")
	// gitee 凭据为测试值，GetUserInfo 必然失败，应 302 回跳并携带 code=0
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/oauth/gitee/callback?state=%s&code=fake", state), nil)
	testEngine.ServeHTTP(w, req)
	if w.Code != 302 {
		t.Fatalf("应 302 跳转: %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, testTarget+"/landing") {
		t.Fatalf("回跳地址错误: %s", loc)
	}
	if !strings.Contains(loc, "code=0") {
		t.Fatalf("应携带 code=0: %s", loc)
	}
}

func testUser(t *testing.T) (models.User, string) {
	t.Helper()
	username := "u" + utils.RandomString(8)
	hash, _ := utils.HashPassword("secret123")
	user := models.User{
		Username: username, Nickname: username,
		Password: hash, PasswordSet: true, Role: "user",
	}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("创建测试用户失败: %v", err)
	}
	token, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		t.Fatalf("生成 token 失败: %v", err)
	}
	return user, token
}

func doAuthedJSON(t *testing.T, method, rawURL, body, token string) (int, map[string]interface{}) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, rawURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	testEngine.ServeHTTP(w, req)
	var m map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w.Code, m
}

func seedBindCode(account, code string) {
	database.DB.Create(&models.VerificationCode{
		Scope: "bind", Account: account, Code: code,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	})
}

func TestUserCenter(t *testing.T) {
	database.DB.Model(&models.Provider{}).Where("name = ?", "gitee").Updates(map[string]interface{}{
		"enabled": true, "client_id": "test-client", "client_secret": "test-secret",
	})
	database.DB.Model(&models.Provider{}).Where("name = ?", "wechat").Updates(map[string]interface{}{
		"enabled": true, "client_id": "test-client", "client_secret": "test-secret",
	})
	user, token := testUser(t)

	t.Run("update profile", func(t *testing.T) {
		newUsername := "u" + utils.RandomString(6)
		code, m := doAuthedJSON(t, http.MethodPut, "/api/auth/me",
			fmt.Sprintf(`{"nickname":"小王","avatar":"https://a/av.png","username":"%s"}`, newUsername), token)
		if code != 200 || int(m["code"].(float64)) != 0 {
			t.Fatalf("更新资料失败: %d %v", code, m)
		}
		data := m["data"].(map[string]interface{})
		if data["nickname"] != "小王" || data["avatar"] != "https://a/av.png" || data["username"] != newUsername {
			t.Fatalf("资料未更新: %v", data)
		}
		_, m2 := doAuthedJSON(t, http.MethodPut, "/api/auth/me",
			`{"username":"admin"}`, token)
		if int(m2["code"].(float64)) != 400 {
			t.Fatalf("重名应失败: %v", m2)
		}
	})

	t.Run("email change requires code", func(t *testing.T) {
		mail := "u" + utils.RandomString(6) + "@test.com"
		_, m := doAuthedJSON(t, http.MethodPut, "/api/auth/me",
			fmt.Sprintf(`{"email":"%s"}`, mail), token)
		if !strings.Contains(m["message"].(string), "验证码") {
			t.Fatalf("缺少验证码应失败: %v", m)
		}
		seedBindCode(mail, "123456")
		_, m2 := doAuthedJSON(t, http.MethodPut, "/api/auth/me",
			fmt.Sprintf(`{"email":"%s","email_code":"123456"}`, mail), token)
		if int(m2["code"].(float64)) != 0 {
			t.Fatalf("带验证码更新邮箱失败: %v", m2)
		}
	})

	t.Run("change password", func(t *testing.T) {
		_, m := doAuthedJSON(t, http.MethodPut, "/api/auth/password",
			`{"old_password":"wrong","new_password":"newpass123"}`, token)
		if !strings.Contains(m["message"].(string), "原密码错误") {
			t.Fatalf("错误原密码应失败: %v", m)
		}
		_, m2 := doAuthedJSON(t, http.MethodPut, "/api/auth/password",
			`{"old_password":"secret123","new_password":"newpass123"}`, token)
		if int(m2["code"].(float64)) != 0 {
			t.Fatalf("修改密码失败: %v", m2)
		}
		var reloaded models.User
		database.DB.First(&reloaded, user.ID)
		if !utils.CheckPassword(reloaded.Password, "newpass123") {
			t.Fatalf("新密码未生效")
		}
	})

	t.Run("bindings flow", func(t *testing.T) {
		_, m := doAuthedJSON(t, http.MethodGet, "/api/auth/bindings", "", token)
		if int(m["code"].(float64)) != 0 {
			t.Fatalf("查询绑定列表失败: %v", m)
		}
		list := m["data"].([]interface{})
		var giteeItem map[string]interface{}
		for _, item := range list {
			it := item.(map[string]interface{})
			if it["name"] == "gitee" {
				giteeItem = it
			}
		}
		if giteeItem == nil || giteeItem["bound"] != false {
			t.Fatalf("gitee 应未绑定: %v", list)
		}

		_, m2 := doAuthedJSON(t, http.MethodGet, "/api/auth/bind/gitee", "", token)
		if int(m2["code"].(float64)) != 0 {
			t.Fatalf("获取绑定地址失败: %v", m2)
		}
		u, err := url.Parse(m2["data"].(map[string]interface{})["url"].(string))
		if err != nil || !strings.Contains(u.String(), "gitee.com/oauth/authorize") {
			t.Fatalf("绑定地址不正确: %v", m2)
		}
		bindState := u.Query().Get("state")
		if bindState == "" {
			t.Fatalf("绑定地址缺少 state")
		}

		// 模拟回调（gitee 测试凭据必然失败）应跳回 /user-center?bind=fail
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/oauth/gitee/callback?state=%s&code=fake", bindState), nil)
		testEngine.ServeHTTP(w, req)
		if w.Code != 302 || !strings.Contains(w.Header().Get("Location"), "/user-center?bind=fail") {
			t.Fatalf("绑定回调跳转错误: %d %s", w.Code, w.Header().Get("Location"))
		}
	})

	t.Run("unbind rules", func(t *testing.T) {
		info := &providers.UserInfo{OpenID: "uc-" + utils.RandomString(6), Nickname: "小赵"}
		if err := services.BindProviderAccount(user.ID, "gitee", info); err != nil {
			t.Fatalf("绑定失败: %v", err)
		}
		_, m := doAuthedJSON(t, http.MethodGet, "/api/auth/bindings", "", token)
		list := m["data"].([]interface{})
		for _, item := range list {
			it := item.(map[string]interface{})
			if it["name"] == "gitee" && it["bound"] != true {
				t.Fatalf("gitee 应已绑定: %v", list)
			}
		}

		// 与另一用户冲突
		other, _ := testUser(t)
		if err := services.BindProviderAccount(other.ID, "gitee", info); err == nil ||
			!strings.Contains(err.Error(), "已绑定其他用户") {
			t.Fatalf("重复绑定应报错: %v", err)
		}

		_, m2 := doAuthedJSON(t, http.MethodDelete, "/api/auth/bind/gitee", "", token)
		if int(m2["code"].(float64)) != 0 {
			t.Fatalf("解绑失败: %v", m2)
		}

		// 未设置密码的第三方用户：最后一个绑定不可解绑
		oauthUser := models.User{
			Username: "o_" + utils.RandomString(8), Nickname: "第三方用户",
			Password: "x", PasswordSet: false, Role: "user",
		}
		database.DB.Create(&oauthUser)
		services.BindProviderAccount(oauthUser.ID, "wechat", &providers.UserInfo{OpenID: "w-" + utils.RandomString(6)})
		oauthToken, _ := utils.GenerateToken(oauthUser.ID, oauthUser.Role)
		_, m3 := doAuthedJSON(t, http.MethodDelete, "/api/auth/bind/wechat", "", oauthToken)
		if !strings.Contains(m3["message"].(string), "未设置密码") {
			t.Fatalf("最后一个绑定应禁止解绑: %v", m3)
		}
		_, m4 := doAuthedJSON(t, http.MethodPut, "/api/auth/password",
			`{"new_password":"setpass123"}`, oauthToken)
		if int(m4["code"].(float64)) != 0 {
			t.Fatalf("未设密用户设置密码失败: %v", m4)
		}
		_, m5 := doAuthedJSON(t, http.MethodDelete, "/api/auth/bind/wechat", "", oauthToken)
		if int(m5["code"].(float64)) != 0 {
			t.Fatalf("设置密码后解绑失败: %v", m5)
		}
	})
}

// TestDocsRoutes 接口文档路由：/docs、OpenAPI 规范与 Swagger UI
func TestDocsRoutes(t *testing.T) {
	t.Run("文档首页", func(t *testing.T) {
		w, body := doGet(t, "/docs")
		if w.Code != http.StatusOK {
			t.Fatalf("/docs 状态码异常: %d", w.Code)
		}
		for _, kw := range []string{"彩虹聚合登录协议", "REST 风格接口", "/api/connect.php", "/api/v1/oauth/userinfo"} {
			if !strings.Contains(body, kw) {
				t.Fatalf("/docs 缺少内容: %s", kw)
			}
		}
	})

	t.Run("OpenAPI 规范", func(t *testing.T) {
		w, body := doGet(t, "/docs/openapi.yaml")
		if w.Code != http.StatusOK {
			t.Fatalf("openapi.yaml 状态码异常: %d", w.Code)
		}
		for _, kw := range []string{"openapi: 3.0.3", "/api/connect.php", "/api/v1/oauth/query", "彩虹聚合登录协议"} {
			if !strings.Contains(body, kw) {
				t.Fatalf("openapi.yaml 缺少内容: %s", kw)
			}
		}
	})

	t.Run("Swagger UI", func(t *testing.T) {
		w, body := doGet(t, "/docs/swagger")
		if w.Code != http.StatusOK {
			t.Fatalf("/docs/swagger 状态码异常: %d", w.Code)
		}
		if !strings.Contains(body, "swagger-ui") || !strings.Contains(body, "/docs/openapi.yaml") {
			t.Fatalf("Swagger UI 页面内容异常")
		}
	})
}

// TestLoginRecordAccountLabel 登录记录用户名按登录方式展示（QQ号 / OpenID 等）
func TestLoginRecordAccountLabel(t *testing.T) {
	_, token := testUser(t)
	database.DB.Create(&models.LoginRecord{
		AppID: 1, AppName: "测试站", OpenID: "qq-openid-001", Nickname: "QQ用户",
		Platform: "qq", IP: "1.2.3.4", Status: 1, UserAgent: "Mozilla/5.0",
	})
	database.DB.Create(&models.LoginRecord{
		AppID: 1, AppName: "测试站", OpenID: "wx-openid-002", Nickname: "微信用户",
		Platform: "wechat", IP: "1.2.3.4", Status: 1,
	})
	database.DB.Create(&models.LoginRecord{
		AppID: 1, AppName: "测试站", Username: "imported_user", Nickname: "导入用户",
		Platform: "qq", IP: "1.2.3.4", Status: 1,
	})

	code, m := doAuthedJSON(t, http.MethodGet,
		"/api/logins?page_size=20", "", token)
	if code != 200 || int(m["code"].(float64)) != 0 {
		t.Fatalf("获取登录记录失败: %d %v", code, m)
	}
	list := m["data"].(map[string]interface{})["list"].([]interface{})
	if len(list) < 3 {
		t.Fatalf("登录记录不足: %d", len(list))
	}

	labelOf := func(item map[string]interface{}) string {
		return item["uid_label"].(string)
	}
	valueOf := func(item map[string]interface{}) string {
		return item["uid_value"].(string)
	}
	findByOpenID := func(openID string) map[string]interface{} {
		for _, it := range list {
			item := it.(map[string]interface{})
			if item["open_id"] == openID {
				return item
			}
		}
		return nil
	}
	findByUsername := func(username string) map[string]interface{} {
		for _, it := range list {
			item := it.(map[string]interface{})
			if item["username"] == username {
				return item
			}
		}
		return nil
	}

	qq := findByOpenID("qq-openid-001")
	if qq == nil || labelOf(qq) != "QQ号" || valueOf(qq) != "qq-openid-001" {
		t.Fatalf("QQ 记录用户名展示异常: %v", qq)
	}
	wx := findByOpenID("wx-openid-002")
	if wx == nil || labelOf(wx) != "OpenID" || valueOf(wx) != "wx-openid-002" {
		t.Fatalf("微信记录用户名展示异常: %v", wx)
	}
	imp := findByUsername("imported_user")
	if imp == nil || labelOf(imp) != "用户名" || valueOf(imp) != "imported_user" {
		t.Fatalf("导入记录用户名展示异常: %v", imp)
	}
}
