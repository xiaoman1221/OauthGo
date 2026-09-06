package router

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"OauthGo/config"
	"OauthGo/handlers"
	"OauthGo/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 初始化路由
func Setup() *gin.Engine {
	gin.SetMode(config.AppConfig.GinMode)
	// 自建 engine：日志中间件不打印 URL 查询串，避免 appid/appkey/sign/密码等敏感参数进入日志
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(securityHeaders())
	r.Use(requestLogger())

	// 默认不信任任何代理（禁止伪造 X-Forwarded-For 污染 c.ClientIP 审计）。
	// 部署在反向代理后时，通过环境变量 TRUSTED_PROXIES 配置可信代理 CIDR（逗号分隔）。
	if raw := strings.TrimSpace(config.AppConfig.TrustedProxies); raw != "" {
		proxies := make([]string, 0)
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				proxies = append(proxies, p)
			}
		}
		if err := r.SetTrustedProxies(proxies); err != nil {
			log.Printf("[WARN] TRUSTED_PROXIES 配置无效: %v", err)
		}
	} else {
		_ = r.SetTrustedProxies(nil)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		// 彩虹聚合登录协议兼容接口
		// 根路径 /connect.php 与 /api/connect.php 均可访问，支持 GET/POST
		r.GET("/connect.php", handlers.RainbowConnect)
		r.POST("/connect.php", handlers.RainbowConnect)
		api.GET("/connect.php", handlers.RainbowConnect)
		api.POST("/connect.php", handlers.RainbowConnect)

		// 标准 OAuth2 / OIDC 授权服务器接口（authorization code flow）
		oauth2 := api.Group("/oauth2")
		{
			oauth2.GET("/authorize", handlers.AuthorizeOAuth2)
			oauth2.POST("/authorize", handlers.AuthorizeOAuth2)
			oauth2.POST("/token", handlers.TokenOAuth2)
			oauth2.GET("/userinfo", handlers.OAuth2Userinfo)
			oauth2.POST("/userinfo", handlers.OAuth2Userinfo)
			oauth2.GET("/jwks", handlers.OAuth2JWKS)
			oauth2.POST("/revoke", handlers.OAuth2Revoke)
			oauth2.POST("/introspect", handlers.OAuth2Introspect)
			oauth2.POST("/platform-login", handlers.OAuth2PlatformLogin)
			oauth2.GET("/.well-known/openid-configuration", handlers.OAuth2Discovery)
			oauth2.GET("/.well-known/oauth-authorization-server", handlers.OAuth2Discovery)
		}

		// REST 风格聚合登录接口
		v1 := api.Group("/v1/oauth")
		{
			v1.GET("/login", handlers.RESTLogin)
			v1.POST("/login", handlers.RESTLogin)
			v1.POST("/userinfo", handlers.RESTUserInfo)
			v1.POST("/query", handlers.RESTQuery)
		}

		// 第三方登录（免登录）
		oauth := api.Group("/oauth")
		{
			oauth.GET("/providers", handlers.PublicProviders)
			oauth.GET("/:provider/login", handlers.OAuthLogin)
			oauth.POST("/:provider/login", handlers.OAuthLogin)
			oauth.GET("/:provider/callback", handlers.OAuthCallback)
			oauth.POST("/:provider/callback", handlers.OAuthCallback)
		}

		// 站点公开接口（登录页背景等）
		site := api.Group("/site")
		{
			site.GET("/bing-daily", handlers.BingDaily)
		}

		// 认证模块
		auth := api.Group("/auth")
		{
			auth.GET("/config", handlers.AuthConfig)
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.POST("/send-code", handlers.SendCode)
			auth.POST("/forgot", handlers.ForgotPassword)
			auth.GET("/me", middleware.JWT(), handlers.Me)

			// Passkey / WebAuthn
			passkey := auth.Group("/passkey")
			{
				passkey.POST("/register/begin", middleware.JWT(), handlers.PasskeyRegisterBegin)
				passkey.POST("/register/finish", middleware.JWT(), handlers.PasskeyRegisterFinish)
				passkey.GET("/login/begin", handlers.PasskeyLoginBegin)
				passkey.POST("/login/begin", handlers.PasskeyLoginBegin)
				passkey.POST("/login/finish", handlers.PasskeyLoginFinish)
				passkey.GET("", middleware.JWT(), handlers.PasskeyList)
				passkey.DELETE("/:id", middleware.JWT(), handlers.PasskeyDelete)
			}

			// 用户中心
			auth.GET("/bindings", middleware.JWT(), handlers.MyBindings)
			auth.PUT("/me", middleware.JWT(), handlers.UpdateProfile)
			auth.PUT("/password", middleware.JWT(), handlers.ChangePassword)
			auth.GET("/bind/:provider", middleware.JWT(), handlers.BindLogin)
			auth.POST("/bind/:provider", middleware.JWT(), handlers.BindLogin)
			auth.DELETE("/bind/:provider", middleware.JWT(), handlers.UnbindLogin)
		}

		// 需要登录的接口
		authed := api.Group("")
		authed.Use(middleware.JWT())
		{
			// 应用管理
			apps := authed.Group("/apps")
			{
				apps.GET("", handlers.ListApps)
				apps.POST("", handlers.CreateApp)
				apps.GET("/:id", handlers.GetApp)
				apps.PUT("/:id", handlers.UpdateApp)
				apps.DELETE("/:id", handlers.DeleteApp)
			}

			// 登录管理
			logins := authed.Group("/logins")
			{
				logins.GET("", handlers.ListLoginRecords)
				logins.DELETE("/:id", handlers.DeleteLoginRecord)
				logins.POST("/batch-delete", handlers.BatchDeleteLoginRecords)
				logins.GET("/export", handlers.ExportLoginRecords)
			}

			// 系统设置模块
			settings := authed.Group("/settings")
			{
				// 普通用户可读取设置（包含用户级限制），管理员可写入
				settings.GET("", handlers.ListSettings)
				adminSettings := settings.Group("")
				adminSettings.Use(middleware.AdminOnly())
				{
					adminSettings.PUT("", handlers.UpdateSettings)
					adminSettings.POST("/test/smtp", handlers.TestSMTP)
					adminSettings.POST("/test/sms", handlers.TestSMS)
				}
			}

			// 用户管理模块（管理员）
			users := authed.Group("/users")
			users.Use(middleware.AdminOnly())
			{
				users.GET("", handlers.ListUsers)
				users.POST("", handlers.CreateUser)
				users.PUT("/:id", handlers.UpdateUser)
				users.DELETE("/:id", handlers.DeleteUser)
			}

			// 登录渠道管理模块（管理员）
			providersGroup := authed.Group("/providers")
			providersGroup.Use(middleware.AdminOnly())
			{
				providersGroup.GET("", handlers.ListProviders)
				providersGroup.PUT("/:name", handlers.UpdateProvider)
				providersGroup.POST("/:name/test", handlers.TestProvider)
			}
		}
	}

	// 标准 OAuth2 / OIDC 授权服务器端点（与 issuer=HOST 对齐的官方路径）
	// RFC 6749 / OIDC Core / RFC 8414：Discovery 文档位于 {issuer}/.well-known/openid-configuration
	oauth2Root := r.Group("")
	{
		oauth2Root.GET("/authorize", handlers.AuthorizeOAuth2)
		oauth2Root.POST("/authorize", handlers.AuthorizeOAuth2)
		oauth2Root.POST("/token", handlers.TokenOAuth2)
		oauth2Root.GET("/userinfo", handlers.OAuth2Userinfo)
		oauth2Root.POST("/userinfo", handlers.OAuth2Userinfo)
		oauth2Root.GET("/jwks", handlers.OAuth2JWKS)
		oauth2Root.POST("/revoke", handlers.OAuth2Revoke)
		oauth2Root.POST("/introspect", handlers.OAuth2Introspect)
		oauth2Root.GET("/.well-known/openid-configuration", handlers.OAuth2Discovery)
		oauth2Root.GET("/.well-known/oauth-authorization-server", handlers.OAuth2Discovery)
	}

	// 接口文档与 OpenAPI
	docs := r.Group("/docs")
	{
		docs.GET("", handlers.DocsIndex)
		docs.GET("/", handlers.DocsIndex)
		docs.GET("/openapi.yaml", handlers.DocsOpenAPI)
		docs.GET("/swagger", handlers.DocsSwagger)
	}

	serveFrontend(r)

	return r
}

// securityHeaders 基础安全响应头：防嗅探 / 防点击劫持 iframe / 收紧 referrer
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// requestLogger 脱敏请求日志：仅记录方法/路径/状态/耗时/客户端 IP，不记录 query 与表单
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s %s %d %s", c.ClientIP(), c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(start))
	}
}

// serveFrontend 提供前端静态资源（构建产物位于 web/dist）
func serveFrontend(r *gin.Engine) {
	dist := frontendDistDir()
	if info, err := os.Stat(dist); err != nil || !info.IsDir() {
		return
	}

	r.Static("/assets", filepath.Join(dist, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(dist, "favicon.ico"))
	r.StaticFile("/favicon.svg", filepath.Join(dist, "favicon.svg"))
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			c.File(filepath.Join(dist, "index.html"))
			return
		}
		c.JSON(404, gin.H{"code": 404, "message": "not found"})
	})
}

// frontendDistDir 定位前端构建产物目录：
// 优先取可执行文件同级的 web/dist（发布包/容器），回退到当前工作目录的 web/dist（本地 go run / 仓库内运行）。
func frontendDistDir() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "web", "dist")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return "web/dist"
}
