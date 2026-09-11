package services

import (
	"log"
	"time"

	"OauthGo/database"
	"OauthGo/models"
)

// startExpirySweeper 定期清理已过期的一次性码、OAuth 授权码/令牌与验证码，
// 防止数据表无限膨胀（授权码/令牌均为短生命周期，无保留价值）。
func startExpirySweeper() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			SweepExpiredData()
		}
	}()
}

func init() {
	startExpirySweeper()
}

// SweepExpiredData 删除全部已过期数据（可手动调用，便于测试）
func SweepExpiredData() {
	if database.DB == nil {
		return
	}
	now := time.Now()
	if err := database.DB.Where("expires_at < ?", now).Delete(&models.VerificationCode{}).Error; err != nil {
		log.Printf("[sweep] 清理验证码失败: %v", err)
	}
	if err := database.DB.Where("used = ? OR expires_at < ?", true, now).Delete(&models.LoginCode{}).Error; err != nil {
		log.Printf("[sweep] 清理授权码失败: %v", err)
	}
	if err := database.DB.Where("used = ? OR expires_at < ?", true, now).Delete(&models.OAuthCode{}).Error; err != nil {
		log.Printf("[sweep] 清理 OAuth 授权码失败: %v", err)
	}
	if err := database.DB.Where("expires_at < ?", now).Delete(&models.OAuthAccessToken{}).Error; err != nil {
		log.Printf("[sweep] 清理访问令牌失败: %v", err)
	}
	if err := database.DB.Where("expires_at < ?", now).Delete(&models.OAuthRefreshToken{}).Error; err != nil {
		log.Printf("[sweep] 清理刷新令牌失败: %v", err)
	}
	if err := database.DB.Where("used = ? OR expires_at < ?", true, now).Delete(&models.PlatformLoginCode{}).Error; err != nil {
		log.Printf("[sweep] 清理平台登录码失败: %v", err)
	}
}
