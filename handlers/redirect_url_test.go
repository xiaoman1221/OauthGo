package handlers

import (
	"net/url"
	"strings"
	"testing"
)

// TestBuildRedirectURL 覆盖 buildRedirectURL 在 redirect_uri 是否已携带
// query 时的两种拼接行为（防止出现 "https://x/open?juhe=1?code=..." 双 ? 拼接）。
func TestBuildRedirectURL(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		params map[string]string
		want   string
	}{
		{
			name: "无 query 时用 ? 分隔",
			base: "https://example.com/callback",
			params: map[string]string{
				"type": "wx",
				"code": "601E5ECFB0F0FA21A46A2B35",
				"sign": "f4a22d1d3c0c2fd73b2734a5514b6c04",
			},
			want: "https://example.com/callback?code=601E5ECFB0F0FA21A46A2B35&sign=f4a22d1d3c0c2fd73b2734a5514b6c04&type=wx",
		},
		{
			name: "已携带 ?juhe=1 时用 & 追加，绝不出现两个 ?",
			base: "https://www.moeupcloud.com/open?juhe=1",
			params: map[string]string{
				"type": "wx",
				"code": "601E5ECFB0F0FA21A46A2B35",
				"sign": "f4a22d1d3c0c2fd73b2734a5514b6c04",
			},
			want: "https://www.moeupcloud.com/open?juhe=1&code=601E5ECFB0F0FA21A46A2B35&sign=f4a22d1d3c0c2fd73b2734a5514b6c04&type=wx",
		},
		{
			name: "已携带多个 query 时继续用 & 追加",
			base: "https://example.com/open?juhe=1&from=pc",
			params: map[string]string{
				"type": "qq",
				"code": "ABCD",
			},
			want: "https://example.com/open?juhe=1&from=pc&code=ABCD&type=qq",
		},
		{
			name:   "params 为空时不附加分隔符",
			base:   "https://example.com/callback?foo=bar",
			params: map[string]string{},
			want:   "https://example.com/callback?foo=bar",
		},
		{
			name:   "base 不带 query 且 params 为空，原样返回",
			base:   "https://example.com/callback",
			params: map[string]string{},
			want:   "https://example.com/callback",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRedirectURL(tt.base, tt.params)
			if got != tt.want {
				t.Fatalf("拼接错误\n got: %s\nwant: %s", got, tt.want)
			}
			if strings.Count(got, "?") > 1 {
				t.Fatalf("URL 出现了多个 ?: %s", got)
			}
			if _, err := url.Parse(got); err != nil {
				t.Fatalf("生成的不是合法 URL: %v", err)
			}
		})
	}
}
