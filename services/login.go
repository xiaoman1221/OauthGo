package services

import (
	"errors"
	"strings"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/utils"
)

// RecordLogin 记录一次登录行为
func RecordLogin(appID uint, appName string, record models.LoginRecord) error {
	record.AppID = appID
	record.AppName = appName
	if record.LoginTime.IsZero() {
		record.LoginTime = time.Now()
	}
	return database.DB.Create(&record).Error
}

// PlatformLoginCodeTTL 平台登录一次性码有效期（签发后由前端立即兑换，2 分钟足够）
const PlatformLoginCodeTTL = 2 * time.Minute

// IssuePlatformLoginCode 为用户签发一次性登录码（前端以 code 回跳，兑换 JWT，避免 JWT 进入 URL）
func IssuePlatformLoginCode(userID uint) (string, error) {
	code := strings.ToUpper(utils.RandomString(32))
	record := models.PlatformLoginCode{
		Code:      code,
		UserID:    userID,
		ExpiresAt: time.Now().Add(PlatformLoginCodeTTL),
	}
	if err := database.DB.Create(&record).Error; err != nil {
		return "", err
	}
	return code, nil
}

// ExchangePlatformLoginCode 兑换一次性登录码（一次性，原子占用），返回对应用户
func ExchangePlatformLoginCode(code string) (*models.User, error) {
	if code == "" {
		return nil, errors.New("缺少 code")
	}
	var record models.PlatformLoginCode
	if err := database.DB.Where("code = ?", code).First(&record).Error; err != nil {
		return nil, errors.New("code 无效")
	}
	if record.Used {
		return nil, errors.New("code 已使用")
	}
	if time.Now().After(record.ExpiresAt) {
		return nil, errors.New("code 已过期")
	}
	// 原子占用，防止并发兑换两次
	occupy := database.DB.Model(&models.PlatformLoginCode{}).Where("id = ? AND used = ?", record.ID, false).Update("used", true)
	if occupy.Error != nil {
		return nil, errors.New("code 状态更新失败")
	}
	if occupy.RowsAffected == 0 {
		return nil, errors.New("code 已使用")
	}
	var user models.User
	if err := database.DB.First(&user, record.UserID).Error; err != nil {
		return nil, errors.New("用户不存在")
	}
	return &user, nil
}
