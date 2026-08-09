package providers

import (
	"fmt"
	"net/http"
	"net/url"
)

// GitHubProvider GitHub 登录
type GitHubProvider struct {
	cfg    Config
	client *http.Client
}

func (p *GitHubProvider) Name() string { return "github" }

func (p *GitHubProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.RedirectURL),
		url.QueryEscape("read:user user:email"), url.QueryEscape(state))
}

func (p *GitHubProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := postFormClient(p.client, "https://github.com/login/oauth/access_token",
		map[string]string{"Accept": "application/json"}, url.Values{
			"client_id":     {p.cfg.ClientID},
			"client_secret": {p.cfg.ClientSecret},
			"code":          {code},
			"redirect_uri":  {p.cfg.RedirectURL},
		}, &token); err != nil {
		return nil, err
	}

	var user struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := getJSONClient(p.client, "https://api.github.com/user",
		map[string]string{
			"Authorization": "token " + token.AccessToken,
			"Accept":        "application/vnd.github+json",
			"User-Agent":    "OauthGo",
		}, &user); err != nil {
		return nil, err
	}

	nickname := user.Name
	if nickname == "" {
		nickname = user.Login
	}
	return &UserInfo{
		OpenID:      fmt.Sprintf("%d", user.ID),
		Nickname:    nickname,
		Avatar:      user.AvatarURL,
		Email:       user.Email,
		AccessToken: token.AccessToken,
	}, nil
}
