package handlers

import (
	"strconv"
	"strings"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
)

// PasskeyRegisterBegin 开始注册 Passkey（需登录）
func PasskeyRegisterBegin(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)

	passkeyUser, err := services.LoadPasskeyUser(uid)
	if err != nil {
		utils.FailNotFound(c, "用户不存在")
		return
	}
	wa, err := services.NewPasskeyWebAuthn()
	if err != nil {
		utils.FailInternal(c, "WebAuthn 配置错误："+err.Error())
		return
	}
	creation, session, err := wa.BeginRegistration(passkeyUser)
	if err != nil {
		utils.FailInternal(c, "创建注册挑战失败："+err.Error())
		return
	}
	sessionID := services.StorePasskeySession(services.PasskeySessionRegister, uid, session)
	utils.Success(c, gin.H{"options": creation, "session_id": sessionID})
}

// PasskeyRegisterFinish 完成注册并保存凭据（需登录）
func PasskeyRegisterFinish(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)

	sessionID := strings.TrimSpace(c.Query("session_id"))
	name := strings.TrimSpace(c.Query("name"))
	if len([]rune(name)) > 64 {
		name = string([]rune(name)[:64])
	}
	if sessionID == "" {
		utils.FailBadRequest(c, "缺少 session_id")
		return
	}
	sess, ok := services.GetPasskeySession(sessionID)
	if !ok || sess.Kind != services.PasskeySessionRegister || sess.UserID != uid {
		utils.FailBadRequest(c, "注册会话无效或已过期，请重试")
		return
	}
	passkeyUser, err := services.LoadPasskeyUser(uid)
	if err != nil {
		utils.FailNotFound(c, "用户不存在")
		return
	}
	wa, err := services.NewPasskeyWebAuthn()
	if err != nil {
		utils.FailInternal(c, "WebAuthn 配置错误："+err.Error())
		return
	}
	cred, err := wa.FinishRegistration(passkeyUser, *sess.Data, c.Request)
	if err != nil {
		utils.FailBadRequest(c, "注册校验失败："+err.Error())
		return
	}
	if err := services.SavePasskeyCredential(uid, name, cred); err != nil {
		utils.FailInternal(c, "保存凭据失败")
		return
	}
	utils.SuccessMsg(c, "Passkey 注册成功")
}

// PasskeyLoginBegin 开始 Passkey 登录（需提供用户名；返回可断言的挑战）
// GET/POST /api/auth/passkey/login/begin?username=xxx
// 防账号枚举：账号不存在 / 未注册 Passkey 时返回结构与真实账号完全一致的假挑战，
// 该会话绑定的用户为 0，无法完成签名，最终在 finish 处以「用户不存在」失败。
func PasskeyLoginBegin(c *gin.Context) {
	username := strings.TrimSpace(c.DefaultQuery("username", ""))
	if username == "" {
		utils.FailBadRequest(c, "请输入用户名（用于定位 Passkey 凭据）")
		return
	}
	wa, err := services.NewPasskeyWebAuthn()
	if err != nil {
		utils.FailInternal(c, "WebAuthn 配置错误："+err.Error())
		return
	}

	var passkeyUser *services.PasskeyUser
	if u, err := findUserByAccount(username); err == nil {
		if pu, puErr := services.LoadPasskeyUser(u.ID); puErr == nil && len(pu.Credentials) > 0 {
			passkeyUser = pu
		}
	}
	if passkeyUser == nil {
		passkeyUser = &services.PasskeyUser{
			User:        models.User{Username: username},
			Credentials: []webauthn.Credential{{ID: []byte(utils.RandomString(32))}},
		}
	}

	assertion, session, err := wa.BeginLogin(passkeyUser)
	if err != nil {
		utils.FailInternal(c, "创建登录挑战失败："+err.Error())
		return
	}
	sessionID := services.StorePasskeySession(services.PasskeySessionLogin, passkeyUser.User.ID, session)
	utils.Success(c, gin.H{"options": assertion, "session_id": sessionID})
}

// PasskeyLoginFinish 完成 Passkey 登录：签发平台 JWT
func PasskeyLoginFinish(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Query("session_id"))
	if sessionID == "" {
		utils.FailBadRequest(c, "缺少 session_id")
		return
	}
	sess, ok := services.GetPasskeySession(sessionID)
	if !ok || sess.Kind != services.PasskeySessionLogin {
		utils.FailBadRequest(c, "登录会话无效或已过期，请重试")
		return
	}
	passkeyUser, err := services.LoadPasskeyUser(sess.UserID)
	if err != nil {
		utils.FailBadRequest(c, "用户不存在")
		return
	}
	wa, err := services.NewPasskeyWebAuthn()
	if err != nil {
		utils.FailInternal(c, "WebAuthn 配置错误："+err.Error())
		return
	}
	cred, err := wa.FinishLogin(passkeyUser, *sess.Data, c.Request)
	if err != nil {
		utils.FailBadRequest(c, "登录校验失败："+err.Error())
		return
	}
	// 更新凭据（计数器）与使用时间
	_ = services.SavePasskeyCredential(sess.UserID, "", cred)
	services.UpdatePasskeyLastUsed(cred.ID)

	token, err := utils.GenerateToken(sess.UserID, passkeyUser.User.Role)
	if err != nil {
		utils.FailInternal(c, "生成令牌失败")
		return
	}
	// 同步建立 OIDC 登录会话（授权页免重复登录）
	establishSession(c, sess.UserID)
	_ = services.RecordLogin(0, "", models.LoginRecord{
		OpenID:    "",
		Username:  passkeyUser.User.Username,
		Nickname:  passkeyUser.User.Nickname,
		Avatar:    passkeyUser.User.Avatar,
		Platform:  "passkey",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    1,
	})
	resp := gin.H{"token": token, "user": passkeyUser.User}
	// 附带一次性登录码：主站登录前端优先用 code 回跳 /oauth-callback 兑换 JWT，
	// 避免 token 进入 URL；token 保留供 OAuth2 授权页（platform-login）等旧流程使用
	if loginCode, codeErr := services.IssuePlatformLoginCode(sess.UserID); codeErr == nil {
		resp["code"] = loginCode
	}
	utils.Success(c, resp)
}

// PasskeyList 列出当前用户 Passkey（用户中心管理）
func PasskeyList(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	rows, err := services.ListPasskeyCredentials(uid)
	if err != nil {
		utils.FailInternal(c, "查询失败")
		return
	}
	utils.Success(c, gin.H{"list": rows})
}

// PasskeyDelete 删除当前用户的一条 Passkey
func PasskeyDelete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid, _ := userID.(uint)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}
	var row models.PasskeyCredential
	if err := database.DB.Where("id = ? AND user_id = ?", id, uid).First(&row).Error; err != nil {
		utils.FailNotFound(c, "Passkey 不存在")
		return
	}
	if err := database.DB.Delete(&row).Error; err != nil {
		utils.FailInternal(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "已删除")
}
