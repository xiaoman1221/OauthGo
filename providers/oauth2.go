package providers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// OAuth2ChannelName 通用 OAuth2 / OIDC 客户端渠道标识
const OAuth2ChannelName = "oauth2"

// oidcDiscovery OIDC Discovery 文档（.well-known/openid-configuration）关键字段
type oidcDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

// OAuth2Provider 通用 OAuth2 / OIDC 客户端渠道。
// 配置方式二选一：
//   - discovery_url：提供 OIDC Discovery 地址，自动解析各端点；
//   - 手动填写 authorize_url / token_url / userinfo_url。
//
// 扩展配置（Config.Extra）：
//   - discovery_url      OIDC Discovery 文档地址（推荐，优先于手动端点）
//   - authorize_url      授权端点（手动模式）
//   - token_url          Token 端点（手动模式）
//   - userinfo_url       用户信息端点（手动模式）
//   - scope               授权 scope，空格分隔；默认 openid profile email
//   - token_auth          client 认证方式：post（默认，放请求体）/ basic（Authorization Basic）
//   - claims_map          JSON 对象，将外部 claims 映射为本平台字段：
//     {"openid":"sub","unionid":"unionid","nickname":"name","avatar":"picture","email":"email","gender":"gender","location":"location"}
type OAuth2Provider struct {
	cfg     Config
	client  *http.Client
	disc    *oidcDiscovery
	discErr error
}

func (p *OAuth2Provider) Name() string { return OAuth2ChannelName }

// scope 返回授权 scope（默认 openid profile email）
func (p *OAuth2Provider) scope() string {
	if s := strings.TrimSpace(p.cfg.ExtraString("scope")); s != "" {
		return s
	}
	return "openid profile email"
}

// authStyle 返回 client 认证方式
func (p *OAuth2Provider) authStyle() string {
	if strings.ToLower(strings.TrimSpace(p.cfg.ExtraString("token_auth"))) == "basic" {
		return "basic"
	}
	return "post"
}

// ensureDiscovery 若配置了 discovery_url 则拉取并缓存发现文档
func (p *OAuth2Provider) ensureDiscovery() error {
	if p.disc != nil || p.discErr != nil {
		return p.discErr
	}
	rawURL := strings.TrimSpace(p.cfg.ExtraString("discovery_url"))
	if rawURL == "" {
		return nil
	}
	var disc oidcDiscovery
	if err := getJSONClient(p.client, rawURL, nil, &disc); err != nil {
		p.discErr = fmt.Errorf("拉取 OIDC Discovery 失败: %w", err)
		return p.discErr
	}
	if disc.AuthorizationEndpoint == "" || disc.TokenEndpoint == "" {
		p.discErr = fmt.Errorf("OIDC Discovery 缺少 authorization_endpoint / token_endpoint")
		return p.discErr
	}
	p.disc = &disc
	return nil
}

// authorizeEndpoint 返回授权端点（discovery 优先）
func (p *OAuth2Provider) authorizeEndpoint() (string, error) {
	if err := p.ensureDiscovery(); err != nil {
		return "", err
	}
	if p.disc != nil && p.disc.AuthorizationEndpoint != "" {
		return p.disc.AuthorizationEndpoint, nil
	}
	u := strings.TrimSpace(p.cfg.ExtraString("authorize_url"))
	if u == "" {
		return "", fmt.Errorf("通用 OAuth2 渠道缺少 authorize_url 或 discovery_url 配置")
	}
	return u, nil
}

// tokenEndpoint 返回 Token 端点
func (p *OAuth2Provider) tokenEndpoint() (string, error) {
	if err := p.ensureDiscovery(); err != nil {
		return "", err
	}
	if p.disc != nil && p.disc.TokenEndpoint != "" {
		return p.disc.TokenEndpoint, nil
	}
	u := strings.TrimSpace(p.cfg.ExtraString("token_url"))
	if u == "" {
		return "", fmt.Errorf("通用 OAuth2 渠道缺少 token_url 或 discovery_url 配置")
	}
	return u, nil
}

// userinfoEndpoint 返回用户信息端点（部分纯 OAuth2 服务无此端点，此时返回空串）
func (p *OAuth2Provider) userinfoEndpoint() (string, error) {
	if err := p.ensureDiscovery(); err != nil {
		return "", err
	}
	if p.disc != nil && p.disc.UserinfoEndpoint != "" {
		return p.disc.UserinfoEndpoint, nil
	}
	return strings.TrimSpace(p.cfg.ExtraString("userinfo_url")), nil
}

// GetAuthURL 生成跳转授权地址
func (p *OAuth2Provider) GetAuthURL(state string) string {
	ep, err := p.authorizeEndpoint()
	if err != nil {
		return ""
	}
	q := url.Values{}
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURL)
	q.Set("response_type", "code")
	q.Set("scope", p.scope())
	q.Set("state", state)
	sep := "?"
	if strings.Contains(ep, "?") {
		sep = "&"
	}
	return ep + sep + q.Encode()
}

