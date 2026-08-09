package handlers

import (
	"net/http"

	"OauthGo/models"
	"OauthGo/providers"
	"OauthGo/services"

	"github.com/gin-gonic/gin"
)

// rainbowResponse 彩虹聚合登录协议统一 JSON 响应
func rainbowResponse(c *gin.Context, code int, msg, typeName string, data gin.H) {
	body := gin.H{"code": code, "msg": msg}
	if typeName != "" {
		body["type"] = typeName
	}
	for k, v := range data {
		body[k] = v
	}
	c.JSON(http.StatusOK, body)
}

// resolveApp 校验应用、appkey 与模式是否允许指定协议
func resolveApp(appID, appKey string, allowedMode ...string) (*models.App, string) {
	app, err := services.GetAppByID(appID)
	if err != nil {
		return nil, err.Error()
	}
	if appKey == "" || app.AppKey != appKey {
		return nil, "appkey 校验失败"
	}
	for _, m := range allowedMode {
		if app.Mode == m {
			return app, ""
		}
	}
	return nil, "该应用未开启此协议"
}

// resolveAppMode 仅校验应用与模式（身份由服务端签名校验）
func resolveAppMode(appID string, allowedMode ...string) (*models.App, string) {
	app, err := services.GetAppByID(appID)
	if err != nil {
		return nil, err.Error()
	}
	for _, m := range allowedMode {
		if app.Mode == m {
			return app, ""
		}
	}
	return nil, "该应用未开启此协议"
}

// buildLoginURL 构建跳转授权地址（应用登录会话）
func buildLoginURL(app *models.App, typeName, redirectURI string) (string, string) {
	providerName, ok := services.ResolveType(typeName, app.Mode)
	if !ok {
		return "", "不支持该登录类型"
	}
	if !services.AppSupportsType(app, providerName) {
		return "", "该应用未开启此登录类型"
	}
	if !services.ValidateRedirect(redirectURI, app) {
		return "", "回跳地址不在允许范围内"
	}

	prov, ok := LoadProvider(providerName)
	if !ok {
		return "", "登录渠道未配置或未启用"
	}

	state := services.CreateAppSession(app.AppID, providerName, typeName, redirectURI)
	authURL := prov.GetAuthURL(state)
	if authURL == "" {
		return "", "该登录类型不支持网页跳转"
	}
	return authURL, ""
}

// rainbowParam 读取请求参数：优先 URL 查询参数，兼容 POST 表单提交
// （彩虹官方客户端可能以 POST 方式携带查询串或表单参数）
func rainbowParam(c *gin.Context, key string) string {
	return c.Request.FormValue(key)
}

// RainbowConnect 彩虹聚合登录协议兼容接口（支持 GET/POST，参数可放查询串或表单）
// GET/POST /connect.php?act=login&appid=&appkey=&type=&redirect_uri=
// GET/POST /connect.php?act=callback&appid=&appkey=&type=&code=
// GET/POST /connect.php?act=query&appid=&appkey=&type=&social_uid=
func RainbowConnect(c *gin.Context) {
	act := rainbowParam(c, "act")
	appID := rainbowParam(c, "appid")
	appKey := rainbowParam(c, "appkey")
	typeName := rainbowParam(c, "type")

	switch act {
	case "login":
		app, errMsg := resolveApp(appID, appKey, services.ModeRainbow, services.ModeCompat)
		if app == nil {
			rainbowResponse(c, 1, errMsg, "", nil)
			return
		}
		authURL, errMsg := buildLoginURL(app, typeName, rainbowParam(c, "redirect_uri"))
		if authURL == "" {
			rainbowResponse(c, 1, errMsg, typeName, nil)
			return
		}
		rainbowResponse(c, 0, "succ", typeName, gin.H{"url": authURL, "qrcode": ""})
		return

	case "callback":
		app, errMsg := resolveApp(appID, appKey, services.ModeRainbow, services.ModeCompat)
		if app == nil {
			rainbowResponse(c, 1, errMsg, "", nil)
			return
		}
		record, err := services.ExchangeCode(appID, rainbowParam(c, "code"))
		if err != nil {
			rainbowResponse(c, 2, "未完成登录", typeName, nil)
			return
		}
		rainbowResponse(c, 0, "succ", record.Type, gin.H{
			"access_token": record.AccessToken,
			"social_uid":   record.OpenID,
			"faceimg":      record.Avatar,
			"nickname":     record.Nickname,
			"location":     record.Location,
			"gender":       record.Gender,
			"ip":           record.IP,
		})
		return

	case "query":
		app, errMsg := resolveApp(appID, appKey, services.ModeRainbow, services.ModeCompat)
		if app == nil {
			rainbowResponse(c, 1, errMsg, "", nil)
			return
		}
		record, err := services.QueryUserBySocialUID(appID, typeName, rainbowParam(c, "social_uid"))
		if err != nil {
			rainbowResponse(c, 1, err.Error(), typeName, nil)
			return
		}
		rainbowResponse(c, 0, "succ", record.Platform, gin.H{
			"social_uid": record.OpenID,
			"faceimg":    record.Avatar,
			"nickname":   record.Nickname,
			"location":   record.Location,
			"ip":         record.IP,
		})
		return

	default:
		rainbowResponse(c, 1, "未知的 act 参数", "", nil)
	}
}

