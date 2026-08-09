package services

import (
	"encoding/json"

	"OauthGo/models"
)

// ParseChannelConfig 解析通知渠道配置
func ParseChannelConfig(channel models.NotificationChannel) (ChannelConfig, error) {
	var cfg ChannelConfig
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// SendByChannel 按渠道类型发送通知
func SendByChannel(channel models.NotificationChannel, subject, content string) error {
	cfg, err := ParseChannelConfig(channel)
	if err != nil {
		return err
	}

	switch channel.Type {
	case "webhook":
		return SendWebhook(cfg.URL, subject, content)
	case "email":
		return SendEmail(cfg.SMTPHost, cfg.SMTPPort, cfg.Username, cfg.Password,
			cfg.From, cfg.To, subject, content)
	case "bark":
		return SendBark(cfg.BarkServer, cfg.BarkKey, cfg.BarkGroup, cfg.BarkSound, subject, content)
	default:
		return &UnsupportedChannelError{Type: channel.Type}
	}
}

// UnsupportedChannelError 不支持的渠道类型错误
type UnsupportedChannelError struct {
	Type string
}

func (e *UnsupportedChannelError) Error() string {
	return "不支持的通知渠道类型: " + e.Type
}
