package handlers

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/providers"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// AppRequest 应用创建/更新请求
type AppRequest struct {
	Name          string            `json:"name" binding:"required"`
	Platform      string            `json:"platform"`
	Mode          string            `json:"mode"`
	Types         []string          `json:"types"`
	Domains       string            `json:"domains"`
	RedirectURIs  []string          `json:"redirect_uris"`  // OAuth2/OIDC 回调地址白名单（按 scheme+host+path 匹配，完整 URL）
	SampleParams  map[string]string `json:"sample_params"`  // 接入示例参数：接入方自带参数的样例值，仅用于生成接入文档
	EnableRefresh bool              `json:"enable_refresh"` // OAuth2 是否签发 refresh_token
	RegenerateKey bool              `json:"regenerate_key"`
	Status        *int              `json:"status"`
}

// appView 应用输出（types 解析为数组）
func appView(app *models.App) gin.H {
	var types []string
	_ = json.Unmarshal([]byte(app.Types), &types)
	if types == nil {
		types = []string{}
	}
	var redirectURIs []string
	_ = json.Unmarshal([]byte(app.RedirectURIs), &redirectURIs)
	if redirectURIs == nil {
		redirectURIs = []string{}
	}
	var sampleParams map[string]string
	_ = json.Unmarshal([]byte(app.SampleParams), &sampleParams)
	if sampleParams == nil {
		sampleParams = map[string]string{}
	}
	return gin.H{
		"id":             app.ID,
		"owner_id":       app.OwnerID,
		"name":           app.Name,
		"platform":       app.Platform,
		"appid":          app.AppID,
		"app_key":        app.AppKey,
		"mode":           app.Mode,
		"types":          types,
		"domains":        app.Domains,
		"redirect_uris":  redirectURIs,
		"sample_params":  sampleParams,
		"enable_refresh": app.EnableRefresh,
		// OIDC Discovery URL（本平台单 issuer=HOST），供 oidc-client 等 SDK 自动发现
		"oidc_discovery_url": services.OAuthIssuer() + "/.well-known/openid-configuration",
		"status":             app.Status,
		"created_at":         app.CreatedAt,
		"updated_at":         app.UpdatedAt,
	}
}

// validateAppReq 校验并规整应用请求
func validateAppReq(req *AppRequest) string {
	if req.Name == "" {
		return "应用名称不能为空"
	}
	if req.Mode == "" {
		req.Mode = services.ModeCompat
	}
	if req.Mode != services.ModeRainbow && req.Mode != services.ModeREST &&
		req.Mode != services.ModeOAuth2 && req.Mode != services.ModeCompat {
		return "模式不合法（rainbow / rest / oauth2 / compat）"
	}
	for _, u := range req.RedirectURIs {
		if !isHTTPURL(u) {
			return "OAuth2 回调地址必须是合法的 http(s) URL: " + u
		}
	}
	for _, t := range req.Types {
		if _, ok := providers.FindMeta(t); !ok {
			return "存在不支持的登录类型: " + t
		}
	}
	if msg := validateSampleParams(req.SampleParams); msg != "" {
		return msg
	}
	if req.Domains != "" {
		normalized, msg := services.NormalizeDomains(req.Domains)
		if msg != "" {
			return msg
		}
		req.Domains = normalized
	}
	return ""
}

func typesToJSON(types []string) string {
	if types == nil {
		types = []string{}
	}
	b, _ := json.Marshal(types)
	return string(b)
}

func redirectURIsToJSON(list []string) string {
	if list == nil {
		list = []string{}
	}
	b, _ := json.Marshal(list)
	return string(b)
}

// sampleParamKeyPat 接入示例参数名：合法 query 参数名（字母、数字、下划线、点、连字符）
var sampleParamKeyPat = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)

// reservedSampleParamKeys 不允许出现在接入示例参数里的平台参数：会与生成的示例地址冲突
var reservedSampleParamKeys = map[string]bool{
	"client_id": true, "redirect_uri": true, "response_type": true,
}

// validateSampleParams 校验接入示例参数（仅用于文档示例，限制数量与长度避免被当成配置滥用）
func validateSampleParams(params map[string]string) string {
	if len(params) > 20 {
		return "接入示例参数过多（最多 20 个）"
	}
	for k, v := range params {
		if !sampleParamKeyPat.MatchString(k) {
			return "接入示例参数名不合法（仅限字母、数字、下划线、点、连字符）：" + k
		}
		if reservedSampleParamKeys[k] {
			return "接入示例参数不能覆盖平台参数：" + k
		}
		if len(v) > 512 {
			return "接入示例参数值过长（最多 512 字节）：" + k
		}
	}
	return ""
}

func sampleParamsToJSON(params map[string]string) string {
	if params == nil {
		params = map[string]string{}
	}
	b, _ := json.Marshal(params)
	return string(b)
}

// ListApps 应用列表
func ListApps(c *gin.Context) {
	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	var apps []models.App
	if role == "admin" {
		if err := database.DB.Order("id desc").Find(&apps).Error; err != nil {
			utils.FailInternal(c, "查询失败")
			return
		}
	} else {
		uid, _ := userAny.(uint)
		if err := database.DB.Where("owner_id = ?", uid).Order("id desc").Find(&apps).Error; err != nil {
			utils.FailInternal(c, "查询失败")
			return
		}
	}
	result := make([]gin.H, 0, len(apps))
	for i := range apps {
		result = append(result, appView(&apps[i]))
	}
	utils.Success(c, gin.H{"list": result})
}

