package providers

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const alipayGateway = "https://openapi.alipay.com/gateway.do"

// AlipayProvider 支付宝登录（RSA2 签名）
type AlipayProvider struct {
	cfg Config
}

func (p *AlipayProvider) Name() string { return "alipay" }

func (p *AlipayProvider) GetAuthURL(state string) string {
	return fmt.Sprintf(
		"https://openauth.alipay.com/oauth2/publicAppAuthorize.htm?app_id=%s&scope=auth_user&redirect_uri=%s&state=%s",
		p.cfg.ClientID, url.QueryEscape(p.cfg.RedirectURL), url.QueryEscape(state))
}

func (p *AlipayProvider) GetUserInfo(code string) (*UserInfo, error) {
	privateKey := p.cfg.ExtraString("app_private_key")
	if privateKey == "" {
		return nil, fmt.Errorf("支付宝渠道缺少扩展配置 app_private_key（应用私钥）")
	}

	params := url.Values{
		"app_id":     {p.cfg.ClientID},
		"method":     {"alipay.system.oauth.token"},
		"charset":    {"utf-8"},
		"sign_type":  {"RSA2"},
		"timestamp":  {time.Now().Format("2006-01-02 15:04:05")},
		"version":    {"1.0"},
		"grant_type": {"authorization_code"},
		"code":       {code},
	}

	var tokenResp struct {
		AlipaySystemOauthTokenResponse struct {
			AccessToken string `json:"access_token"`
			UserID      string `json:"user_id"`
			ExpiresIn   int64  `json:"expires_in"`
			Code        string `json:"code"`
			Msg         string `json:"msg"`
		} `json:"alipay_system_oauth_token_response"`
	}
	if err := p.callGateway(params, privateKey, &tokenResp); err != nil {
		return nil, err
	}
	if tokenResp.AlipaySystemOauthTokenResponse.AccessToken == "" {
		return nil, fmt.Errorf("支付宝获取 access_token 失败: %s %s",
			tokenResp.AlipaySystemOauthTokenResponse.Code,
			tokenResp.AlipaySystemOauthTokenResponse.Msg)
	}

	accessToken := tokenResp.AlipaySystemOauthTokenResponse.AccessToken
	userID := tokenResp.AlipaySystemOauthTokenResponse.UserID

	infoParams := url.Values{
		"app_id":     {p.cfg.ClientID},
		"method":     {"alipay.user.info.share"},
		"charset":    {"utf-8"},
		"sign_type":  {"RSA2"},
		"timestamp":  {time.Now().Format("2006-01-02 15:04:05")},
		"version":    {"1.0"},
		"auth_token": {accessToken},
	}

	var userResp struct {
		AlipayUserInfoShareResponse struct {
			UserID   string `json:"user_id"`
			NickName string `json:"nick_name"`
			Avatar   string `json:"avatar"`
			Email    string `json:"email"`
			Code     string `json:"code"`
			Msg      string `json:"msg"`
		} `json:"alipay_user_info_share_response"`
	}
	if err := p.callGateway(infoParams, privateKey, &userResp); err != nil {
		return nil, err
	}
	infoResp := userResp.AlipayUserInfoShareResponse
	if infoResp.Code != "" && infoResp.Code != "10000" {
		return nil, fmt.Errorf("支付宝获取用户信息失败: %s %s", infoResp.Code, infoResp.Msg)
	}
	if userID == "" {
		userID = infoResp.UserID
	}

	return &UserInfo{
		OpenID:   userID,
		Nickname: infoResp.NickName,
		Avatar:   infoResp.Avatar,
		Email:    infoResp.Email,
	}, nil
}

// callGateway 调用支付宝网关（带 RSA2 签名）
func (p *AlipayProvider) callGateway(params url.Values, privateKey string, target interface{}) error {
	sign, err := alipaySign(params, privateKey)
	if err != nil {
		return err
	}
	params.Set("sign", sign)
	return postForm(alipayGateway, params, target)
}

func alipaySign(params url.Values, privateKey string) (string, error) {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := params.Get(k)
		if v == "" {
			continue
		}
		parts = append(parts, k+"="+v)
	}
	content := strings.Join(parts, "&")

	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return "", fmt.Errorf("解析支付宝应用私钥失败")
	}

	var key *rsa.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		key = k
	} else if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		key = k.(*rsa.PrivateKey)
	} else {
		return "", fmt.Errorf("解析支付宝应用私钥失败")
	}

	digest := sha256.Sum256([]byte(content))
	sign, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sign), nil
}
