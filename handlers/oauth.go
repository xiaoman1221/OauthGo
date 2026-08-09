package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"OauthGo/config"
	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/providers"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// PublicProviders 登录页获取已启用的登录渠道
func PublicProviders(c *gin.Context) {
	var list []models.Provider
	if err := database.DB.Where("enabled = ?", true).Order("sort asc").Find(&list).Error; err != nil {
		utils.FailInternal(c, "查询失败")
		return
	}

	result := make([]gin.H, 0, len(list))
	for _, p := range list {
		result = append(result, gin.H{
			"name":         p.Name,
			"display_name": p.DisplayName,
			"category":     p.Category,
		})
	}
	utils.Success(c, result)
}

// ListProviders 渠道配置列表（管理员）
func ListProviders(c *gin.Context) {
	var list []models.Provider
	if err := database.DB.Order("sort asc").Find(&list).Error; err != nil {
		utils.FailInternal(c, "查询失败")
		return
	}
	utils.Success(c, gin.H{"list": list})
}

// ProviderRequest 更新渠道配置请求
type ProviderRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
	Config       string `json:"config"`
	Enabled      *bool  `json:"enabled"`
}

// UpdateProvider 更新渠道配置（管理员）
func UpdateProvider(c *gin.Context) {
	name := c.Param("name")

	var p models.Provider
	if err := database.DB.Where("name = ?", name).First(&p).Error; err != nil {
		utils.FailNotFound(c, "登录渠道不存在")
		return
	}

	var req ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误："+err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.ClientID != "" {
		updates["client_id"] = req.ClientID
	}
	if req.ClientSecret != "" {
		updates["client_secret"] = req.ClientSecret
	}
	if req.RedirectURL != "" {
		updates["redirect_url"] = req.RedirectURL
	}
	if req.Config != "" {
		if !json.Valid([]byte(req.Config)) {
			utils.FailBadRequest(c, "扩展配置必须是合法的 JSON")
			return
		}
		updates["config"] = req.Config
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if len(updates) == 0 {
		utils.FailBadRequest(c, "没有需要更新的字段")
		return
	}

	if err := database.DB.Model(&p).Updates(updates).Error; err != nil {
		utils.FailInternal(c, "保存失败")
		return
	}
	utils.SuccessMsg(c, "保存成功")
}

