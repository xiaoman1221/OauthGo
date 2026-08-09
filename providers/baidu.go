package providers

import (
	"fmt"
	"net/url"
)

// BaiduProvider 百度登录
type BaiduProvider struct {
	cfg Config
}

func (p *BaiduProvider) Name() string { return "baidu" }

func (p *BaiduProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://openapi.baidu.com/oauth/2.0/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=basic&state=%s",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *BaiduProvider) GetUserInfo(code string) (*UserInfo, error) {
	tokenURL := fmt.Sprintf(
		"https://openapi.baidu.com/oauth/2.0/token?grant_type=authorization_code&code=%s&client_id=%s&client_secret=%s&redirect_uri=%s",
		url.QueryEscape(code), p.cfg.ClientID, p.cfg.ClientSecret, url.QueryEscape(p.cfg.RedirectURL))

	var token struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := getJSON(tokenURL, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("百度获取 access_token 失败: %s %s", token.Error, token.ErrorDesc)
	}

	userURL := fmt.Sprintf("https://openapi.baidu.com/rest/2.0/passport/users/getInfo?access_token=%s",
		url.QueryEscape(token.AccessToken))

	var user struct {
		ErrorCode int64  `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
		OpenID    string `json:"openid"`
		UserName  string `json:"username"`
		Portrait  string `json:"portrait"`
	}
	if err := getJSON(userURL, &user); err != nil {
		return nil, err
	}
	if user.ErrorCode != 0 {
		return nil, fmt.Errorf("百度获取用户信息失败: %s", user.ErrorMsg)
	}

	return &UserInfo{
		OpenID:   user.OpenID,
		Nickname: user.UserName,
		Avatar:   fmt.Sprintf("https://himg.bdimg.com/sys/portrait/item/%s.jpg", user.Portrait),
	}, nil
}
