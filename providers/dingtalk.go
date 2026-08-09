package providers

import (
	"fmt"
	"net/url"
)

// DingTalkProvider 钉钉登录
type DingTalkProvider struct {
	cfg Config
}

func (p *DingTalkProvider) Name() string { return "dingtalk" }

func (p *DingTalkProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://login.dingtalk.com/oauth2/auth?redirect_uri=%s&response_type=code&client_id=%s&scope=openid&state=%s&prompt=consent",
		url.QueryEscape(p.cfg.RedirectURL), p.cfg.ClientID, url.QueryEscape(state))
}

func (p *DingTalkProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		AccessToken string `json:"accessToken"`
		ErrorCode   string `json:"code"`
		ErrMsg      string `json:"message"`
	}
	if err := postJSON("https://api.dingtalk.com/v1.0/oauth2/userAccessToken", map[string]interface{}{
		"clientId":     p.cfg.ClientID,
		"clientSecret": p.cfg.ClientSecret,
		"code":         code,
		"grantType":    "authorization_code",
	}, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("钉钉获取 access_token 失败: %s %s", token.ErrorCode, token.ErrMsg)
	}

	userURL := "https://api.dingtalk.com/v1.0/contact/users/me"
	var user struct {
		OpenID    string `json:"openId"`
		UnionID   string `json:"unionId"`
		Nick      string `json:"nick"`
		AvatarURL string `json:"avatarUrl"`
		Email     string `json:"email"`
	}
	if err := getJSONWithHeaders(userURL, map[string]string{
		"x-acs-dingtalk-access-token": token.AccessToken,
	}, &user); err != nil {
		return nil, err
	}

	return &UserInfo{
		OpenID:   user.OpenID,
		UnionID:  user.UnionID,
		Nickname: user.Nick,
		Avatar:   user.AvatarURL,
		Email:    user.Email,
	}, nil
}