// OAuthLogin 发起第三方登录
// GET:  生成 state 并 302 跳转到渠道授权页
// POST: 无跳转渠道（如微信小程序），body 传入 {code: "js_code"} 直接登录
func OAuthLogin(c *gin.Context) {
	name := c.Param("provider")
	prov, ok := loadProvider(name)
	if !ok {
		utils.FailNotFound(c, "登录渠道不存在或未启用")
		return
	}

	if c.Request.Method == http.MethodPost {
		var req struct {
			Code string `json:"code" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.FailBadRequest(c, "缺少 code 参数")
			return
		}
		info, err := prov.GetUserInfo(req.Code)
		if err != nil {
			utils.FailInternal(c, "获取用户信息失败："+err.Error())
			return
		}
		token, err := finishLogin(c, name, info)
		if err != nil {
			utils.FailInternal(c, "登录失败："+err.Error())
			return
		}
		utils.Success(c, gin.H{"token": token})
		return
	}

	state := providers.GenerateState()
	authURL := prov.GetAuthURL(state)
	if authURL == "" {
		utils.FailBadRequest(c, "该渠道不支持网页跳转登录，请调用登录接口")
		return
	}
	c.Redirect(302, authURL)
}

// OAuthCallback 第三方渠道回调
func OAuthCallback(c *gin.Context) {
	name := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	// 优先处理目标站点登录会话（彩虹 / REST 协议）
	if session, ok := services.ResolveAppSession(state); ok {
		handleAppCallback(c, session, code)
		return
	}

	// 处理「用户中心」绑定会话
	if session, ok := services.ResolveBindSession(state); ok {
		handleBindCallback(c, name, session, code)
		return
	}

	if !providers.VerifyState(state) {
		utils.FailBadRequest(c, "state 校验失败，请重新发起登录")
		return
	}

	prov, ok := loadProvider(name)
	if !ok {
		utils.FailNotFound(c, "登录渠道不存在或未启用")
		return
	}

	info, err := prov.GetUserInfo(code)
	if err != nil {
		utils.FailInternal(c, "获取用户信息失败："+err.Error())
		return
	}

	token, err := finishLogin(c, name, info)
	if err != nil {
		utils.FailInternal(c, "登录失败："+err.Error())
		return
	}

	c.Redirect(302, baseHost()+"/oauth-callback?token="+token)
}

// handleAppCallback 目标站点登录回调：签发授权码并跳回目标站点
func handleAppCallback(c *gin.Context, session services.AppSession, providerCode string) {
	prov, ok := loadProvider(session.Provider)
	redirectTarget := func(params map[string]string) {
		q := "?"
		for k, v := range params {
			q += k + "=" + url.QueryEscape(v) + "&"
		}
		c.Redirect(302, session.RedirectURI+q[:len(q)-1])
	}

	if !ok {
		redirectTarget(map[string]string{"code": "0", "msg": "登录渠道未配置或未启用"})
		return
	}

	info, err := prov.GetUserInfo(providerCode)
	if err != nil {
		redirectTarget(map[string]string{"type": session.Type, "code": "0", "msg": "获取用户信息失败"})
		return
	}

	app, err := services.GetAppByID(session.AppID)
	if err != nil {
		redirectTarget(map[string]string{"type": session.Type, "code": "0", "msg": err.Error()})
		return
	}

	record, err := services.IssueLoginCode(session.AppID, session.Type, session.Provider, info, c.ClientIP())
	if err != nil {
		redirectTarget(map[string]string{"type": session.Type, "code": "0", "msg": "签发授权码失败"})
		return
	}

	displayName := app.Name
	_ = services.RecordLogin(app.ID, displayName, models.LoginRecord{
		OpenID:    info.OpenID,
		Nickname:  info.Nickname,
		Avatar:    info.Avatar,
		Platform:  session.Type,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    1,
	})

	params := map[string]string{
		"type": session.Type,
		"code": record.Code,
	}
	// 附带服务端签名，便于目标站点服务端二次校验
	params["sign"] = services.ComputeSign(params, app.AppKey)
	redirectTarget(params)
}

// handleBindCallback 用户中心绑定回调：获取第三方信息并绑定到当前用户后跳回前端
func handleBindCallback(c *gin.Context, providerName string, session services.BindSession, providerCode string) {
	redirect := func(params map[string]string) {
		q := "?"
		for k, v := range params {
			q += k + "=" + url.QueryEscape(v) + "&"
		}
		c.Redirect(302, baseHost()+"/user-center"+q[:len(q)-1])
	}

	prov, ok := loadProvider(providerName)
	if !ok {
		redirect(map[string]string{"bind": "fail", "msg": "登录渠道未配置或未启用"})
		return
	}

	info, err := prov.GetUserInfo(providerCode)
	if err != nil {
		redirect(map[string]string{"bind": "fail", "msg": "获取用户信息失败"})
		return
	}

	if err := services.BindProviderAccount(session.UserID, providerName, info); err != nil {
		redirect(map[string]string{"bind": "fail", "msg": err.Error()})
		return
	}
	redirect(map[string]string{"bind": "success", "provider": providerName})
}

// finishLogin 绑定第三方账号与本地用户，签发 JWT 并记录登录日志
func finishLogin(c *gin.Context, providerName string, info *providers.UserInfo) (string, error) {
	user, err := bindProviderUser(providerName, info)
	if err != nil {
		return "", err
	}

	token, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", err
	}

	// 用户中心直登不经过 App，App 名留空（展示为 NULL）
	_ = services.RecordLogin(0, "", models.LoginRecord{
		OpenID:    info.OpenID,
		Nickname:  info.Nickname,
		Avatar:    info.Avatar,
		Platform:  providerName,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    1,
	})
	return token, nil
}

// bindProviderUser 按第三方 openid 查找或创建本地用户
func bindProviderUser(providerName string, info *providers.UserInfo) (*models.User, error) {
	if info.OpenID == "" {
		return nil, errors.New("未获取到第三方用户唯一标识")
	}

	var account models.ProviderAccount
	if err := database.DB.Where("provider = ? AND open_id = ?", providerName, info.OpenID).First(&account).Error; err == nil {
		var user models.User
		if err := database.DB.First(&user, account.UserID).Error; err != nil {
			return nil, err
		}
		database.DB.Model(&account).Updates(map[string]interface{}{
			"nickname": info.Nickname,
			"avatar":   info.Avatar,
			"email":    info.Email,
		})
		return &user, nil
	}

	// 新用户：生成随机密码（无法通过密码登录），绑定渠道账号
	password, _ := randomHex(16)
	hash, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	username := fmt.Sprintf("%s_%s", providerName, shortHash(info.OpenID))
	user := models.User{
		Username:    username,
		Nickname:    info.Nickname,
		Avatar:      info.Avatar,
		Password:    hash,
		PasswordSet: false,
		Role:        "user",
		Email:       info.Email,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		return nil, err
	}

	account = models.ProviderAccount{
		UserID:   user.ID,
		Provider: providerName,
		OpenID:   info.OpenID,
		UnionID:  info.UnionID,
		Nickname: info.Nickname,
		Avatar:   info.Avatar,
		Email:    info.Email,
	}
	if err := database.DB.Create(&account).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// loadProvider 从数据库加载渠道配置并构建适配器
func loadProvider(name string) (providers.Provider, bool) {
	var p models.Provider
	if err := database.DB.Where("name = ?", name).First(&p).Error; err != nil || !p.Enabled {
		return nil, false
	}

	if p.RedirectURL == "" {
		p.RedirectURL = baseHost() + "/api/oauth/" + name + "/callback"
	}

	var extra map[string]interface{}
	_ = json.Unmarshal([]byte(p.Config), &extra)

	cfg := providers.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		RedirectURL:  p.RedirectURL,
		Extra:        extra,
	}
	prov, err := providers.New(name, cfg)
	if err != nil {
		return nil, false
	}
	return prov, true
}

// baseHost 返回前端站点地址（配置的 HOST）
func baseHost() string {
	host := strings.TrimSuffix(config.AppConfig.Host, "/")
	if host == "" {
		return "http://localhost:8080"
	}
	return host
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:6])
}
