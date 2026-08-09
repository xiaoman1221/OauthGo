package providers

import (
	"fmt"
	"net/http"
	"net/url"
)

// FacebookProvider Facebook 登录
type FacebookProvider struct {
	cfg    Config
	client *http.Client
}

func (p *FacebookProvider) Name() string { return "facebook" }

func (p *FacebookProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://www.facebook.com/v19.0/dialog/oauth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.RedirectURL),
		url.QueryEscape("email public_profile"), url.QueryEscape(state))
}

func (p *FacebookProvider) GetUserInfo(code string) (*UserInfo, error) {
	tokenURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/oauth/access_token?client_id=%s&client_secret=%s&redirect_uri=%s&code=%s",
		url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.ClientSecret),
		url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(code))

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := getJSONClient(p.client, tokenURL, nil, &token); err != nil {
		return nil, err
	}

	userURL := fmt.Sprintf(
		"https://graph.facebook.com/v19.0/me?fields=id,name,email,picture&access_token=%s",
		url.QueryEscape(token.AccessToken))
	var user struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := getJSONClient(p.client, userURL, nil, &user); err != nil {
		return nil, err
	}

	return &UserInfo{
		OpenID:      user.ID,
		Nickname:    user.Name,
		Avatar:      user.Picture.Data.URL,
		Email:       user.Email,
		AccessToken: token.AccessToken,
	}, nil
}
