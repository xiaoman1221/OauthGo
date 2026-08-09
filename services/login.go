package services

import (
	"time"

	"OauthGo/database"
	"OauthGo/models"
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
