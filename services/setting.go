package services

import (
	"strconv"
	"sync"

	"OauthGo/database"
	"OauthGo/models"
)

// settingCache 系统设置缓存，全部设置持久化在数据库
var settingCache = struct {
	sync.RWMutex
	m map[string]string
}{m: map[string]string{}}

// InitSettings 加载数据库设置到缓存（服务启动时调用）
func InitSettings() {
	settingCache.Lock()
	defer settingCache.Unlock()
	settingCache.m = loadAllSettings()
}

// loadAllSettings 读取数据库中的全部设置（叠加默认值）
func loadAllSettings() map[string]string {
	m := make(map[string]string)
	for _, def := range models.DefaultSettingDefs() {
		m[def.Key] = def.Value
	}

	var settings []models.Setting
	database.DB.Find(&settings)
	for _, s := range settings {
		m[s.Key] = s.Value
	}
	return m
}

// AllSettings 返回全部设置
func AllSettings() map[string]string {
	settingCache.RLock()
	defer settingCache.RUnlock()
	cp := make(map[string]string, len(settingCache.m))
	for k, v := range settingCache.m {
		cp[k] = v
	}
	return cp
}

// GetSetting 读取字符串设置
func GetSetting(key, defaultValue string) string {
	settingCache.RLock()
	defer settingCache.RUnlock()
	if v, ok := settingCache.m[key]; ok {
		return v
	}
	return defaultValue
}

// GetBoolSetting 读取布尔设置
func GetBoolSetting(key string, defaultValue bool) bool {
	v := GetSetting(key, "")
	if v == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultValue
	}
	return b
}

// GetIntSetting 读取整数设置
func GetIntSetting(key string, defaultValue int) int {
	v := GetSetting(key, "")
	if v == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return i
}

// SetSetting 写入设置并更新缓存
func SetSetting(key, value string) {
	if _, ok := models.DefaultSettingDefsMap()[key]; !ok {
		return
	}

	var setting models.Setting
	if err := database.DB.Where("key = ?", key).First(&setting).Error; err != nil {
		database.DB.Create(&models.Setting{Key: key, Value: value})
	} else {
		database.DB.Model(&setting).Update("value", value)
	}

	settingCache.Lock()
	settingCache.m[key] = value
	settingCache.Unlock()
}
