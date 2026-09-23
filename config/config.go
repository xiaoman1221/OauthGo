package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config 全局配置
type Config struct {
	Port           string
	GinMode        string
	DBPath         string
	JWTKey         string
	Host           string
	TrustedProxies string // 可信代理 CIDR（逗号分隔），空则完全禁用 X-Forwarded-For 信任
}

// AppConfig 应用配置实例
var AppConfig Config

// Load 加载环境变量配置。
// JWT_KEY 留空时不在此时兜底：待数据库初始化后由 services.EnsureJWTKey
// 生成随机密钥并持久化（杜绝使用公开默认密钥被伪造 admin 令牌的风险）。
func Load() {
	_ = godotenv.Load()

	AppConfig = Config{
		Host:           os.Getenv("HOST"),
		TrustedProxies: os.Getenv("TRUSTED_PROXIES"),
		Port:           getEnv("PORT", "8080"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		DBPath:         getEnv("DB_PATH", "data.db"),
		JWTKey:         strings.TrimSpace(os.Getenv("JWT_KEY")),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
