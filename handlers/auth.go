package handlers

import (
	"strconv"
	"strings"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Code     string `json:"code"`
}

// Register 用户注册
func Register(c *gin.Context) {
	if !services.GetBoolSetting("register_enabled", true) {
		utils.FailBadRequest(c, "暂未开放注册")
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误："+err.Error())
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.Username == "" {
		utils.FailBadRequest(c, "用户名不能为空")
		return
	}
	if req.Email == "" && req.Phone == "" {
		utils.FailBadRequest(c, "邮箱或手机号至少填写一个")
		return
	}

	minLen := services.GetIntSetting("password_min_length", 6)
	if len(req.Password) < minLen {
		utils.FailBadRequest(c, "密码长度不能少于 "+strconv.Itoa(minLen)+" 位")
		return
	}

	// 注册验证
	if services.GetBoolSetting("register_email_verify", false) {
		if req.Email == "" {
			utils.FailBadRequest(c, "注册邮箱验证已开启，请填写邮箱")
			return
		}
		if err := services.VerifyVerifyCode("register", req.Email, req.Code); err != nil {
			utils.FailBadRequest(c, err.Error())
			return
		}
	}

	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		utils.FailBadRequest(c, "用户名已存在")
		return
	}
	if req.Email != "" {
		database.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "邮箱已被注册")
			return
		}
	}
	if req.Phone != "" {
		database.DB.Model(&models.User{}).Where("phone = ?", req.Phone).Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "手机号已被注册")
			return
		}
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.FailInternal(c, "密码加密失败")
		return
	}

	role := services.GetSetting("default_role", "user")
	if role == "" {
		role = "user"
	}
	user := models.User{
		Username:    req.Username,
		Nickname:    req.Username,
		Password:    hash,
		PasswordSet: true,
		Email:       req.Email,
		Phone:       req.Phone,
		Role:        role,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		utils.FailInternal(c, "创建用户失败")
		return
	}

	// 发送欢迎邮件（失败不影响注册）
	if req.Email != "" && services.SMTPEnabled() {
		_ = services.SendTemplateMail(req.Email, "welcome", "欢迎加入 "+services.GetSetting("site_name", "OauthGo"),
			map[string]interface{}{
				"username":  req.Username,
				"site_name": services.GetSetting("site_name", "OauthGo"),
				"site_url":  services.GetSetting("site_url", ""),
			})
	}

	utils.SuccessMsg(c, "注册成功")
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login 用户登录（支持用户名/邮箱/手机号）
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var user models.User
	err := database.DB.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error
	if err != nil {
		err = database.DB.Where("email = ?", strings.TrimSpace(req.Username)).First(&user).Error
	}
	if err != nil {
		err = database.DB.Where("phone = ?", strings.TrimSpace(req.Username)).First(&user).Error
	}
	if err != nil {
		utils.FailBadRequest(c, "用户名或密码错误")
		return
	}
	if !utils.CheckPassword(user.Password, req.Password) {
		utils.FailBadRequest(c, "用户名或密码错误")
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Role)
	if err != nil {
		utils.FailInternal(c, "生成令牌失败")
		return
	}
	utils.Success(c, gin.H{"token": token, "user": user})
}

// AuthConfig 公开的认证配置（供登录/注册页使用）
func AuthConfig(c *gin.Context) {
	utils.Success(c, gin.H{
		"register_enabled":      services.GetBoolSetting("register_enabled", true),
		"register_email_verify": services.GetBoolSetting("register_email_verify", false),
		"password_min_length":   services.GetIntSetting("password_min_length", 6),
		"code_length":           services.GetIntSetting("code_length", 6),
		"site_name":             services.GetSetting("site_name", "OauthGo"),
	})
}

// SendCodeRequest 发送验证码请求
type SendCodeRequest struct {
	Scope   string `json:"scope" binding:"required"` // register / reset
	Account string `json:"account" binding:"required"`
}

// SendCode 发送验证码（邮箱走 SMTP，手机号走短信）
func SendCode(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	account := strings.TrimSpace(req.Account)
	if req.Scope == "" || (req.Scope != "register" && req.Scope != "reset" && req.Scope != "bind") {
		utils.FailBadRequest(c, "验证码类型错误")
		return
	}

	// 找回密码场景：账号必须存在
	if req.Scope == "reset" {
		var count int64
		dbQuery := database.DB.Model(&models.User{})
		if strings.Contains(account, "@") {
			dbQuery = dbQuery.Where("email = ?", account)
		} else {
			dbQuery = dbQuery.Where("phone = ?", account)
		}
		dbQuery.Count(&count)
		if count == 0 {
			utils.FailBadRequest(c, "账号不存在")
			return
		}
	}

	// 注册场景：邮箱/手机号不能已存在
	if req.Scope == "register" {
		var count int64
		dbQuery := database.DB.Model(&models.User{})
		if strings.Contains(account, "@") {
			dbQuery = dbQuery.Where("email = ?", account)
		} else {
			dbQuery = dbQuery.Where("phone = ?", account)
		}
		dbQuery.Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "该账号已被注册")
			return
		}
	}

	// 绑定场景：新邮箱/手机号不能已被任何账号占用
	if req.Scope == "bind" {
		var count int64
		dbQuery := database.DB.Model(&models.User{})
		if strings.Contains(account, "@") {
			dbQuery = dbQuery.Where("email = ?", account)
		} else {
			dbQuery = dbQuery.Where("phone = ?", account)
		}
		dbQuery.Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "该邮箱或手机号已被使用")
			return
		}
	}

	if err := services.GenerateVerifyCode(req.Scope, account); err != nil {
		utils.FailBadRequest(c, err.Error())
		return
	}
	utils.SuccessMsg(c, "验证码已发送")
}

// ForgotPasswordRequest 找回密码请求
type ForgotPasswordRequest struct {
	Account  string `json:"account" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ForgotPassword 找回密码
func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	account := strings.TrimSpace(req.Account)

	var user models.User
	var err error
	if strings.Contains(account, "@") {
		err = database.DB.Where("email = ?", account).First(&user).Error
	} else {
		err = database.DB.Where("phone = ?", account).First(&user).Error
	}
	if err != nil {
		utils.FailBadRequest(c, "账号不存在")
		return
	}

	if err := services.VerifyVerifyCode("reset", account, req.Code); err != nil {
		utils.FailBadRequest(c, err.Error())
		return
	}

	if err := services.ChangePassword(user.ID, req.Password); err != nil {
		utils.FailBadRequest(c, err.Error())
		return
	}
	utils.SuccessMsg(c, "密码重置成功")
}

// Me 获取当前登录用户信息
func Me(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.FailNotFound(c, "用户不存在")
		return
	}
	utils.Success(c, user)
}
