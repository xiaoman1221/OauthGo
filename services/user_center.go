package services

import (
	"errors"
	"sync"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/providers"
	"OauthGo/utils"
)

// BindSession 一次「登录账号绑定」会话
type BindSession struct {
	UserID    uint
	Provider  string
	CreatedAt time.Time
}

var bindSessionStore = struct {
	sync.RWMutex
	m map[string]BindSession
}{m: map[string]BindSession{}}

const bindSessionTTL = 10 * time.Minute

// startBindSessionJanitor 定期清理过期会话，避免未完成的绑定会话无限累积
func startBindSessionJanitor() {
	go func() {
		ticker := time.NewTicker(bindSessionTTL)
		defer ticker.Stop()
		for range ticker.C {
			bindSessionStore.Lock()
			for s, sess := range bindSessionStore.m {
				if time.Since(sess.CreatedAt) > bindSessionTTL {
					delete(bindSessionStore.m, s)
				}
			}
			bindSessionStore.Unlock()
		}
	}()
}

func init() {
	startBindSessionJanitor()
}

// CreateBindSession 创建绑定会话并返回 state（由渠道回调带回）
func CreateBindSession(userID uint, providerName string) string {
	state := utils.RandomString(32)
	bindSessionStore.Lock()
	bindSessionStore.m[state] = BindSession{
		UserID:    userID,
		Provider:  providerName,
		CreatedAt: time.Now(),
	}
	bindSessionStore.Unlock()
	return state
}

// ResolveBindSession 按 state 查找并消费绑定会话（一次性）
func ResolveBindSession(state string) (BindSession, bool) {
	bindSessionStore.Lock()
	defer bindSessionStore.Unlock()
	s, ok := bindSessionStore.m[state]
	if !ok {
		return BindSession{}, false
	}
	delete(bindSessionStore.m, state)
	if time.Since(s.CreatedAt) > bindSessionTTL {
		return BindSession{}, false
	}
	return s, true
}

// BindProviderAccount 将第三方账号绑定到当前用户
func BindProviderAccount(userID uint, providerName string, info *providers.UserInfo) error {
	if info.OpenID == "" && info.UnionID == "" {
		return errors.New("未获取到第三方用户唯一标识")
	}

	// 优先依据 union_id 校验（若存在），否则按 open_id
	if info.UnionID != "" {
		var cnt int64
		database.DB.Model(&models.ProviderAccount{}).
			Where("provider = ? AND union_id = ?", providerName, info.UnionID).Count(&cnt)
		if cnt > 0 {
			var account models.ProviderAccount
			database.DB.Where("provider = ? AND union_id = ?", providerName, info.UnionID).First(&account)
			if account.UserID == userID {
				return errors.New("该渠道账号已绑定")
			}
			return errors.New("该第三方账号已绑定其他用户")
		}
	}

	if info.OpenID != "" {
		var cnt int64
		database.DB.Model(&models.ProviderAccount{}).
			Where("provider = ? AND open_id = ?", providerName, info.OpenID).Count(&cnt)
		if cnt > 0 {
			var account models.ProviderAccount
			database.DB.Where("provider = ? AND open_id = ?", providerName, info.OpenID).First(&account)
			if account.UserID == userID {
				return errors.New("该渠道账号已绑定")
			}
			return errors.New("该第三方账号已绑定其他用户")
		}
	}

	account := models.ProviderAccount{
		UserID:   userID,
		Provider: providerName,
		OpenID:   info.OpenID,
		UnionID:  info.UnionID,
		Nickname: info.Nickname,
		Avatar:   info.Avatar,
		Email:    info.Email,
	}
	if err := database.DB.Create(&account).Error; err != nil {
		return err
	}
	return nil
}

// UnbindProviderAccount 解绑第三方账号。
// 用户未设置密码且仅剩一个绑定渠道时禁止解绑，避免账号无法登录。
func UnbindProviderAccount(userID uint, providerName string) error {
	var account models.ProviderAccount
	if err := database.DB.Where("user_id = ? AND provider = ?", userID, providerName).First(&account).Error; err != nil {
		return errors.New("未绑定该渠道")
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	var count int64
	database.DB.Model(&models.ProviderAccount{}).Where("user_id = ?", userID).Count(&count)
	if !user.PasswordSet && count <= 1 {
		return errors.New("当前账号未设置密码，请先设置密码或绑定其他渠道")
	}

	if err := database.DB.Delete(&account).Error; err != nil {
		return err
	}
	return nil
}
