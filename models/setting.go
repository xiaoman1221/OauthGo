package models

import "time"

// Setting 系统设置（键值对）
type Setting struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Key         string    `gorm:"uniqueIndex;size:128;not null" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SettingDef 系统设置项定义（默认值与分组元数据）
type SettingDef struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	Group       string `json:"group"`     // site / security / smtp / sms / template
	Sensitive   bool   `json:"sensitive"` // 敏感字段，前端以密码框展示
}

// DefaultSettingDefs 全部系统设置项默认值
func DefaultSettingDefs() []SettingDef {
	return []SettingDef{
		// 站点设置
		{Key: "site_name", Value: "OauthGo", Description: "站点名称", Group: "site"},
		{Key: "site_url", Value: "", Description: "站点地址", Group: "site"},
		{Key: "site_logo", Value: "", Description: "站点 Logo 地址", Group: "site"},
		{Key: "site_desc", Value: "统一授权管理平台", Description: "站点描述", Group: "site"},
		{Key: "register_enabled", Value: "1", Description: "开放注册", Group: "site"},
		{Key: "register_email_verify", Value: "0", Description: "注册邮箱验证", Group: "site"},
		{Key: "default_role", Value: "user", Description: "默认注册角色", Group: "site"},
		// 普通用户相关设置
		{Key: "user_max_apps", Value: "5", Description: "普通用户最多可创建的应用数量", Group: "site"},
		// 登录页背景设置（前端展示并可配置）
		{Key: "login_bg_mode", Value: "color", Description: "登录页背景类型 (color|image|bing)", Group: "site"},
		{Key: "login_bg_color", Value: "#1f4037", Description: "登录页背景纯色（hex）", Group: "site"},
		{Key: "login_bg_image_url", Value: "", Description: "登录页背景图片 URL", Group: "site"},

		// 安全设置
		{Key: "password_min_length", Value: "6", Description: "密码最小长度", Group: "security"},
		{Key: "code_length", Value: "6", Description: "验证码长度", Group: "security"},
		{Key: "code_expire_minutes", Value: "10", Description: "验证码有效期（分钟）", Group: "security"},

		// 头像设置
		{Key: "avatar_source", Value: "auto", Description: "头像来源（auto：QQ邮箱自动用QQ头像，其余用Gravatar / qq：仅QQ邮箱用QQ头像 / gravatar：全部使用Gravatar）", Group: "avatar"},
		{Key: "gravatar_mirror_enabled", Value: "1", Description: "启用 Gravatar 镜像站", Group: "avatar"},
		{Key: "gravatar_mirror", Value: "https://cravatar.cn/avatar", Description: "Gravatar 镜像站地址（国内默认 Cravatar，留空则使用官方 gravatar.com）", Group: "avatar"},

		// SOCKS5 代理设置（用于境外登录渠道，需在各渠道中开启「使用代理」）
		{Key: "proxy_enabled", Value: "0", Description: "启用 SOCKS5 代理", Group: "proxy"},
		{Key: "proxy_addr", Value: "", Description: "SOCKS5 代理地址（host:port）", Group: "proxy"},
		{Key: "proxy_username", Value: "", Description: "代理用户名（可选）", Group: "proxy"},
		{Key: "proxy_password", Value: "", Description: "代理密码（可选）", Group: "proxy", Sensitive: true},

		// SMTP 邮件设置
		{Key: "smtp_enabled", Value: "0", Description: "启用 SMTP 发信", Group: "smtp"},
		{Key: "smtp_host", Value: "", Description: "SMTP 服务器", Group: "smtp"},
		{Key: "smtp_port", Value: "465", Description: "SMTP 端口（465 隐式 TLS / 587 STARTTLS）", Group: "smtp"},
		{Key: "smtp_username", Value: "", Description: "SMTP 账号", Group: "smtp"},
		{Key: "smtp_password", Value: "", Description: "SMTP 密码", Group: "smtp", Sensitive: true},
		{Key: "smtp_from", Value: "", Description: "发件人邮箱", Group: "smtp"},
		{Key: "smtp_from_name", Value: "OauthGo", Description: "发件人名称", Group: "smtp"},
		{Key: "smtp_tls", Value: "1", Description: "使用 TLS 加密", Group: "smtp"},

		// 短信设置
		{Key: "sms_provider", Value: "none", Description: "短信服务商", Group: "sms"},
		{Key: "sms_access_key_id", Value: "", Description: "AccessKey ID", Group: "sms"},
		{Key: "sms_access_key_secret", Value: "", Description: "AccessKey Secret", Group: "sms", Sensitive: true},
		{Key: "sms_region_id", Value: "cn-hangzhou", Description: "短信地域", Group: "sms"},
		{Key: "sms_sign_name", Value: "", Description: "短信签名", Group: "sms"},
		{Key: "sms_aliyun_template_code", Value: "", Description: "阿里云验证码模板CODE", Group: "sms"},
		{Key: "sms_tencent_sdk_app_id", Value: "", Description: "腾讯云 SDK AppID", Group: "sms"},
		{Key: "sms_tencent_template_id", Value: "", Description: "腾讯云验证码模板ID", Group: "sms"},
		{Key: "smsbao_username", Value: "", Description: "短信宝账号", Group: "sms"},
		{Key: "smsbao_password", Value: "", Description: "短信宝密码", Group: "sms", Sensitive: true},

		// 邮件模板
		{Key: "email_template_register", Value: "【{{site_name}}】您的注册验证码是 {{code}}，请在 {{expire_minutes}} 分钟内完成验证，请勿泄露给他人。", Description: "注册验证邮件模板", Group: "template"},
		{Key: "email_template_reset", Value: "【{{site_name}}】您的密码重置验证码是 {{code}}，请在 {{expire_minutes}} 分钟内完成重置，请勿泄露给他人。", Description: "找回密码邮件模板", Group: "template"},
		{Key: "email_template_bind", Value: "【{{site_name}}】您正在绑定邮箱 {{account}}，验证码是 {{code}}，请在 {{expire_minutes}} 分钟内完成绑定，请勿泄露给他人。", Description: "绑定邮箱验证邮件模板", Group: "template"},
		{Key: "email_template_welcome", Value: "【{{site_name}}】欢迎您，{{username}}！您的账号已注册成功，请妥善保管账号信息。", Description: "欢迎邮件模板", Group: "template"},
	}
}

// DefaultSettingDefsMap 返回默认设置定义（key -> def）
func DefaultSettingDefsMap() map[string]SettingDef {
	defs := DefaultSettingDefs()
	m := make(map[string]SettingDef, len(defs))
	for _, d := range defs {
		m[d.Key] = d
	}
	return m
}
