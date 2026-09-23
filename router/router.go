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
			// GET 也支持换令牌：QQ / 微信 / WPS 企业 SSO 等接入方用 GET + query 参数交换。
			// 只注册 POST 时，这类请求会落到前端路由回退、返回 HTML，接入方解析 JSON 失败
			// （典型报错 "token response json decode failed"）。参数在 c.Request.Form 中，
			// GET 时即 query，handler 无需区分。
			// 注意：GET 会把 client_secret 暴露在 URL（可能进代理日志），能用 POST 时应优先 POST。
			oauth2.GET("/token", handlers.TokenOAuth2)
			oauth2.GET("/userinfo", handlers.OAuth2Userinfo)
			oauth2.POST("/userinfo", handlers.OAuth2Userinfo)
			oauth2.GET("/jwks", handlers.OAuth2JWKS)
			oauth2.POST("/revoke", handlers.OAuth2Revoke)
			oauth2.POST("/introspect", handlers.OAuth2Introspect)
			oauth2.POST("/platform-login", handlers.OAuth2PlatformLogin)
			// 已登录用户一键确认授权（会话 Cookie 即凭据）
			oauth2.POST("/consent", handlers.OAuth2Consent)
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
			// 一次性登录码兑换 JWT（/oauth-callback?code= 回跳后调用，免认证）
			auth.POST("/code-exchange", handlers.LoginCodeExchange)
			// OIDC 登录会话：退出与在线会话管理（logout 以会话 Cookie 为凭据，无需 JWT）
			auth.POST("/logout", handlers.Logout)
			auth.GET("/sessions", middleware.JWT(), handlers.ListSessions)
			auth.DELETE("/sessions/:id", middleware.JWT(), handlers.RevokeSession)
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

			// 系统设置模块（仅管理员；普通用户所需的头像等公开配置走 /api/auth/config）
			adminSettings := authed.Group("/settings")
			adminSettings.Use(middleware.AdminOnly())
			{
				adminSettings.GET("", handlers.ListSettings)
				adminSettings.PUT("", handlers.UpdateSettings)
				adminSettings.POST("/test/smtp", handlers.TestSMTP)
				adminSettings.POST("/test/sms", handlers.TestSMS)
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
		oauth2Root.GET("/token", handlers.TokenOAuth2) // 见上方 GET 换令牌说明
		oauth2Root.GET("/userinfo", handlers.OAuth2Userinfo)
		oauth2Root.POST("/userinfo", handlers.OAuth2Userinfo)
		oauth2Root.GET("/jwks", handlers.OAuth2JWKS)
		oauth2Root.POST("/revoke", handlers.OAuth2Revoke)
		oauth2Root.POST("/introspect", handlers.OAuth2Introspect)
		oauth2Root.POST("/platform-login", handlers.OAuth2PlatformLogin)
		oauth2Root.POST("/consent", handlers.OAuth2Consent)
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

// securityHeaders 基础安全响应头：防嗅探 / 防点击劫持 iframe / 收紧 referrer。
// X-Frame-Options 用 SAMEORIGIN：控制台页面不允许被第三方站点嵌入，
// 但需允许本站前端「服务文档」页以同源 iframe 内嵌 /docs。
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "SAMEORIGIN")
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
	hasDist := false
	if info, err := os.Stat(dist); err == nil && info.IsDir() {
		hasDist = true
		r.Static("/assets", filepath.Join(dist, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(dist, "favicon.ico"))
		r.StaticFile("/favicon.svg", filepath.Join(dist, "favicon.svg"))
	}

	// 未构建前端（无 web/dist）时同样注册 NoRoute：后端命名空间必须回 JSON 404，
	// 不能落到 gin 默认的纯文本 404
	r.NoRoute(func(c *gin.Context) {
		// 后端命名空间（/api、/oauth2、/.well-known、/connect.php 及 OAuth2 端点路径）
		// 一律返回 404 JSON：绝不回退前端页面，否则接入方（如 WPS 企业 SSO）会拿到 HTML
		// 而报 "token response json decode failed" 这类难以定位的错误。
		if !hasDist || isReservedBackendPath(c.Request.URL.Path) {
			c.JSON(404, gin.H{"code": 404, "message": "not found"})
			return
		}
		// GET/HEAD 均回退前端入口（http.ServeFile 对 HEAD 自动省略响应体）
		if m := c.Request.Method; m == http.MethodGet || m == http.MethodHead {
			c.File(filepath.Join(dist, "index.html"))
			return
		}
		c.JSON(404, gin.H{"code": 404, "message": "not found"})
	})
}

// reservedBackendPrefixes 后端接口路径前缀（含子路径）
var reservedBackendPrefixes = []string{"/api", "/oauth2", "/.well-known", "/connect.php"}

// reservedBackendPaths OAuth2 端点的精确路径（方法未注册时同样不能回退前端页面）
var reservedBackendPaths = map[string]struct{}{
	"/authorize": {}, "/token": {}, "/userinfo": {}, "/jwks": {},
	"/revoke": {}, "/introspect": {}, "/platform-login": {}, "/consent": {},
}

// isReservedBackendPath 判断路径是否属于后端接口/OAuth2 端点
func isReservedBackendPath(p string) bool {
	if _, ok := reservedBackendPaths[p]; ok {
		return true
	}
	for _, prefix := range reservedBackendPrefixes {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	return false
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
