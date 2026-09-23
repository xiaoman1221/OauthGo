package main

import (
	"log"

	"OauthGo/config"
	"OauthGo/database"
	"OauthGo/router"
	"OauthGo/services"
)

func main() {
	config.Load()
	database.Init()
	services.InitSettings()
	// 解析 JWT 签名密钥（环境变量优先，否则生成并持久化）
	services.EnsureJWTKey()

	r := router.Setup()
	log.Println("OauthGo Powered By xiaoman1221")
	log.Println("OauthGo [https://github.com/xiaoman1221/OauthGo]")
	log.Printf("OauthGo 服务已启动: %s", config.AppConfig.Host)
	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
