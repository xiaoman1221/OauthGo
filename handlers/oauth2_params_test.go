package handlers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/utils"
)

// 样例接入方（WPS 企业 SSO）在「应用管理」里的配置：回调白名单 + 接入示例参数。
// 这些值只是测试用来模拟某个接入方的配置输入，平台与测试逻辑都不依赖它们的字面量，
// 运行时统一从应用记录读取（见 configureApp / appConfig）。
var wpsAppConfig = struct {
	CallbackBase string
	SampleParams map[string]string
}{
	CallbackBase: "https://account.wps.cn/permit/ssoafterlogin.html",
	SampleParams: map[string]string{
		"app_id":      "FT20260923XWAIGM",
		"checkcfg":    "FT20260923XWAIGM",
		"oauth_state": "6ba797234e2f4a2d83418c3a261f4504",
		"state":       "YWN0aW9uPXZlcmlmeSZjYj1odHRwcyUzQSUyRiUyRmFjY291bnQud3BzLmNu",
	},
}

// encodeParams 把接入示例参数编码成 query 串（模拟接入方在回跳地址上追加的参数）
func encodeParams(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values.Encode()
}

// appConfig 从应用记录读回配置（= 应用管理里设置的内容）
type appConfig struct {
	RedirectURI  string
	SampleParams map[string]string
}

func readAppConfig(t *testing.T, id uint) appConfig {
	t.Helper()
	var app models.App
	if err := database.DB.First(&app, id).Error; err != nil {
		t.Fatalf("读取应用失败: %v", err)
	}
	var uris []string
	if err := json.Unmarshal([]byte(app.RedirectURIs), &uris); err != nil || len(uris) == 0 {
		t.Fatalf("应用回调白名单异常: %q (%v)", app.RedirectURIs, err)
	}
	var params map[string]string
	if err := json.Unmarshal([]byte(app.SampleParams), &params); err != nil {
		t.Fatalf("接入示例参数异常: %q (%v)", app.SampleParams, err)
	}
	return appConfig{RedirectURI: uris[0], SampleParams: params}
}

// configureApp 通过应用更新接口写入回调白名单与接入示例参数（与应用管理走同一条路径）
func configureApp(t *testing.T, app models.App, callback string, sampleParams map[string]string) {
	t.Helper()
	_, adminToken := newTestAdmin(t)
	body, err := json.Marshal(map[string]interface{}{
		"name":          app.Name,
		"platform":      app.Platform,
		"mode":          app.Mode,
		"types":         []string{"gitee", "wechat"},
		"domains":       app.Domains,
		"redirect_uris": []string{callback},
		"sample_params": sampleParams,
	})
	if err != nil {
		t.Fatal(err)
	}
	code, m := doAuthedJSON(t, http.MethodPut, fmt.Sprintf("/api/apps/%d", app.ID), string(body), adminToken)
	if code != http.StatusOK {
		t.Fatalf("更新应用配置失败: %d %v", code, m)
	}
	// 接口回显应与写入一致（应用管理页面即读取该字段渲染示例）
	data, _ := m["data"].(map[string]interface{})
	got, _ := data["sample_params"].(map[string]interface{})
	if len(got) != len(sampleParams) {
		t.Fatalf("sample_params 回显不一致: %v", data["sample_params"])
	}
	for k, v := range sampleParams {
		if got[k] != v {
			t.Fatalf("sample_params[%s] 回显不一致: %v", k, data["sample_params"])
		}
	}
}

