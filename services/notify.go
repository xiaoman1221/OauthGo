package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
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

	// bark 配置
	BarkServer string `json:"bark_server"`
	BarkKey    string `json:"bark_key"`
	BarkGroup  string `json:"bark_group"`
	BarkSound  string `json:"bark_sound"`
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

// SendBark 发送 Bark 推送通知（iOS）
// server 为空时使用官方服务器 https://api.day.app
func SendBark(server, key, group, sound, title, body string) error {
	if key == "" {
		return fmt.Errorf("Bark Device Key 未配置")
	}
	if server == "" {
		server = "https://api.day.app"
	}
	server = strings.TrimRight(server, "/")

	form := url.Values{}
	form.Set("title", title)
	form.Set("body", body)
	form.Set("device_key", key)
	if group != "" {
		form.Set("group", group)
	}
	if sound != "" {
		form.Set("sound", sound)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(server+"/push", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Bark 服务器返回异常状态码: %d: %s", resp.StatusCode, truncateStr(string(b)))
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if result.Code != 200 {
		return fmt.Errorf("Bark 推送失败: %s", result.Message)
	}
	return nil
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
