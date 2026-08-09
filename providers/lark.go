package providers

import (
	"fmt"
	"net/url"
)

// LarkProvider 飞书登录
type LarkProvider struct {
	cfg Config
}

func (p *LarkProvider) Name() string { return "lark" }

func (p *LarkProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://open.feishu.cn/open-apis/authen/v1/index?app_id=%s&redirect_uri=%s&state=%s",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *LarkProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		Code        int    `json:"code"`
		Msg         string `json:"msg"`
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := postJSON("https://open.feishu.cn/open-apis/authen/v1/oidc/access_token", map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  p.cfg.RedirectURL,
		"client_id":     p.cfg.ClientID,
		"client_secret": p.cfg.ClientSecret,
	}, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("飞书获取 access_token 失败: %d %s", token.Code, token.Msg)
	}

	var user struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
		Email   string `json:"email"`
		UnionID string `json:"union_id"`
	}
	if err := getJSONWithHeaders("https://open.feishu.cn/open-apis/authen/v1/user_info", map[string]string{
		"Authorization": "Bearer " + token.AccessToken,
	}, &user); err != nil {
		return nil, err
	}

	return &UserInfo{
		OpenID:   user.Sub,
		UnionID:  user.UnionID,
		Nickname: user.Name,
		Avatar:   user.Picture,
		Email:    user.Email,
	}, nil
}
