package providers

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// QQProvider QQ 登录
type QQProvider struct {
	cfg Config
}

func (p *QQProvider) Name() string { return "qq" }

func (p *QQProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://graph.qq.com/oauth2.0/authorize?response_type=code&client_id=%s&redirect_uri=%s&state=%s&scope=get_user_info",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *QQProvider) GetUserInfo(code string) (*UserInfo, error) {
	tokenURL := fmt.Sprintf(
		"https://graph.qq.com/oauth2.0/token?grant_type=authorization_code&client_id=%s&client_secret=%s&code=%s&redirect_uri=%s",
		p.cfg.ClientID, p.cfg.ClientSecret, url.QueryEscape(code), url.QueryEscape(p.cfg.RedirectURL))

	raw, err := getText(tokenURL)
	if err != nil {
		return nil, err
	}
	params := parseQueryString(raw)
	accessToken := params.Get("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("QQ 获取 access_token 失败: %s", raw)
	}

	// 获取 openid（JSONP 格式）
	meURL := fmt.Sprintf("https://graph.qq.com/oauth2.0/me?access_token=%s", url.QueryEscape(accessToken))
	meRaw, err := getText(meURL)
	if err != nil {
		return nil, err
	}
	meJSON, err := extractJSONCallback(meRaw)
	if err != nil {
		return nil, err
	}
	var me struct {
		ClientID string `json:"client_id"`
		OpenID   string `json:"openid"`
		UnionID  string `json:"unionid"`
	}
	if err := json.Unmarshal([]byte(meJSON), &me); err != nil {
		return nil, err
	}

	userURL := fmt.Sprintf(
		"https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&openid=%s",
		url.QueryEscape(accessToken), p.cfg.ClientID, url.QueryEscape(me.OpenID))

	var userResp struct {
		Ret          int    `json:"ret"`
		Msg          string `json:"msg"`
		Nickname     string `json:"nickname"`
		FigureURLQQ1 string `json:"figureurl_qq_1"`
		FigureURLQQ2 string `json:"figureurl_qq_2"`
	}
	if err := getJSON(userURL, &userResp); err != nil {
		return nil, err
	}
	if userResp.Ret != 0 {
		return nil, fmt.Errorf("QQ 获取用户信息失败: %s", userResp.Msg)
	}

	avatar := userResp.FigureURLQQ2
	if avatar == "" {
		avatar = userResp.FigureURLQQ1
	}
	return &UserInfo{
		OpenID:      me.OpenID,
		UnionID:     me.UnionID,
		Nickname:    userResp.Nickname,
		Avatar:      avatar,
		AccessToken: accessToken,
	}, nil
}
