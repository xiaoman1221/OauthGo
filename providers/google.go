package providers

import (
	"fmt"
	"net/http"
	"net/url"
)

// GoogleProvider Google 登录
type GoogleProvider struct {
	cfg    Config
	client *http.Client
}

func (p *GoogleProvider) Name() string { return "google" }

func (p *GoogleProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.RedirectURL),
		url.QueryEscape("openid email profile"), url.QueryEscape(state))
}

func (p *GoogleProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := postFormClient(p.client, "https://oauth2.googleapis.com/token", nil, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"redirect_uri":  {p.cfg.RedirectURL},
	}, &token); err != nil {
		return nil, err
	}

	var user struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
		Email   string `json:"email"`
	}
	if err := getJSONClient(p.client, "https://www.googleapis.com/oauth2/v2/userinfo",
		map[string]string{"Authorization": "Bearer " + token.AccessToken}, &user); err != nil {
		return nil, err
	}

	return &UserInfo{
		OpenID:      user.ID,
		Nickname:    user.Name,
		Avatar:      user.Picture,
		Email:       user.Email,
		AccessToken: token.AccessToken,
	}, nil
}