// GetUserInfo 使用授权码换取用户信息
func (p *OAuth2Provider) GetUserInfo(code string) (*UserInfo, error) {
	tokenURL, err := p.tokenEndpoint()
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", p.cfg.RedirectURL)

	var headers map[string]string
	if p.authStyle() == "basic" {
		headers = map[string]string{
			"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(p.cfg.ClientID+":"+p.cfg.ClientSecret)),
		}
	} else {
		form.Set("client_id", p.cfg.ClientID)
		form.Set("client_secret", p.cfg.ClientSecret)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		Description  string `json:"error_description"`
	}
	if err := postFormClient(p.client, tokenURL, headers, form, &tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.AccessToken == "" {
		if tokenResp.Error != "" {
			return nil, fmt.Errorf("外部授权服务器换取令牌失败: %s %s", tokenResp.Error, tokenResp.Description)
		}
		return nil, fmt.Errorf("外部授权服务器未返回 access_token")
	}

	// 获取用户信息：优先 userinfo 端点；无端点时报清晰错误
	userURL, err := p.userinfoEndpoint()
	if err != nil {
		return nil, err
	}
	if userURL == "" {
		return nil, fmt.Errorf("外部授权服务器未提供 userinfo 端点，请在扩展配置中填写 userinfo_url")
	}

	var claims map[string]interface{}
	if err := getJSONClient(p.client, userURL,
		map[string]string{"Authorization": "Bearer " + tokenResp.AccessToken}, &claims); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, fmt.Errorf("外部授权服务器 userinfo 返回为空")
	}

	info, err := mapClaimsToUserInfo(p.cfg.ExtraString("claims_map"), claims, tokenResp.AccessToken)
	if err != nil {
		return nil, err
	}
	if tokenResp.RefreshToken != "" {
		info.Extra = ensureExtra(info.Extra)
		info.Extra["refresh_token"] = tokenResp.RefreshToken
	}
	return info, nil
}

// mapClaimsToUserInfo 将外部 userinfo claims 映射为平台 UserInfo。
// claims_map（JSON）可覆盖默认映射，未指定的键回退到通用默认取值。
func mapClaimsToUserInfo(claimsMapRaw string, claims map[string]interface{}, accessToken string) (*UserInfo, error) {
	cfg := map[string]string{}
	if strings.TrimSpace(claimsMapRaw) != "" {
		if err := json.Unmarshal([]byte(claimsMapRaw), &cfg); err != nil {
			return nil, fmt.Errorf("claims_map 必须是合法的 JSON 对象: %v", err)
		}
	}

	getClaim := func(k string) (string, bool) {
		v, ok := claims[k]
		if !ok || v == nil {
			return "", false
		}
		switch t := v.(type) {
		case string:
			return t, t != ""
		case float64:
			return strconv.FormatInt(int64(t), 10), true
		case json.Number:
			return t.String(), true
		case bool:
			return strconv.FormatBool(t), true
		default:
			return fmt.Sprintf("%v", t), true
		}
	}
	field := func(key string, defaults ...string) string {
		if v := cfg[key]; v != "" {
			if s, ok := getClaim(v); ok {
				return s
			}
		}
		for _, d := range defaults {
			if s, ok := getClaim(d); ok {
				return s
			}
		}
		return ""
	}

	openid := field("openid", "sub", "openid", "id", "user_id")
	if openid == "" {
		return nil, fmt.Errorf("外部 userinfo 缺少可用的唯一标识（sub/openid/id），可配置 claims_map.openid 指定")
	}

	info := &UserInfo{
		OpenID:      openid,
		UnionID:     field("unionid", "unionid", "union_id"),
		Nickname:    field("nickname", "name", "preferred_username", "nickname", "username", "login", "unique_name"),
		Avatar:      field("avatar", "picture", "avatar_url", "avatar", "headimgurl", "portrait"),
		Email:       field("email", "email"),
		AccessToken: accessToken,
		Extra:       map[string]interface{}{},
	}
	// 透传性别 / 地区等，供彩虹聚合协议 / userinfo 快照使用
	for _, k := range []string{"gender", "location", "locale", "phone", "phone_number"} {
		if v := field(k, k); v != "" {
			info.Extra[k] = v
		}
	}
	return info, nil
}

func ensureExtra(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}

// Validate 校验渠道配置完整性（供「测试渠道」使用）
func (p *OAuth2Provider) Validate() error {
	if strings.TrimSpace(p.cfg.ClientID) == "" {
		return fmt.Errorf("缺少 Client ID")
	}
	if _, err := p.authorizeEndpoint(); err != nil {
		return err
	}
	if _, err := p.tokenEndpoint(); err != nil {
		return err
	}
	ui, err := p.userinfoEndpoint()
	if err != nil {
		return err
	}
	if strings.TrimSpace(ui) == "" {
		return fmt.Errorf("缺少 userinfo 端点：请配置 discovery_url 或 userinfo_url")
	}
	return nil
}
