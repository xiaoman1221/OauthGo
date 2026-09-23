package services

import (
	"testing"

	"OauthGo/models"
)

// TestValidOAuthRedirectURI 覆盖回调白名单按 scheme + host + path 匹配（忽略 query）
func TestValidOAuthRedirectURI(t *testing.T) {
	app := &models.App{RedirectURIs: `["https://target.example.com/callback","https://a.example.com:8443/cb"]`}
	cases := []struct {
		in   string
		want bool
	}{
		{"https://target.example.com/callback", true},                       // 精确匹配
		{"https://target.example.com/callback?oauth_state=x&state=y", true}, // 接入方追加的动态 query
		{"https://target.example.com/callback#frag", true},                  // fragment 同样忽略
		{"https://TARGET.example.com/callback", true},                       // host 大小写不敏感
		{"https://a.example.com:8443/cb?x=1", true},                         // 非默认端口
		{"https://target.example.com/callback2", false},                     // path 不同
		{"https://target.example.com/other", false},                         // path 不同
		{"https://evil.example.com/callback", false},                        // host 不同
		{"https://target.example.com:8443/callback", false},                 // 端口不同
		{"http://target.example.com/callback", false},                       // scheme 不同
		{"/callback", false},                                                // 相对地址
		{"", false},
	}
	for _, c := range cases {
		if got := ValidOAuthRedirectURI(app, c.in); got != c.want {
			t.Errorf("ValidOAuthRedirectURI(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	// 未配置白名单 → 一律拒绝（不回退到域名白名单）
	if ValidOAuthRedirectURI(&models.App{}, "https://target.example.com/callback") {
		t.Error("未配置 redirect_uris 时应拒绝")
	}
	if ValidOAuthRedirectURI(&models.App{RedirectURIs: `["not a uri"]`}, "https://target.example.com/callback") {
		t.Error("白名单存在非法条目时不应放行")
	}
}

func TestSameOAuthRedirect(t *testing.T) {
	if !SameOAuthRedirect("https://t.example.com/cb?a=1", "https://t.example.com/cb?b=2") {
		t.Error("同落点（仅 query 不同）应判等")
	}
	if SameOAuthRedirect("https://t.example.com/cb", "https://t.example.com/cb/x") {
		t.Error("path 不同不应判等")
	}
	if SameOAuthRedirect("", "https://t.example.com/cb") || SameOAuthRedirect("https://t.example.com/cb", "") {
		t.Error("空地址不应判等")
	}
}
