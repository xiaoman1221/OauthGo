package providers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOAuth2AuthURLManual(t *testing.T) {
	p := &OAuth2Provider{
		cfg: Config{
			ClientID:    "cid",
			RedirectURL: "https://oauth.example.com/api/oauth/oauth2/callback",
			Extra:       map[string]interface{}{"authorize_url": "https://idp.example.com/oauth2/authorize", "scope": "openid profile"},
			UseProxy:    false,
		},
	}
	u := p.GetAuthURL("s1")
	for _, want := range []string{"https://idp.example.com/oauth2/authorize?", "client_id=cid", "redirect_uri=https%3A%2F%2Foauth.example.com%2Fapi%2Foauth%2Foauth2%2Fcallback", "response_type=code", "scope=openid+profile", "state=s1"} {
		if !strings.Contains(u, want) {
			t.Fatalf("授权地址缺少 %q: %s", want, u)
		}
	}
}

func TestOAuth2AuthURLMissingEndpoint(t *testing.T) {
	p := &OAuth2Provider{cfg: Config{ClientID: "cid", RedirectURL: "https://x/cb", Extra: map[string]interface{}{}}}
	if got := p.GetAuthURL("s1"); got != "" {
		t.Fatalf("缺少端点时应返回空授权地址: %s", got)
	}
}

// TestOAuth2DiscoveryFlow 使用 httptest 模拟外部 OIDC Provider：
// discovery → 授权地址生成 → 换 token → 拉 userinfo → claims 映射
func TestOAuth2DiscoveryFlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"issuer":"http://%s","authorization_endpoint":"http://%s/authorize","token_endpoint":"http://%s/token","userinfo_endpoint":"http://%s/userinfo","jwks_uri":"http://%s/jwks"}`, r.Host, r.Host, r.Host, r.Host, r.Host)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("token 表单解析失败: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "the-code" {
			t.Errorf("token 参数异常: %v", r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"at-123","token_type":"Bearer","refresh_token":"rt-456","scope":"openid profile email"}`)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer at-123" {
			t.Errorf("userinfo 缺少 Bearer: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"sub":"user-001","name":"张三","picture":"https://avatar.example.com/1.png","email":"zhangsan@example.com","gender":"男","location":"广东"}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	p := &OAuth2Provider{
		cfg: Config{
			ClientID:     "cid",
			ClientSecret: "secret",
			RedirectURL:  "https://oauth.example.com/api/oauth/oauth2/callback",
			Extra:        map[string]interface{}{"discovery_url": srv.URL + "/.well-known/openid-configuration"},
		},
		client: httpClient,
	}

	u := p.GetAuthURL("s1")
	if !strings.Contains(u, srv.URL+"/authorize?") {
		t.Fatalf("授权地址未使用 discovery 的端点: %s", u)
	}

	info, err := p.GetUserInfo("the-code")
	if err != nil {
		t.Fatalf("GetUserInfo 失败: %v", err)
	}
	if info.OpenID != "user-001" {
		t.Fatalf("openid 映射错误: %q", info.OpenID)
	}
	if info.Nickname != "张三" || info.Avatar != "https://avatar.example.com/1.png" || info.Email != "zhangsan@example.com" {
		t.Fatalf("profile 映射错误: %+v", info)
	}
	if info.Extra["gender"] != "男" || info.Extra["location"] != "广东" {
		t.Fatalf("extra 透传错误: %+v", info.Extra)
	}
	if info.Extra["refresh_token"] != "rt-456" {
		t.Fatalf("refresh_token 未透传: %+v", info.Extra)
	}
	if info.AccessToken != "at-123" {
		t.Fatalf("access_token 未保存: %q", info.AccessToken)
	}
}

func TestMapClaimsToUserInfo(t *testing.T) {
	claims := map[string]interface{}{
		"id":            float64(10086),
		"login":         "zhangsan",
		"avatar_url":    "https://x/avatar.png",
		"email":         "z@example.com",
		"name":          "Zhang San",
		"custom_userid": "CU-01",
	}
	// 默认映射：id / login / avatar_url / email / name
	info, err := mapClaimsToUserInfo("", claims, "at")
	if err != nil {
		t.Fatalf("默认映射失败: %v", err)
	}
	if info.OpenID != "10086" || info.Nickname != "Zhang San" || info.Avatar != "https://x/avatar.png" || info.Email != "z@example.com" {
		t.Fatalf("默认映射结果错误: %+v", info)
	}

	// claims_map 覆盖：openid 用 custom_userid，nickname 优先 login
	info2, err := mapClaimsToUserInfo(`{"openid":"custom_userid","nickname":"login"}`, claims, "at")
	if err != nil {
		t.Fatalf("claims_map 映射失败: %v", err)
	}
	if info2.OpenID != "CU-01" || info2.Nickname != "zhangsan" {
		t.Fatalf("claims_map 结果错误: %+v", info2)
	}

	// 缺少唯一标识时应报错
	if _, err := mapClaimsToUserInfo("", map[string]interface{}{"nickname": "x"}, "at"); err == nil {
		t.Fatal("缺少唯一标识时应报错")
	}
}

func TestOAuth2Validate(t *testing.T) {
	// 手动端点齐全 → 通过
	p := &OAuth2Provider{cfg: Config{ClientID: "cid", Extra: map[string]interface{}{
		"authorize_url": "https://idp/a", "token_url": "https://idp/t", "userinfo_url": "https://idp/u",
	}}}
	if err := p.Validate(); err != nil {
		t.Fatalf("配置齐全时应通过: %v", err)
	}
	// 缺 userinfo → 报错
	p2 := &OAuth2Provider{cfg: Config{ClientID: "cid", Extra: map[string]interface{}{
		"authorize_url": "https://idp/a", "token_url": "https://idp/t",
	}}}
	if err := p2.Validate(); err == nil {
		t.Fatal("缺 userinfo_url 时应报错")
	}
	// 空配置 → 报错
	p3 := &OAuth2Provider{cfg: Config{}}
	if err := p3.Validate(); err == nil {
		t.Fatal("空配置时应报错")
	}
}
