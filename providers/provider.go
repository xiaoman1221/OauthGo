package providers

import "fmt"

// Config 第三方登录渠道配置
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Extra        map[string]interface{}
}

// ExtraString 读取扩展配置中的字符串值
func (c Config) ExtraString(key string) string {
	if c.Extra == nil {
		return ""
	}
	if v, ok := c.Extra[key].(string); ok {
		return v
	}
	return ""
}

// UserInfo 第三方渠道返回的用户信息
type UserInfo struct {
	OpenID      string
	UnionID     string
	Nickname    string
	Avatar      string
	Email       string
	AccessToken string
	Extra       map[string]interface{}
}

// Provider 第三方登录渠道适配器
type Provider interface {
	// Name 渠道唯一标识
	Name() string
	// GetAuthURL 生成跳转授权地址；无浏览器跳转的渠道（如微信小程序）返回空字符串
	GetAuthURL(state string) string
	// GetUserInfo 使用 code / js_code 换取用户信息
	GetUserInfo(code string) (*UserInfo, error)
}

// Meta 渠道元数据
type Meta struct {
	Name        string
	DisplayName string
	Category    string // social / enterprise
}

// All 返回所有支持的渠道
func All() []Meta {
	return []Meta{
		{Name: "wechat", DisplayName: "微信", Category: "social"},
		{Name: "wechat_miniprogram", DisplayName: "微信小程序", Category: "social"},
		{Name: "qq", DisplayName: "QQ", Category: "social"},
		{Name: "weibo", DisplayName: "微博", Category: "social"},
		{Name: "gitee", DisplayName: "Gitee", Category: "social"},
		{Name: "douyin", DisplayName: "抖音", Category: "social"},
		{Name: "baidu", DisplayName: "百度", Category: "social"},
		{Name: "alipay", DisplayName: "支付宝", Category: "social"},
		{Name: "dingtalk", DisplayName: "钉钉", Category: "enterprise"},
		{Name: "wecom", DisplayName: "企业微信", Category: "enterprise"},
		{Name: "lark", DisplayName: "飞书", Category: "enterprise"},
		{Name: "infoflow", DisplayName: "如流", Category: "enterprise"},
	}
}

// FindMeta 查找渠道元数据
func FindMeta(name string) (*Meta, bool) {
	for _, m := range All() {
		if m.Name == name {
			return &m, true
		}
	}
	return nil, false
}

// New 根据渠道名创建适配器实例
func New(name string, cfg Config) (Provider, error) {
	switch name {
	case "wechat":
		return &WeChatProvider{cfg: cfg}, nil
	case "wechat_miniprogram":
		return &WeChatMiniProgramProvider{cfg: cfg}, nil
	case "qq":
		return &QQProvider{cfg: cfg}, nil
	case "weibo":
		return &WeiboProvider{cfg: cfg}, nil
	case "gitee":
		return &GiteeProvider{cfg: cfg}, nil
	case "douyin":
		return &DouyinProvider{cfg: cfg}, nil
	case "baidu":
		return &BaiduProvider{cfg: cfg}, nil
	case "alipay":
		return &AlipayProvider{cfg: cfg}, nil
	case "dingtalk":
		return &DingTalkProvider{cfg: cfg}, nil
	case "wecom":
		return &WeComProvider{cfg: cfg}, nil
	case "lark":
		return &LarkProvider{cfg: cfg}, nil
	case "infoflow":
		return &InfoflowProvider{cfg: cfg}, nil
	default:
		return nil, fmt.Errorf("不支持的登录渠道: %s", name)
	}
}
