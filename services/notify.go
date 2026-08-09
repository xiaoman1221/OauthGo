package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"time"
)

// ChannelConfig 通知渠道通用配置
type ChannelConfig struct {
	// webhook 配置
	URL string `json:"url"`

	// email 配置
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
}

// SendWebhook 发送 Webhook 通知
func SendWebhook(url, subject, content string) error {
	payload, err := json.Marshal(map[string]string{
		"subject": subject,
		"content": content,
	})
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook 返回异常状态码: %d", resp.StatusCode)
	}
	return nil
}

// SendEmail 发送邮件通知（SMTP）
func SendEmail(host string, port int, username, password, from string, to []string, subject, content string) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	auth := smtp.PlainAuth("", username, password, host)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		joinEmails(to), subject, content))
	return smtp.SendMail(addr, auth, from, to, msg)
}

func joinEmails(emails []string) string {
	buf := bytes.NewBuffer(nil)
	for i, e := range emails {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(e)
	}
	return buf.String()
}
