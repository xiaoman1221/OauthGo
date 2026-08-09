package providers

import (
	"fmt"
	"net/url"
)

// WeiboProvider 微博登录
type WeiboProvider struct {
	cfg Config
}

func (p *WeiboProvider) Name() string { return "weibo" }

func (p *WeiboProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://api.weibo.com/oauth2/authorize?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *WeiboProvider) GetUserInfo(code string) (*UserInfo, error) {
	var token struct {
		AccessToken string `json:"access_token"`
		UID         string `json:"uid"`
	}
	if err := postForm("https://api.weibo.com/oauth2/access_token", url.Values{
		"client_id":     {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.cfg.RedirectURL},
	}, &token); err != nil {
		return nil, err
	}

	userURL := fmt.Sprintf("https://api.weibo.com/2/users/show.json?access_token=%s&uid=%s",
		url.QueryEscape(token.AccessToken), url.QueryEscape(token.UID))

	var user struct {
		ID          int64  `json:"id"`
		IDStr       string `json:"idstr"`
		ScreenName  string `json:"screen_name"`
		Name        string `json:"name"`
		AvatarLarge string `json:"avatar_large"`
	}
	if err := getJSON(userURL, &user); err != nil {
		return nil, err
	}

	uid := user.IDStr
	if uid == "" {
		uid = fmt.Sprintf("%d", user.ID)
	}
	nickname := user.ScreenName
	if nickname == "" {
		nickname = user.Name
	}
	return &UserInfo{
		OpenID:   uid,
		Nickname: nickname,
		Avatar:   user.AvatarLarge,
	}, nil
}
