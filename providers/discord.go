package providers

import (
	"fmt"
	"net/http"
	"net/url"
)

// DiscordProvider Discord 登录
type DiscordProvider struct {
	cfg    Config
	client *http.Client
}

func (p *DiscordProvider) Name() string { return "discord" }

func (p *DiscordProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://discord.com/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s",
		url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.RedirectURL),
		url.QueryEscape("identify email"), url.QueryEscape(state))
}

func (p *DiscordProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := postFormClient(p.client, "https://discord.com/api/oauth2/token", nil, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"redirect_uri":  {p.cfg.RedirectURL},
	}, &token); err != nil {
		return nil, err
	}

	var user struct {
		ID         string `json:"id"`
		Username   string `json:"username"`
		GlobalName string `json:"global_name"`
		Avatar     string `json:"avatar"`
		Email      string `json:"email"`
	}
	if err := getJSONClient(p.client, "https://discord.com/api/users/@me",
		map[string]string{"Authorization": "Bearer " + token.AccessToken}, &user); err != nil {
		return nil, err
	}

	nickname := user.GlobalName
	if nickname == "" {
		nickname = user.Username
	}
	avatar := ""
	if user.Avatar != "" {
		avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", user.ID, user.Avatar)
	}
	return &UserInfo{
		OpenID:      user.ID,
		Nickname:    nickname,
		Avatar:      avatar,
		Email:       user.Email,
		AccessToken: token.AccessToken,
	}, nil
}
