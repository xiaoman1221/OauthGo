package providers

import (
	"fmt"
	"net/url"
)

// WeChatProvider 微信开放平台扫码登录
type WeChatProvider struct {
	cfg Config
}

func (p *WeChatProvider) Name() string { return "wechat" }

func (p *WeChatProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *WeChatProvider) GetUserInfo(code string) (*UserInfo, error) {
	tokenURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		p.cfg.ClientID, p.cfg.ClientSecret, url.QueryEscape(code))

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
		UnionID     string `json:"unionid"`
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
	}
	if err := getJSON(tokenURL, &tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.Errcode != 0 {
		return nil, fmt.Errorf("微信获取 access_token 失败: %s", tokenResp.Errmsg)
	}

	userURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s",
		url.QueryEscape(tokenResp.AccessToken), url.QueryEscape(tokenResp.OpenID))

	var userResp struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		Nickname   string `json:"nickname"`
		HeadImgURL string `json:"headimgurl"`
		Errcode    int    `json:"errcode"`
		Errmsg     string `json:"errmsg"`
	}
	if err := getJSON(userURL, &userResp); err != nil {
		return nil, err
	}
	if userResp.Errcode != 0 {
		return nil, fmt.Errorf("微信获取用户信息失败: %s", userResp.Errmsg)
	}

	return &UserInfo{
		OpenID:   userResp.OpenID,
		UnionID:  userResp.UnionID,
		Nickname: userResp.Nickname,
		Avatar:   userResp.HeadImgURL,
	}, nil
}

// WeChatMiniProgramProvider 微信小程序登录（无浏览器跳转，前端传入 js_code）
type WeChatMiniProgramProvider struct {
	cfg Config
}

func (p *WeChatMiniProgramProvider) Name() string { return "wechat_miniprogram" }

func (p *WeChatMiniProgramProvider) GetAuthURL(state string) string { return "" }

func (p *WeChatMiniProgramProvider) GetUserInfo(jsCode string) (*UserInfo, error) {
	codeURL := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		p.cfg.ClientID, p.cfg.ClientSecret, url.QueryEscape(jsCode))

	var resp struct {
		OpenID     string `json:"openid"`
		SessionKey string `json:"session_key"`
		UnionID    string `json:"unionid"`
		Errcode    int    `json:"errcode"`
		Errmsg     string `json:"errmsg"`
	}
	if err := getJSON(codeURL, &resp); err != nil {
		return nil, err
	}
	if resp.Errcode != 0 {
		return nil, fmt.Errorf("微信小程序 code2session 失败: %s", resp.Errmsg)
	}

	return &UserInfo{
		OpenID:  resp.OpenID,
		UnionID: resp.UnionID,
	}, nil
}
