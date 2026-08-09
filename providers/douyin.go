package providers

import (
	"fmt"
	"net/url"
)

// DouyinProvider 抖音登录
type DouyinProvider struct {
	cfg Config
}

func (p *DouyinProvider) Name() string { return "douyin" }

func (p *DouyinProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://open.douyin.com/platform/oauth/connect/?client_key=%s&response_type=code&scope=user_info&redirect_uri=%s&state=%s",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *DouyinProvider) GetUserInfo(code string) (*UserInfo, error) {
	var tokenResp struct {
		Data struct {
			OpenID      string `json:"open_id"`
			AccessToken string `json:"access_token"`
			ExpiresIn   int64  `json:"expires_in"`
		} `json:"data"`
	}
	if err := postForm("https://open.douyin.com/oauth/access_token/", url.Values{
		"client_key":    {p.cfg.ClientID},
		"client_secret": {p.cfg.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}, &tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.Data.AccessToken == "" {
		return nil, fmt.Errorf("抖音获取 access_token 失败")
	}

	userURL := fmt.Sprintf("https://open.douyin.com/oauth/userinfo/?access_token=%s&open_id=%s",
		url.QueryEscape(tokenResp.Data.AccessToken), url.QueryEscape(tokenResp.Data.OpenID))

	var userResp struct {
		Data struct {
			OpenID   string `json:"open_id"`
			Nickname string `json:"nickname"`
			Avatar   string `json:"avatar"`
			UnionID  string `json:"union_id"`
		} `json:"data"`
	}
	if err := getJSON(userURL, &userResp); err != nil {
		return nil, err
	}

	return &UserInfo{
		OpenID:   userResp.Data.OpenID,
		UnionID:  userResp.Data.UnionID,
		Nickname: userResp.Data.Nickname,
		Avatar:   userResp.Data.Avatar,
	}, nil
}