// restOK REST 接口统一成功响应
func restOK(c *gin.Context, data gin.H) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": data})
}

// restFail REST 接口统一失败响应
func restFail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "message": msg})
}

// restParams 解析 REST 请求参数（支持查询参数或 JSON body，body 优先）
func restParams(c *gin.Context) map[string]string {
	params := map[string]string{}
	for k, vs := range c.Request.URL.Query() {
		if len(vs) > 0 {
			params[k] = vs[0]
		}
	}
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err == nil {
		for k, v := range body {
			if s, ok := v.(string); ok {
				params[k] = s
			}
		}
	}
	return params
}

// RESTLogin REST 风格登录接口
// POST /api/v1/oauth/login  body: {appid, appkey, type, redirect_uri}
func RESTLogin(c *gin.Context) {
	params := restParams(c)
	appID := params["appid"]
	appKey := params["appkey"]
	typeName := params["type"]
	redirectURI := params["redirect_uri"]

	app, errMsg := resolveApp(appID, appKey, services.ModeREST, services.ModeCompat)
	if app == nil {
		restFail(c, 1, errMsg)
		return
	}
	authURL, errMsg := buildLoginURL(app, typeName, redirectURI)
	if authURL == "" {
		restFail(c, 1, errMsg)
		return
	}
	restOK(c, gin.H{"url": authURL, "type": typeName})
}

// RESTUserInfo 使用 code 换取用户信息（服务端签名校验）
// POST /api/v1/oauth/userinfo  body: {appid, code, type, sign}
func RESTUserInfo(c *gin.Context) {
	params := restParams(c)
	appID := params["appid"]
	sign := params["sign"]

	app, errMsg := resolveAppMode(appID, services.ModeREST, services.ModeCompat)
	if app == nil {
		restFail(c, 1, errMsg)
		return
	}
	if !services.VerifySign(map[string]string{
		"appid": appID,
		"code":  params["code"],
		"type":  params["type"],
	}, app.AppKey, sign) {
		restFail(c, 1, "签名校验失败")
		return
	}

	record, err := services.ExchangeCode(appID, params["code"])
	if err != nil {
		restFail(c, 2, err.Error())
		return
	}
	restOK(c, gin.H{
		"type":         record.Type,
		"openid":       record.OpenID,
		"unionid":      record.UnionID,
		"nickname":     record.Nickname,
		"avatar":       record.Avatar,
		"email":        record.Email,
		"gender":       record.Gender,
		"location":     record.Location,
		"access_token": record.AccessToken,
		"ip":           record.IP,
	})
}

// RESTQuery 按第三方 UID 查询用户信息（服务端签名校验）
// POST /api/v1/oauth/query  body: {appid, type, social_uid, sign}
func RESTQuery(c *gin.Context) {
	params := restParams(c)
	appID := params["appid"]
	sign := params["sign"]

	app, errMsg := resolveAppMode(appID, services.ModeREST, services.ModeCompat)
	if app == nil {
		restFail(c, 1, errMsg)
		return
	}
	if !services.VerifySign(map[string]string{
		"appid":      appID,
		"type":       params["type"],
		"social_uid": params["social_uid"],
	}, app.AppKey, sign) {
		restFail(c, 1, "签名校验失败")
		return
	}

	record, err := services.QueryUserBySocialUID(appID, params["type"], params["social_uid"])
	if err != nil {
		restFail(c, 1, err.Error())
		return
	}
	restOK(c, gin.H{
		"type":     record.Platform,
		"openid":   record.OpenID,
		"nickname": record.Nickname,
		"avatar":   record.Avatar,
		"location": record.Location,
		"ip":       record.IP,
	})
}

// LoadProvider 从数据库加载渠道配置并构建适配器（导出，供聚合接口使用）
func LoadProvider(name string) (providers.Provider, bool) {
	return loadProvider(name)
}
