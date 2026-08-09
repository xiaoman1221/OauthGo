package services

import (
	"testing"

	"OauthGo/models"
)

func TestNormalizeDomains(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"https://a.example.com/path, b.example.com\nc.example.com", "a.example.com\nb.example.com\nc.example.com", false},
		{"EXAMPLE.com , www.example.com", "example.com\nwww.example.com", false},
		{"example.com;  sub.example.com  example.com", "example.com\nsub.example.com", false},
		{"bad domain", "", true},
		{"", "", false},
	}
	for _, c := range cases {
		got, msg := NormalizeDomains(c.in)
		if c.wantErr && msg == "" {
			t.Fatalf("NormalizeDomains(%q) 应返回错误", c.in)
		}
		if !c.wantErr && got != c.want {
			t.Fatalf("NormalizeDomains(%q) = %q, 期望 %q", c.in, got, c.want)
		}
	}
}

func TestValidateRedirect(t *testing.T) {
	app := &models.App{Domains: "example.com"}
	cases := []struct {
		uri  string
		want bool
	}{
		{"https://example.com/oauth/callback", true},
		{"https://www.example.com/oauth/callback", true},
		{"https://a.b.example.com/login?x=1", true},
		{"https://example.com.evil.com/callback", false},
		{"https://evil-example.com/callback", false},
		{"https://other.com/callback", false},
		{"http://example.com/callback", true},
		{"ftp://example.com/x", false},
		{"", false},
	}
	for _, c := range cases {
		if got := ValidateRedirect(c.uri, app); got != c.want {
			t.Fatalf("ValidateRedirect(%q) = %v, 期望 %v", c.uri, got, c.want)
		}
	}

	// 子域名白名单：仅允许该子域名及更深层级
	subApp := &models.App{Domains: "app.example.com"}
	subCases := []struct {
		uri  string
		want bool
	}{
		{"https://app.example.com/cb", true},
		{"https://a.app.example.com/cb", true},
		{"https://www.example.com/cb", false},
		{"https://example.com/cb", false},
		{"https://app.example.com.evil.com/cb", false},
	}
	for _, c := range subCases {
		if got := ValidateRedirect(c.uri, subApp); got != c.want {
			t.Fatalf("子域名匹配 ValidateRedirect(%q) = %v, 期望 %v", c.uri, got, c.want)
		}
	}

	// 未配置白名单：允许任意 http(s)
	openApp := &models.App{Domains: ""}
	if !ValidateRedirect("https://anything.com/x", openApp) {
		t.Fatalf("未配置白名单应允许任意地址")
	}
	if ValidateRedirect("javascript:alert(1)", openApp) {
		t.Fatalf("非 http(s) 地址应拒绝")
	}
}
