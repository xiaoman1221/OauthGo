package services

import (
	"log"
	"strings"

	"OauthGo/config"
	"OauthGo/utils"
)

// JWTSecretKeySetting 控制台 JWT 签名密钥的持久化设置项（未配置环境变量时使用）
const JWTSecretKeySetting = "jwt_secret_key"

// EnsureJWTKey 解析控制台 JWT 签名密钥：
//  1. 环境变量 JWT_KEY 非空时优先使用；
//  2. 否则读取数据库持久化密钥；
//  3. 两者皆无则生成 256 位随机密钥并持久化，彻底消除使用公开默认密钥
//     导致伪造 admin 令牌的风险。
//
// 依赖数据库与设置缓存，需在 database.Init / InitSettings 之后调用。
func EnsureJWTKey() {
	if strings.TrimSpace(config.AppConfig.JWTKey) != "" {
		return
	}
	if k := GetSetting(JWTSecretKeySetting, ""); k != "" {
		config.AppConfig.JWTKey = k
		return
	}
	k := utils.RandomString(64)
	SetSetting(JWTSecretKeySetting, k)
	config.AppConfig.JWTKey = k
	log.Println("[INFO] 未配置 JWT_KEY，已自动生成随机密钥并持久化到数据库（如需跨实例共享登录态请显式配置 JWT_KEY）")
}
