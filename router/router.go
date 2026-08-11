package router

import (
	"net/http"
	"os"
	"path/filepath"

	"OauthGo/handlers"
	"OauthGo/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 初始化路由
func Setup() *gin.Engine {
	r := gin.Default()

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

		// 认证模块
		auth := api.Group("/auth")
		{
			auth.GET("/config", handlers.AuthConfig)
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.POST("/send-code", handlers.SendCode)
			auth.POST("/forgot", handlers.ForgotPassword)
			auth.GET("/me", middleware.JWT(), handlers.Me)

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

// serveFrontend 提供前端静态资源（构建产物位于 web/dist）
func serveFrontend(r *gin.Engine) {
	dist := frontendDistDir()
	if info, err := os.Stat(dist); err != nil || !info.IsDir() {
		return
	}

	r.Static("/assets", filepath.Join(dist, "assets"))
	r.StaticFile("/favicon.ico", filepath.Join(dist, "favicon.ico"))
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
