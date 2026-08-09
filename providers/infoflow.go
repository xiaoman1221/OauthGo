package providers

import (
	"fmt"
	"net/url"
)

// InfoflowProvider 如流（百度企业IM）登录
type InfoflowProvider struct {
	cfg Config
}

func (p *InfoflowProvider) Name() string { return "infoflow" }

func (p *InfoflowProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://xpc.im.baidu.com/oauth2/authorize?appid=%s&redirect_uri=%s&state=%s",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *InfoflowProvider) GetUserInfo(code string) (*UserInfo, error) {
	tokenURL := fmt.Sprintf(
		"https://xpc.im.baidu.com/oauth2/token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		p.cfg.ClientID, p.cfg.ClientSecret, url.QueryEscape(code))

	var token struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrMsg      string `json:"error_description"`
	}
	if err := getJSON(tokenURL, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("如流获取 access_token 失败: %s %s", token.Error, token.ErrMsg)
	}

	userURL := fmt.Sprintf("https://xpc.im.baidu.com/oauth2/userinfo?access_token=%s",
		url.QueryEscape(token.AccessToken))

	var user struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := getJSON(userURL, &user); err != nil {
		return nil, err
	}

	return &UserInfo{
		OpenID:   user.ID,
		Nickname: user.Name,
		Avatar:   user.AvatarURL,
		Email:    user.Email,
	}, nil
}
