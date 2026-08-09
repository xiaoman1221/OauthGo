package providers

import (
	"fmt"
	"net/url"
)

// WeComProvider 企业微信登录（支持企业自建 CorpApp 与第三方 ServiceApp）
// 扩展配置:
//   - agent_id:      企业自建应用的 AgentId
//   - corp_id:       企业 ID（第三方应用时为服务商的 corpid）
//   - login_type:    CorpApp（默认，企业自建）/ ServiceApp（第三方）
//   - suite_secret:  第三方应用时使用
type WeComProvider struct {
	cfg Config
}

func (p *WeComProvider) Name() string { return "wecom" }

func (p *WeComProvider) GetAuthURL(state string) string {
	loginType := p.cfg.ExtraString("login_type")
	if loginType == "" {
		loginType = "CorpApp"
	}

	redirect := url.QueryEscape(p.cfg.RedirectURL)
	return fmt.Sprintf(
		"https://login.work.weixin.qq.com/wwlogin/sso/login?login_type=%s&appid=%s&agentid=%s&redirect_uri=%s&state=%s",
		loginType, p.cfg.ClientID, p.cfg.ExtraString("agent_id"), redirect, url.QueryEscape(state))
}

func (p *WeComProvider) GetUserInfo(code string) (*UserInfo, error) {
	corpID := p.cfg.ClientID
	if v := p.cfg.ExtraString("corp_id"); v != "" {
		corpID = v
	}

	var token struct {
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
	}
	tokenURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		corpID, p.cfg.ClientSecret)
	if err := getJSON(tokenURL, &token); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, fmt.Errorf("企业微信获取 access_token 失败: %d %s", token.Errcode, token.Errmsg)
	}

	var userInfo struct {
		Errcode  int    `json:"errcode"`
		Errmsg   string `json:"errmsg"`
		UserID   string `json:"UserId"`
		OpenID   string `json:"OpenId"`
		DeviceID string `json:"DeviceId"`
	}
	userInfoURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/user/getuserinfo?access_token=%s&code=%s",
		url.QueryEscape(token.AccessToken), url.QueryEscape(code))
	if err := getJSON(userInfoURL, &userInfo); err != nil {
		return nil, err
	}
	if userInfo.UserID == "" && userInfo.OpenID == "" {
		return nil, fmt.Errorf("企业微信获取用户信息失败: %d %s", userInfo.Errcode, userInfo.Errmsg)
	}

	info := &UserInfo{OpenID: userInfo.UserID}
	if userInfo.UserID == "" {
		info.OpenID = userInfo.OpenID
		return info, nil
	}

	// 通过 UserId 获取成员详情
	var user struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
		Name    string `json:"name"`
		Avatar  string `json:"avatar"`
		Email   string `json:"email"`
	}
	userURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/user/get?access_token=%s&userid=%s",
		url.QueryEscape(token.AccessToken), url.QueryEscape(userInfo.UserID))
	if err := getJSON(userURL, &user); err == nil && user.Errcode == 0 {
		info.Nickname = user.Name
		info.Avatar = user.Avatar
		info.Email = user.Email
	}
	return info, nil
}
