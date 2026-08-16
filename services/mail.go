package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPEnabled 是否启用了 SMTP 发信
func SMTPEnabled() bool {
	return GetBoolSetting("smtp_enabled", false)
}

// SendMail 通过配置的 SMTP 发送邮件
func SendMail(to, subject, body string) error {
	if !SMTPEnabled() {
		return fmt.Errorf("SMTP 未启用，请在系统设置中开启")
	}

	host := GetSetting("smtp_host", "")
	port := GetIntSetting("smtp_port", 465)
	username := GetSetting("smtp_username", "")
	password := GetSetting("smtp_password", "")
	from := GetSetting("smtp_from", username)
	fromName := GetSetting("smtp_from_name", "OauthGo")

	useTLS := GetBoolSetting("smtp_tls", true)

	if host == "" || from == "" {
		return fmt.Errorf("SMTP 配置不完整")
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	header := strings.Join([]string{
		"From: " + fromName + " <" + from + ">",
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	msg := []byte(header)

	// 端口 465 固定使用隐式 TLS；其余端口按 smtp_tls 选择 STARTTLS 或明文发送
	if port == 465 {
		return sendMailTLS(host, addr, username, password, from, to, msg)
	}
	return sendMailSTARTTLSOrPlain(host, addr, username, password, from, to, msg, useTLS)
}

// sendMailSTARTTLSOrPlain 发送邮件：可选 STARTTLS 升级（非 465 端口）
func sendMailSTARTTLSOrPlain(host, addr, username, password, from, to string, msg []byte, useTLS bool) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if useTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return err
			}
		}
	}
	if username != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func sendMailTLS(host, addr, username, password, from, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if username != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// SendTemplateMail 按模板渲染并发送邮件
func SendTemplateMail(to, templateName, subject string, data map[string]interface{}) error {
	body := RenderTemplate(GetSetting("email_template_"+templateName, ""), data)
	return SendMail(to, subject, body)
}

// SendTestMail 发送测试邮件（用于系统设置中的发信测试）
func SendTestMail(to string) error {
	body := "这是一封来自 OauthGo 的测试邮件。\n\n如果您收到此邮件，说明 SMTP 发信配置正常。"
	return SendMail(to, "OauthGo 邮件发送测试", body)
}
