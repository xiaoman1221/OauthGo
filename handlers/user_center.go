package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// UpdateProfileRequest 修改个人资料请求
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	EmailCode string `json:"email_code"`
	Phone     string `json:"phone"`
	PhoneCode string `json:"phone_code"`
}

// UpdateProfile 修改个人资料（昵称/头像/用户名/邮箱/手机号）
func UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.FailNotFound(c, "用户不存在")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误："+err.Error())
		return
	}

	req.Nickname = strings.TrimSpace(req.Nickname)
	req.Avatar = strings.TrimSpace(req.Avatar)
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)

	updates := map[string]interface{}{}

	if req.Nickname != "" && req.Nickname != user.Nickname {
		if len([]rune(req.Nickname)) > 64 {
			utils.FailBadRequest(c, "昵称长度不能超过 64 个字符")
			return
		}
		updates["nickname"] = req.Nickname
	}

	if req.Avatar != "" && req.Avatar != user.Avatar {
		if !isHTTPURL(req.Avatar) || len(req.Avatar) > 512 {
			utils.FailBadRequest(c, "头像地址必须是合法的 http(s) URL")
			return
		}
		updates["avatar"] = req.Avatar
	}

	if req.Username != "" && req.Username != user.Username {
		if len([]rune(req.Username)) > 64 {
			utils.FailBadRequest(c, "用户名长度不能超过 64 个字符")
			return
		}
		var count int64
		database.DB.Model(&models.User{}).Where("username = ? AND id != ?", req.Username, user.ID).Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "用户名已被使用")
			return
		}
		updates["username"] = req.Username
	}

	if req.Email != "" && req.Email != user.Email {
		if !strings.Contains(req.Email, "@") {
			utils.FailBadRequest(c, "邮箱格式不正确")
			return
		}
		var count int64
		database.DB.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, user.ID).Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "邮箱已被其他账号使用")
			return
		}
		if err := services.VerifyVerifyCode("bind", req.Email, req.EmailCode); err != nil {
			utils.FailBadRequest(c, err.Error())
			return
		}
		updates["email"] = req.Email
	}

	if req.Phone != "" && req.Phone != user.Phone {
		var count int64
		database.DB.Model(&models.User{}).Where("phone = ? AND id != ?", req.Phone, user.ID).Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "手机号已被其他账号使用")
			return
		}
		if err := services.VerifyVerifyCode("bind", req.Phone, req.PhoneCode); err != nil {
			utils.FailBadRequest(c, err.Error())
			return
		}
		updates["phone"] = req.Phone
	}

	if len(updates) == 0 {
		utils.FailBadRequest(c, "没有需要修改的信息")
		return
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		utils.FailInternal(c, "保存失败")
		return
	}
	utils.Success(c, user)
}

// ChangePasswordRequest 修改密码请求（旧密码仅已设置密码时校验）
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ChangePassword 修改当前用户密码
func ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误："+err.Error())
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.FailNotFound(c, "用户不存在")
		return
	}

	// 已设置密码的用户须校验原密码；未设置密码（第三方注册）可直接设置
	if user.PasswordSet && !utils.CheckPassword(user.Password, req.OldPassword) {
		utils.FailBadRequest(c, "原密码错误")
		return
	}
	if err := services.ChangePassword(user.ID, req.NewPassword); err != nil {
		utils.FailBadRequest(c, err.Error())
		return
	}
	database.DB.Model(&user).Update("password_set", true)
	utils.SuccessMsg(c, "密码修改成功")
}

// MyBindings 当前用户已启用渠道及其绑定状态
func MyBindings(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var list []models.Provider
	database.DB.Where("enabled = ?", true).Order("sort asc").Find(&list)

	var accounts []models.ProviderAccount
	database.DB.Where("user_id = ?", userID).Find(&accounts)
	bound := make(map[string]models.ProviderAccount)
	for _, a := range accounts {
		bound[a.Provider] = a
	}

	result := make([]gin.H, 0, len(list))
	for _, p := range list {
		item := gin.H{
			"name":         p.Name,
			"display_name": p.DisplayName,
			"category":     p.Category,
			"bound":        false,
		}
		if a, ok := bound[p.Name]; ok {
			item["bound"] = true
			item["nickname"] = a.Nickname
			item["avatar"] = a.Avatar
		}
		result = append(result, item)
	}
	utils.Success(c, result)
}

// BindLogin 绑定第三方登录。
// GET: 生成绑定会话并返回渠道授权地址；POST: 无跳转渠道（如微信小程序）直接以 code 绑定
func BindLogin(c *gin.Context) {
	name := c.Param("provider")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)

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
		if err := services.BindProviderAccount(uid, name, info); err != nil {
			utils.FailBadRequest(c, err.Error())
			return
		}
		utils.SuccessMsg(c, "绑定成功")
		return
	}

	state := services.CreateBindSession(uid, name)
	authURL := prov.GetAuthURL(state)
	if authURL == "" {
		utils.FailBadRequest(c, "该渠道不支持网页跳转绑定，请使用绑定接口")
		return
	}
	utils.Success(c, gin.H{"url": authURL})
}

// UnbindLogin 解绑第三方登录
func UnbindLogin(c *gin.Context) {
	name := c.Param("provider")
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)

	if err := services.UnbindProviderAccount(uid, name); err != nil {
		utils.FailBadRequest(c, err.Error())
		return
	}
	utils.SuccessMsg(c, "解绑成功")
}

// isHTTPURL 判断是否为合法的 http/https 地址
func isHTTPURL(s string) bool {
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}
