package handlers_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"OauthGo/database"
	"OauthGo/models"
)

// 样例接入方（WPS 企业 SSO）的授权请求与回跳地址参数，按真实抓包值填写；
// 需要复现其它接入方时只改这一处。
var (
	// wpsRedirectBase 回跳地址的 scheme+host+path：测试会把它写进应用回调白名单（query 被忽略）。
	wpsRedirectBase = "https://account.wps.cn/permit/ssoafterlogin.html"

	// wpsCallbackQuery 接入方在回跳地址上追加的 query，每次请求都可能不同
	// （oauth_state 为随机 hex），正是它让「精确匹配白名单」永远无法通过。
	wpsCallbackQuery = url.Values{
		"app_id":      {"FT20260923XWAIGM"},
		"checkcfg":    {"FT20260923XWAIGM"},
		"oauth_state": {"6ba797234e2f4a2d83418c3a261f4504"},
		"state":       {"YWN0aW9uPXZlcmlmeSZjYj1odHRwcyUzQSUyRiUyRmFjY291bnQud3BzLmNu"},
	}

	// wpsAuthorizeState 授权请求自身携带的 state（接入方用于回跳后校验，平台原样透传）
	wpsAuthorizeState = "YWN0aW9uPXZlcmlmeSZjYj1odHRwczovL2FjY291bnQud3BzLmNu"
)

// wpsCallbackURL 拼出样例回跳地址（带接入方追加的动态 query）
func wpsCallbackURL() string {
	return wpsRedirectBase + "?" + wpsCallbackQuery.Encode()
}

// TestOAuth2AuthorizeWPSStyle 复现 WPS 企业 SSO 的真实请求形态：
// redirect_url 传参 + 回跳地址带每次不同的 oauth_state，此前会被拒为
// 「缺少 client_id 或 redirect_uri」，修好后应正常进入授权页。
func TestOAuth2AuthorizeWPSStyle(t *testing.T) {
	app := seedOAuth2App(t)
	// 白名单按 scheme+host+path 配置（不带 query），与实际接入配置一致
	base, err := url.Parse(wpsRedirectBase)
	if err != nil {
		t.Fatalf("样例回跳地址非法: %v", err)
	}
	whitelist := base.Scheme + "://" + base.Host + base.Path
	if err := database.DB.Model(&models.App{}).Where("id = ?", app.ID).
		Update("redirect_uris", fmt.Sprintf(`[%q]`, whitelist)).Error; err != nil {
		t.Fatal(err)
	}

	w, body := doGet(t, fmt.Sprintf("/authorize?client_id=%s&redirect_url=%s&scope=email&state=%s",
		app.AppID, url.QueryEscape(wpsCallbackURL()), url.QueryEscape(wpsAuthorizeState)))
	if w.Code != http.StatusOK || !strings.Contains(body, "授权登录") {
		t.Fatalf("WPS 形态请求应进入授权页: %d %s", w.Code, body)
	}
	// 渠道链接须重放本次参数，且仍用接入方原始的参数名（回跳地址不能被改名/丢 query）
	for _, want := range []string{"redirect_url=", url.QueryEscape("oauth_state=" + wpsCallbackQuery.Get("oauth_state"))} {
		if !strings.Contains(body, want) {
			t.Fatalf("授权页渠道链接未重放 WPS 参数（缺 %s）: %s", want, body)
		}
	}
}

// TestOAuth2AuthorizeNonStandardParams 覆盖非标准参数名与带动态 query 的回调地址
func TestOAuth2AuthorizeNonStandardParams(t *testing.T) {
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"
	dynamic := cb + "?" + wpsCallbackQuery.Encode()

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
		"state":        {wpsAuthorizeState},
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
		app.AppID, url.QueryEscape("https://evil.example.com"+"?"+wpsCallbackQuery.Encode())))
	if wBad.Code != http.StatusBadRequest || !strings.Contains(bodyBad, "白名单") {
		t.Fatalf("非法 redirect_url 应被拒绝: %d %s", wBad.Code, bodyBad)
	}
}
