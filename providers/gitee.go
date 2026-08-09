package providers

import (
	"fmt"
	"net/url"
)

// GiteeProvider Gitee（码云）登录
type GiteeProvider struct {
	cfg Config
}

func (p *GiteeProvider) Name() string { return "gitee" }

func (p *GiteeProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://gitee.com/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *GiteeProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := postForm("https://gitee.com/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {p.cfg.ClientID},
		"redirect_uri":  {p.cfg.RedirectURL},
		"client_secret": {p.cfg.ClientSecret},
	}, &token); err != nil {
		return nil, err
	}

	userURL := fmt.Sprintf("https://gitee.com/api/v5/user?access_token=%s", url.QueryEscape(token.AccessToken))
	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := getJSON(userURL, &user); err != nil {
		return nil, err
	}

	nickname := user.Name
	if nickname == "" {
		nickname = user.Login
	}
	return &UserInfo{
		OpenID:   fmt.Sprintf("%d", user.ID),
		Nickname: nickname,
		Avatar:   user.AvatarURL,
		Email:    user.Email,
	}, nil
}