// TestAppSampleParams 接入示例参数经应用接口存取，且拒绝非法键 / 覆盖平台参数
func TestAppSampleParams(t *testing.T) {
	app := seedOAuth2App(t)
	_, adminToken := newTestAdmin(t)

	put := func(params map[string]string) (int, map[string]interface{}) {
		body, _ := json.Marshal(map[string]interface{}{
			"name": app.Name, "platform": app.Platform, "mode": app.Mode,
			"types": []string{"gitee"}, "domains": app.Domains,
			"redirect_uris": []string{"https://target.example.com/callback"},
			"sample_params": params,
		})
		return doAuthedJSON(t, http.MethodPut, fmt.Sprintf("/api/apps/%d", app.ID), string(body), adminToken)
	}

	// 正常写入
	if code, m := put(map[string]string{"state": "abc", "oauth_state": "def"}); code != http.StatusOK {
		t.Fatalf("写入示例参数应成功: %d %v", code, m)
	}
	if cfg := readAppConfig(t, app.ID); cfg.SampleParams["state"] != "abc" {
		t.Fatalf("示例参数未持久化: %v", cfg.SampleParams)
	}

	// 非法参数名 → 400
	if code, _ := put(map[string]string{"bad key": "x"}); code != http.StatusBadRequest {
		t.Fatalf("非法参数名应 400: %d", code)
	}
	// 覆盖平台参数 → 400
	if code, _ := put(map[string]string{"client_id": "x"}); code != http.StatusBadRequest {
		t.Fatalf("覆盖 client_id 应 400: %d", code)
	}
}

// TestOAuth2AuthorizeWPSStyle 复现 WPS 企业 SSO 的真实请求形态：
// 回跳地址取自应用白名单配置，接入方参数取自接入示例参数，
// 授权请求用 redirect_url 传参 —— 此前会被拒为「缺少 client_id 或 redirect_uri」。
func TestOAuth2AuthorizeWPSStyle(t *testing.T) {
	app := seedOAuth2App(t)
	configureApp(t, app, wpsAppConfig.CallbackBase, wpsAppConfig.SampleParams)
	cfg := readAppConfig(t, app.ID)

	// 接入方在回跳地址上追加自己的参数（oauth_state 每次请求都不同）
	callback := cfg.RedirectURI + "?" + encodeParams(cfg.SampleParams)
	state := cfg.SampleParams["state"]

	w, body := doGet(t, fmt.Sprintf("/authorize?client_id=%s&redirect_url=%s&scope=email&state=%s",
		app.AppID, url.QueryEscape(callback), url.QueryEscape(state)))
	if w.Code != http.StatusOK || !strings.Contains(body, "授权登录") {
		t.Fatalf("WPS 形态请求应进入授权页: %d %s", w.Code, body)
	}
	// 渠道链接须重放本次参数，且保留接入方原始参数（回跳地址不能被改名/丢 query）
	for _, want := range []string{"redirect_url=", url.QueryEscape("oauth_state=" + cfg.SampleParams["oauth_state"])} {
		if !strings.Contains(body, want) {
			t.Fatalf("授权页渠道链接未重放接入方参数（缺 %s）: %s", want, body)
		}
	}
}

// TestOAuth2TokenViaGETQuery 覆盖用 GET + query 换令牌的接入方（QQ / 微信 / WPS 企业 SSO 风格）。
// 仅注册 POST 时这类请求会落到前端路由回退、返回 HTML，接入方报
// "token response json decode failed"；后端命名空间也不得回退前端页面。
func TestOAuth2TokenViaGETQuery(t *testing.T) {
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"
	user, _ := testUser(t)

	code := models.OAuthCode{
		Code:        strings.ToUpper(utils.RandomString(32)),
		ClientID:    app.AppID,
		UserID:      user.ID,
		Scope:       "openid profile",
		RedirectURI: cb,
		ExpiresAt:   time.Now().Add(time.Minute),
	}
	if err := database.DB.Create(&code).Error; err != nil {
		t.Fatalf("插入授权码失败: %v", err)
	}

	// GET + query 换令牌 → 200 JSON，含 access_token
	w, body := doGet(t, "/token?grant_type=authorization_code&code="+code.Code+
		"&redirect_uri="+url.QueryEscape(cb)+"&client_id="+app.AppID+"&client_secret="+app.AppKey)
	if w.Code != http.StatusOK {
		t.Fatalf("GET 换令牌应成功: %d %s", w.Code, body)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("GET 换令牌应返回 JSON，实际 %s", ct)
	}
	var tok map[string]interface{}
	if err := json.Unmarshal([]byte(body), &tok); err != nil || tok["access_token"] == nil {
		t.Fatalf("GET 换令牌响应异常: %v %s", err, body)
	}

	// 客户端认证失败也必须是 JSON（不能是前端页面）
	wBad, bodyBad := doGet(t, "/token?grant_type=authorization_code&code=x&client_id=nope")
	if wBad.Code != http.StatusUnauthorized || !strings.Contains(wBad.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("GET 换令牌失败应返回 JSON 错误: %d %s", wBad.Code, bodyBad)
	}

	// 后端命名空间下的未知路径：GET 也不能回退前端页面
	// （/token/ 除外：命中 gin 的尾斜杠 301 跳转，属标准行为）
	for _, p := range []string{"/oauth2/token", "/oauth2/whatever", "/api/oauth2/nope", "/.well-known/nope"} {
		wUnknown, bodyUnknown := doGet(t, p)
		if wUnknown.Code != http.StatusNotFound || !strings.Contains(wUnknown.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("后端路径 %s 应返回 404 JSON，实际 %d %s", p, wUnknown.Code, bodyUnknown)
		}
	}

	// 说明：测试二进制的 CWD 是 handlers/，找不到 web/dist，因此这里不校验
	// 「非后端路径回退 SPA 页面」——该行为需要构建产物，已在线上实测（GET /token 曾返回 index.html）。
}

