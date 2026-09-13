package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/utils"
)

// GenerateVerifyCode 生成验证码并通过对应渠道发送
func GenerateVerifyCode(scope, account string) error {
	// 限频：同一 scope+account 在冷却期内不得重复发送，防止短信/邮件滥用
	var last models.VerificationCode
	if err := database.DB.Where("scope = ? AND account = ? AND used = ?", scope, account, false).
		Order("id desc").First(&last).Error; err == nil && time.Since(last.CreatedAt) < 60*time.Second {
		return fmt.Errorf("请求过于频繁，请稍后再试")
	}
	// 作废该账号此前未使用的验证码，避免旧码堆积
	database.DB.Where("scope = ? AND account = ? AND used = ?", scope, account, false).
		Update("used", true)

	length := GetIntSetting("code_length", 6)
	if length <= 0 || length > 16 {
		length = 6
	}
	expire := time.Duration(GetIntSetting("code_expire_minutes", 10)) * time.Minute

	code := randomDigits(length)
	record := models.VerificationCode{
		Scope:     scope,
		Account:   account,
		Code:      code,
		ExpiresAt: time.Now().Add(expire),
	}
	if err := database.DB.Create(&record).Error; err != nil {
		return err
	}

	data := map[string]interface{}{
		"code":           code,
		"site_name":      GetSetting("site_name", "OauthGo"),
		"site_url":       GetSetting("site_url", ""),
		"expire_minutes": fmt.Sprintf("%d", GetIntSetting("code_expire_minutes", 10)),
		"account":        account,
	}

	if strings.Contains(account, "@") {
		if !SMTPEnabled() {
			return fmt.Errorf("SMTP 未启用，无法发送邮件验证码")
		}
		subject := GetSetting("site_name", "OauthGo") + " 验证码"
		return SendTemplateMail(account, scope, subject, data)
	}
	return SendVerifyCodeSMS(account, code)
}

// VerifyVerifyCode 校验验证码（成功后置为已使用）
func VerifyVerifyCode(scope, account, code string) error {
	if code == "" {
		return fmt.Errorf("验证码不能为空")
	}

	var record models.VerificationCode
	err := database.DB.Where("scope = ? AND account = ? AND code = ? AND used = ?", scope, account, code, false).
		Order("id desc").First(&record).Error
	if err != nil {
		return fmt.Errorf("验证码错误或已使用")
	}
	if time.Now().After(record.ExpiresAt) {
		return fmt.Errorf("验证码已过期")
	}

	// 原子占用，防止并发使用同一验证码
	occupy := database.DB.Model(&models.VerificationCode{}).Where("id = ? AND used = ?", record.ID, false).Update("used", true)
	if occupy.Error != nil {
		return fmt.Errorf("验证码校验失败")
	}
	if occupy.RowsAffected == 0 {
		return fmt.Errorf("验证码已使用")
	}
	return nil
}

// randomDigits 生成指定长度的数字验证码
func randomDigits(length int) string {
	var sb strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return fmt.Sprintf("%0*d", length, time.Now().UnixNano()%1000000)
		}
		sb.WriteString(n.String())
	}
	return sb.String()
}

// ChangePassword 修改用户密码（重置密码 / 忘记密码共用）
func ChangePassword(userID uint, newPassword string) error {
	minLen := GetIntSetting("password_min_length", 6)
	if len(newPassword) < minLen {
		return fmt.Errorf("密码长度不能少于 %d 位", minLen)
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	hash, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := database.DB.Model(&user).Updates(map[string]interface{}{
		"password":     hash,
		"password_set": true,
	}).Error; err != nil {
		return err
	}
	// 密码变更后吊销该用户全部登录会话，强制重新登录
	DeleteUserSessions(userID)
	return nil
}
