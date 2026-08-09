package providers

import (
	"fmt"
	"net/http"
	"net/url"
)

// MicrosoftProvider Microsoft（Entra ID / 个人账号）登录
type MicrosoftProvider struct {
	cfg    Config
	client *http.Client
}

func (p *MicrosoftProvider) Name() string { return "microsoft" }

func (p *MicrosoftProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://login.microsoftonline.com/common/oauth2/v2.0/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.RedirectURL),
		url.QueryEscape("openid profile email User.Read"), url.QueryEscape(state))
}

func (p *MicrosoftProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := postFormClient(p.client, "https://login.microsoftonline.com/common/oauth2/v2.0/token", nil, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"redirect_uri":  {p.cfg.RedirectURL},
		"scope":         {"User.Read"},
	}, &token); err != nil {
		return nil, err
	}

	var user struct {
		ID                string `json:"id"`
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
	}
	if err := getJSONClient(p.client, "https://graph.microsoft.com/v1.0/me",
		map[string]string{"Authorization": "Bearer " + token.AccessToken}, &user); err != nil {
		return nil, err
	}

	email := user.Mail
	if email == "" {
		email = user.UserPrincipalName
	}
	return &UserInfo{
		OpenID:      user.ID,
		Nickname:    user.DisplayName,
		Email:       email,
		AccessToken: token.AccessToken,
	}, nil
}
