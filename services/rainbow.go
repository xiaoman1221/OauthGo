package services

import (
	"fmt"
	"net/url"
)

// 彩虹聚合登录适配器
// 说明：对接彩虹聚合登录时需要替换为真实的接口地址与签名算法。

// RainbowClient 彩虹聚合登录客户端
type RainbowClient struct {
	AppID   string
	AppKey  string
	BaseURL string
}

// RainbowUser 彩虹返回的用户信息
type RainbowUser struct {
	OpenID   string
	Nickname string
	Avatar   string
	Email    string
}

// NewRainbowClient 创建彩虹客户端
func NewRainbowClient(appID, appKey string) *RainbowClient {
	return &RainbowClient{
		AppID:   appID,
		AppKey:  appKey,
		BaseURL: "https://www.rainbow.example.com",
	}
}

// LoginURL 构造跳转授权地址
func (c *RainbowClient) LoginURL(platform, state string) string {
	v := url.Values{}
	v.Set("appid", c.AppID)
	v.Set("platform", platform)
	v.Set("state", state)
	return fmt.Sprintf("%s/api/oauth/login?%s", c.BaseURL, v.Encode())
}

// GetUserInfo 通过 code 换取用户信息
func (c *RainbowClient) GetUserInfo(code string) (*RainbowUser, error) {
	// TODO: 调用彩虹聚合登录接口，根据 code 换取用户信息并验签
	return nil, fmt.Errorf("彩虹聚合登录接口地址尚未配置")
}