// TestOAuth2AuthorizeNonStandardParams 覆盖非标准参数名与带动态 query 的回调地址
func TestOAuth2AuthorizeNonStandardParams(t *testing.T) {
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"
	// 动态 query 用接入示例参数（同样来自应用配置，而非代码里的字面量）
	dynamic := cb + "?" + encodeParams(wpsAppConfig.SampleParams)

	// redirect_url 别名 + 回调带动态 query → 授权页，而非「缺少 client_id 或 redirect_uri」
	w, body := doGet(t, fmt.Sprintf("/authorize?client_id=%s&redirect_url=%s&scope=email&state=allowed",
		app.AppID, url.QueryEscape(dynamic)))
	if w.Code != http.StatusOK || !strings.Contains(body, "授权登录") {
		t.Fatalf("redirect_url 别名 + 动态 query 应进入授权页: %d %s", w.Code, body)
	}

	// appid / app_id 别名同样可用
	for _, alias := range []string{"appid", "app_id"} {
		wa, bodyA := doGet(t, fmt.Sprintf("/authorize?%s=%s&redirect_url=%s&state=s", alias, app.AppID, url.QueryEscape(cb)))
		if wa.Code != http.StatusOK || !strings.Contains(bodyA, "授权登录") {
			t.Fatalf("%s 别名应被接受: %d %s", alias, wa.Code, bodyA)
		}
	}

	// POST /authorize 表单体（此前只读 query，POST 路由形同虚设）
	wp, _ := doPostForm(t, "/authorize", url.Values{
		"client_id":    {app.AppID},
		"redirect_uri": {cb},
		"state":        {wpsAppConfig.SampleParams["state"]},
	})
	if wp.Code != http.StatusOK || !strings.Contains(wp.Body.String(), "授权登录") {
		t.Fatalf("POST 表单应被接受: %d %s", wp.Code, wp.Body.String())
	}
	// 授权页里的渠道链接需重放本次授权参数，否则点进去会丢参数
	if html := wp.Body.String(); !strings.Contains(html, "client_id=") || !strings.Contains(html, "redirect_uri=") {
		t.Fatalf("授权页渠道链接缺少授权参数: %s", html)
	}

	// 非法落点仍须拒绝（host 不同）
	wBad, bodyBad := doGet(t, fmt.Sprintf("/authorize?client_id=%s&redirect_url=%s&state=s",
		app.AppID, url.QueryEscape("https://evil.example.com?"+encodeParams(wpsAppConfig.SampleParams))))
	if wBad.Code != http.StatusBadRequest || !strings.Contains(bodyBad, "白名单") {
		t.Fatalf("非法 redirect_url 应被拒绝: %d %s", wBad.Code, bodyBad)
	}
}
