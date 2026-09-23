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

// TestOAuth2AuthorizeWPSStyle 复现 WPS 企业 SSO 的真实请求形态：
// redirect_url 传参 + 回跳地址带每次不同的 oauth_state，此前会被拒为
// 「缺少 client_id 或 redirect_uri」，修好后应正常进入授权页。
func TestOAuth2AuthorizeWPSStyle(t *testing.T) {
	app := seedOAuth2App(t)
	if err := database.DB.Model(&models.App{}).Where("id = ?", app.ID).
		Update("redirect_uris", `["https://account.wps.cn/permit/ssoafterlogin.html"]`).Error; err != nil {
		t.Fatal(err)
	}
	cb := "https://account.wps.cn/permit/ssoafterlogin.html?app_id=FT20260923XWAIGM" +
		"&checkcfg=FT20260923XWAIGM&oauth_state=6ba797234e2f4a2d83418c3a261f4504" +
		"&state=YWN0aW9uPXZlcmlmeSZjYj1odHRwcyUzQSUyRiUyRmFjY291bnQud3BzLmNu"
	w, body := doGet(t, fmt.Sprintf("/authorize?client_id=%s&redirect_url=%s&scope=email&state=%s",
		app.AppID, url.QueryEscape(cb), url.QueryEscape("YWN0aW9uPXZlcmlmeSZjYj1odHRwczovL2FjY291bnQud3BzLmNu")))
	if w.Code != http.StatusOK || !strings.Contains(body, "授权登录") {
		t.Fatalf("WPS 形态请求应进入授权页: %d %s", w.Code, body)
	}
	// 渠道链接须重放本次参数，且仍用接入方原始的参数名（回跳地址不能被改名/丢 query）
	if !strings.Contains(body, "redirect_url=") || !strings.Contains(body, "oauth_state%3D") {
		t.Fatalf("授权页渠道链接未重放 WPS 参数: %s", body)
	}
}

// TestOAuth2AuthorizeNonStandardParams 覆盖非标准参数名与带动态 query 的回调地址
func TestOAuth2AuthorizeNonStandardParams(t *testing.T) {
	app := seedOAuth2App(t)
	cb := "https://target.example.com/callback"
	dynamic := cb + "?app_id=FT20260923XWAIGM&oauth_state=6ba797234e2f4a2d83418c3a261f4504&state=YWN0aW9u"

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
		"state":        {"s3"},
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
		app.AppID, url.QueryEscape("https://evil.example.com/callback?oauth_state=x")))
	if wBad.Code != http.StatusBadRequest || !strings.Contains(bodyBad, "白名单") {
		t.Fatalf("非法 redirect_url 应被拒绝: %d %s", wBad.Code, bodyBad)
	}
}
