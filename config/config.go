package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config 全局配置
type Config struct {
	Port    string
	GinMode string
	DBPath  string
	JWTKey  string
	Host    string
}

// AppConfig 应用配置实例
var AppConfig Config

// Load 加载环境变量配置
func Load() {
	_ = godotenv.Load()

	AppConfig = Config{
		Host:    os.Getenv("HOST"),
		Port:    getEnv("PORT", "8080"),
		GinMode: getEnv("GIN_MODE", "debug"),
		DBPath:  getEnv("DB_PATH", "data.db"),
		JWTKey:  getEnv("JWT_KEY", ""),
	}

	if AppConfig.JWTKey == "" {
		log.Println("[WARN] 未配置 JWT_KEY，使用默认密钥（生产环境请务必修改）")
		AppConfig.JWTKey = "oauthgo-default-jwt-secret-2026"
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
