package handlers

import (
	"encoding/json"
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
	Name          string   `json:"name" binding:"required"`
	Platform      string   `json:"platform"`
	Mode          string   `json:"mode"`
	Types         []string `json:"types"`
	Domains       string   `json:"domains"`
	RegenerateKey bool     `json:"regenerate_key"`
	Status        *int     `json:"status"`
}

// appView 应用输出（types 解析为数组）
func appView(app *models.App) gin.H {
	var types []string
	_ = json.Unmarshal([]byte(app.Types), &types)
	if types == nil {
		types = []string{}
	}
	return gin.H{
		"id":         app.ID,
		"name":       app.Name,
		"platform":   app.Platform,
		"appid":      app.AppID,
		"app_key":    app.AppKey,
		"mode":       app.Mode,
		"types":      types,
		"domains":    app.Domains,
		"status":     app.Status,
		"created_at": app.CreatedAt,
		"updated_at": app.UpdatedAt,
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
	if req.Mode != services.ModeRainbow && req.Mode != services.ModeREST && req.Mode != services.ModeCompat {
		return "模式不合法（rainbow / rest / compat）"
	}
	for _, t := range req.Types {
		if _, ok := providers.FindMeta(t); !ok {
			return "存在不支持的登录类型: " + t
		}
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

// ListApps 应用列表
func ListApps(c *gin.Context) {
	var apps []models.App
	if err := database.DB.Order("id desc").Find(&apps).Error; err != nil {
		utils.FailInternal(c, "查询失败")
		return
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

	app := models.App{
		Name:     strings.TrimSpace(req.Name),
		Platform: req.Platform,
		AppID:    strings.ToLower(utils.RandomString(16)),
		AppKey:   utils.RandomString(32),
		Mode:     req.Mode,
		Types:    typesToJSON(req.Types),
		Domains:  req.Domains,
		Status:   1,
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

	updates := map[string]interface{}{
		"name":     strings.TrimSpace(req.Name),
		"platform": req.Platform,
		"mode":     req.Mode,
		"types":    typesToJSON(req.Types),
		"domains":  req.Domains,
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

	if err := database.DB.Delete(&models.App{}, id).Error; err != nil {
		utils.FailInternal(c, "删除应用失败")
		return
	}
	utils.SuccessMsg(c, "删除成功")
}
