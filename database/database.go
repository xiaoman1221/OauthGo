package database

import (
	"log"
	"strings"

	"OauthGo/config"
	"OauthGo/models"
	"OauthGo/providers"
	"OauthGo/utils"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库实例
var DB *gorm.DB

// Init 初始化数据库连接并执行迁移
func Init() {
	db, err := gorm.Open(sqlite.Open(config.AppConfig.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	DB = db

	migrate()
	seedAdmin()
	seedProviders()
	seedSettings()
}

func migrate() {
	// 仅在本次迁移新增 password_set 列时回填历史用户，避免误改第三方注册用户
	hasPasswordSetCol := DB.Migrator().HasColumn(&models.User{}, "password_set")
	hasDomainsCol := DB.Migrator().HasColumn(&models.App{}, "domains")

	if err := DB.AutoMigrate(
		&models.User{},
		&models.App{},
		&models.LoginCode{},
		&models.LoginRecord{},
		&models.Setting{},
		&models.VerificationCode{},
		&models.Provider{},
		&models.ProviderAccount{},
	); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}
	dropLegacyIndexes()
	backfillAppCredentials()
	if !hasPasswordSetCol {
		backfillUserProfile()
	}
	if !hasDomainsCol {
		backfillAppDomains()
	}
}

// backfillAppDomains 将旧版 redirect_url / callback_url 迁移为回调白名单域名，并移除旧列
func backfillAppDomains() {
	var rows []struct {
		ID          uint
		RedirectURL string
		CallbackURL string
	}
	if err := DB.Raw("SELECT id, redirect_url, callback_url FROM apps").Scan(&rows).Error; err != nil {
		return // 旧列不存在，跳过
	}
	for _, r := range rows {
		list := []string{}
		for _, u := range []string{r.RedirectURL, r.CallbackURL} {
			if d := utils.ExtractDomain(u); d != "" {
				list = append(list, d)
			}
		}
		if len(list) > 0 {
			DB.Model(&models.App{}).Where("id = ?", r.ID).Update("domains", strings.Join(list, "\n"))
		}
	}
	m := DB.Migrator()
	if m.HasColumn(&models.App{}, "redirect_url") {
		_ = m.DropColumn(&models.App{}, "redirect_url")
	}
	if m.HasColumn(&models.App{}, "callback_url") {
		_ = m.DropColumn(&models.App{}, "callback_url")
	}
}

// backfillAppCredentials 为旧版本应用补齐 AppID / AppKey
func backfillAppCredentials() {
	var apps []models.App
	DB.Find(&apps)
	for _, app := range apps {
		if app.AppID == "" || app.AppKey == "" {
			DB.Model(&app).Updates(map[string]interface{}{
				"app_id":  orValue(app.AppID, strings.ToLower(utils.RandomString(16))),
				"app_key": orValue(app.AppKey, utils.RandomString(32)),
			})
		}
	}
}

func orValue(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}

// backfillUserProfile 为旧版本用户补齐昵称与密码标记
func backfillUserProfile() {
	var users []models.User
	DB.Where("nickname = ''").Find(&users)
	for _, user := range users {
		DB.Model(&user).Update("nickname", user.Username)
	}
	DB.Model(&models.User{}).Where("password != '' AND password_set = ?", false).
		Update("password_set", true)
}

// dropLegacyIndexes 移除早期版本遗留的唯一索引（空字符串会造成冲突，唯一性由业务层保证）
func dropLegacyIndexes() {
	m := DB.Migrator()
	for _, idx := range []string{"idx_users_email", "idx_users_phone"} {
		if m.HasIndex(&models.User{}, idx) {
			if err := m.DropIndex(&models.User{}, idx); err != nil {
				log.Printf("删除遗留索引 %s 失败: %v", idx, err)
			}
		}
	}
}

// seedAdmin 首次启动创建默认管理员账号 admin / 123456
func seedAdmin() {
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	hash, err := utils.HashPassword("123456")
	if err != nil {
		log.Fatalf("创建默认管理员失败: %v", err)
	}
	user := models.User{
		Username:    "admin",
		Password:    hash,
		PasswordSet: true,
		Role:        "admin",
	}
	if err := DB.Create(&user).Error; err != nil {
		log.Fatalf("创建默认管理员失败: %v", err)
	}
	log.Println("已创建默认管理员账号: admin / 123456（请登录后立即修改）")
}

// seedSettings 播种默认系统设置
func seedSettings() {
	for _, def := range models.DefaultSettingDefs() {
		var count int64
		DB.Model(&models.Setting{}).Where("key = ?", def.Key).Count(&count)
		if count > 0 {
			continue
		}
		if err := DB.Create(&models.Setting{
			Key:         def.Key,
			Value:       def.Value,
			Description: def.Description,
		}).Error; err != nil {
			log.Fatalf("播种系统设置失败: %v", err)
		}
	}
}

// seedProviders 预置所有支持的第三方登录渠道记录
func seedProviders() {
	for i, meta := range providers.All() {
		var count int64
		DB.Model(&models.Provider{}).Where("name = ?", meta.Name).Count(&count)
		if count > 0 {
			continue
		}
		provider := models.Provider{
			Name:        meta.Name,
			DisplayName: meta.DisplayName,
			Category:    meta.Category,
			Enabled:     false,
			Sort:        i,
		}
		if err := DB.Create(&provider).Error; err != nil {
			log.Fatalf("预置登录渠道失败: %v", err)
		}
	}
}