// GetApp 应用详情
func GetApp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var app models.App
	if err := database.DB.First(&app, id).Error; err != nil {
		utils.FailNotFound(c, "应用不存在")
		return
	}
	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	if role != "admin" {
		uid, _ := userAny.(uint)
		if app.OwnerID != uid {
			utils.FailForbidden(c)
			return
		}
	}
	utils.Success(c, appView(&app))
}

// CreateApp 创建应用（自动生成 AppID / AppKey）
func CreateApp(c *gin.Context) {
	var req AppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误："+err.Error())
		return
	}
	if msg := validateAppReq(&req); msg != "" {
		utils.FailBadRequest(c, msg)
		return
	}

	// 权限与配额校验
	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	uid, _ := userAny.(uint)
	if role != "admin" {
		max := services.GetIntSetting("user_max_apps", 5)
		var cnt int64
		database.DB.Model(&models.App{}).Where("owner_id = ?", uid).Count(&cnt)
		if int(cnt) >= max {
			utils.Fail(c, 403, "超过普通用户可创建的应用上限")
			return
		}
		// 普通用户只能选择已配置且启用并设置为可发起主站登录的渠道（主站登录）
		for _, t := range req.Types {
			var p models.Provider
			if err := database.DB.Where("name = ? AND enabled = ? AND main_site = ?", t, true, true).First(&p).Error; err != nil {
				utils.FailBadRequest(c, "存在未配置或未启用的登录类型: "+t)
				return
			}
		}
	}

	app := models.App{
		OwnerID:       uid,
		Name:          strings.TrimSpace(req.Name),
		Platform:      req.Platform,
		AppID:         strings.ToLower(utils.RandomString(16)),
		AppKey:        utils.RandomString(32),
		Mode:          req.Mode,
		Types:         typesToJSON(req.Types),
		Domains:       req.Domains,
		RedirectURIs:  redirectURIsToJSON(req.RedirectURIs),
		SampleParams:  sampleParamsToJSON(req.SampleParams),
		EnableRefresh: req.EnableRefresh,
		Status:        1,
	}
	if req.Status != nil {
		app.Status = *req.Status
	}

	if err := database.DB.Create(&app).Error; err != nil {
		utils.FailInternal(c, "创建应用失败")
		return
	}
	utils.Success(c, appView(&app))
}

// UpdateApp 更新应用
func UpdateApp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var req AppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}
	if msg := validateAppReq(&req); msg != "" {
		utils.FailBadRequest(c, msg)
		return
	}

	var app models.App
	if err := database.DB.First(&app, id).Error; err != nil {
		utils.FailNotFound(c, "应用不存在")
		return
	}
	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	if role != "admin" {
		uid, _ := userAny.(uint)
		if app.OwnerID != uid {
			utils.FailForbidden(c)
			return
		}
		// 普通用户更新时也需校验所选登录类型已配置且可用于主站登录
		for _, t := range req.Types {
			var p models.Provider
			if err := database.DB.Where("name = ? AND enabled = ? AND main_site = ?", t, true, true).First(&p).Error; err != nil {
				utils.FailBadRequest(c, "存在未配置或未启用的登录类型: "+t)
				return
			}
		}
	}

	updates := map[string]interface{}{
		"name":           strings.TrimSpace(req.Name),
		"platform":       req.Platform,
		"mode":           req.Mode,
		"types":          typesToJSON(req.Types),
		"domains":        req.Domains,
		"redirect_uris":  redirectURIsToJSON(req.RedirectURIs),
		"sample_params":  sampleParamsToJSON(req.SampleParams),
		"enable_refresh": req.EnableRefresh,
	}
	if req.RegenerateKey {
		updates["app_key"] = utils.RandomString(32)
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := database.DB.Model(&app).Updates(updates).Error; err != nil {
		utils.FailInternal(c, "更新应用失败")
		return
	}
	database.DB.First(&app, id)
	utils.Success(c, appView(&app))
}

// DeleteApp 删除应用
func DeleteApp(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var app models.App
	if err := database.DB.First(&app, id).Error; err != nil {
		utils.FailNotFound(c, "应用不存在")
		return
	}
	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	if role != "admin" {
		uid, _ := userAny.(uint)
		if app.OwnerID != uid {
			utils.FailForbidden(c)
			return
		}
	}

	if err := database.DB.Delete(&models.App{}, id).Error; err != nil {
		utils.FailInternal(c, "删除应用失败")
		return
	}
	// 级联清理该应用在 OAuth2/OIDC 流程中的授权码与令牌，防止孤儿数据与残留访问能力
	database.DB.Where("client_id = ?", app.AppID).Delete(&models.OAuthCode{})
	database.DB.Where("client_id = ?", app.AppID).Delete(&models.OAuthAccessToken{})
	database.DB.Where("client_id = ?", app.AppID).Delete(&models.OAuthRefreshToken{})
	utils.SuccessMsg(c, "删除成功")
}
