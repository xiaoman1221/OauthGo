package providers

import (
	"fmt"
	"net/http"
	"net/url"
)

// LinkedInProvider LinkedIn 登录
type LinkedInProvider struct {
	cfg    Config
	client *http.Client
}

func (p *LinkedInProvider) Name() string { return "linkedin" }

func (p *LinkedInProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://www.linkedin.com/oauth/v2/authorization?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
		url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.RedirectURL),
		url.QueryEscape("openid profile email"), url.QueryEscape(state))
}

func (p *LinkedInProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := postFormClient(p.client, "https://www.linkedin.com/oauth/v2/accessToken", nil, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"redirect_uri":  {p.cfg.RedirectURL},
	}, &token); err != nil {
		return nil, err
	}

	var user struct {
		Sub       string `json:"sub"`
		Name      string `json:"name"`
		GivenName string `json:"given_name"`
		Email     string `json:"email"`
		Picture   string `json:"picture"`
	}
	if err := getJSONClient(p.client, "https://api.linkedin.com/v2/userinfo",
		map[string]string{"Authorization": "Bearer " + token.AccessToken}, &user); err != nil {
		return nil, err
	}

	nickname := user.Name
	if nickname == "" {
		nickname = user.GivenName
	}
	return &UserInfo{
		OpenID:      user.Sub,
		Nickname:    nickname,
		Avatar:      user.Picture,
		Email:       user.Email,
		AccessToken: token.AccessToken,
	}, nil
}
