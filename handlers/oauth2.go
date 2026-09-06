package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/providers"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ---------- 授权端点 ----------

// AuthorizeOAuth2 标准 OAuth2 / OIDC 授权端点（authorization code flow）
//
//	GET/POST /api/oauth2/authorize?response_type=code&client_id={appid}&redirect_uri=...&scope=openid&state=...
//
// 可选：type={渠道} 直接发起第三方渠道授权；不传则返回渠道选择页。
// 支持 PKCE（code_challenge + code_challenge_method=S256|plain，可选）。
func AuthorizeOAuth2(c *gin.Context) {
	q := c.Request.URL.Query()
	responseType := q.Get("response_type")
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	scope := q.Get("scope")
	state := q.Get("state")
	nonce := q.Get("nonce")
	challenge := q.Get("code_challenge")
	challengeMethod := q.Get("code_challenge_method")

	if responseType == "" {
		responseType = "code"
	}
	if responseType != "code" {
		oauth2ErrorHTML(c, http.StatusBadRequest, "unsupported_response_type", "仅支持 response_type=code")
		return
	}
	if clientID == "" || redirectURI == "" {
		oauth2ErrorHTML(c, http.StatusBadRequest, "invalid_request", "缺少 client_id 或 redirect_uri")
		return
	}
	if scope == "" {
		scope = "openid profile"
	}

	app, err := services.GetAppByID(clientID)
	if err != nil {
		oauth2ErrorHTML(c, http.StatusBadRequest, "unauthorized_client", "应用不存在或已禁用")
		return
	}
	if app.Mode != services.ModeOAuth2 && app.Mode != services.ModeCompat {
		oauth2ErrorHTML(c, http.StatusBadRequest, "unauthorized_client", "该应用未开启 OAuth2/OIDC 协议")
		return
	}
	if !services.ValidOAuthRedirectURI(app, redirectURI) {
		oauth2ErrorHTML(c, http.StatusBadRequest, "invalid_request", "redirect_uri 不在应用回调白名单内")
		return
	}

	// response_mode：仅支持默认的 query
	if rm := q.Get("response_mode"); rm != "" && rm != "query" {
		oauth2ErrorRedirect(c, redirectURI, state, "unsupported_response_mode", "仅支持 response_mode=query")
		return
	}
	// prompt：本平台无静默授权会话，prompt=none 一律返回 login_required（OIDC Core §3.1.2.1）
	if strings.Contains(q.Get("prompt"), "none") {
		oauth2ErrorRedirect(c, redirectURI, state, "login_required", "无法静默授权，需要用户登录")
		return
	}

	// 校验 PKCE 参数
	if challenge != "" {
		switch challengeMethod {
		case "", "S256", "plain":
		default:
			oauth2ErrorHTML(c, http.StatusBadRequest, "invalid_request", "不支持的 code_challenge_method: "+challengeMethod)
			return
		}
		if challengeMethod == "" {
			challengeMethod = "plain"
		}
	} else if challengeMethod != "" {
		oauth2ErrorHTML(c, http.StatusBadRequest, "invalid_request", "缺少 code_challenge")
		return
	}

	providerName := strings.TrimSpace(q.Get("type"))
	if providerName != "" {
		resolved, ok := services.ResolveType(providerName, app.Mode)
		if !ok || !services.AppSupportsType(app, resolved) {
			oauth2ErrorHTML(c, http.StatusBadRequest, "invalid_request", "该应用未开启此登录类型: "+providerName)
			return
		}
		providerName = resolved
	}

	ctx := services.OAuthCtx{
		ClientID:      app.AppID,
		RedirectURI:   redirectURI,
		Scope:         scope,
		State:         state,
		Nonce:         nonce,
		PKCEChallenge: challenge,
		PKCEMethod:    challengeMethod,
		Provider:      providerName,
	}
	ctxID := services.CreateOAuthCtx(ctx)

	// 未指定渠道 → 返回授权选择页（列出该应用启用的渠道）
	if providerName == "" {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, oauth2ConsentPage(c, app, ctxID))
		return
	}

	prov, ok := loadProvider(providerName)
	if !ok {
		oauth2ErrorHTML(c, http.StatusBadRequest, "temporarily_unavailable", "登录渠道未配置或未启用")
		return
	}
	authURL := prov.GetAuthURL(ctxID)
	if authURL == "" {
		oauth2ErrorHTML(c, http.StatusBadRequest, "temporarily_unavailable", "该渠道不支持网页跳转授权")
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

// oauth2ConsentPage 生成 OAuth2 授权选择页 HTML。
// 除列出应用启用的第三方渠道外，还提供「平台账号登录」（密码 / Passkey），
// 使本平台可作为 IDP / CAS：平台账号登录成功后即签发 OIDC 授权码回跳目标站点。
func oauth2ConsentPage(c *gin.Context, app *models.App, ctxID string) string {
	var types []string
	_ = json.Unmarshal([]byte(app.Types), &types)

	type opt struct {
		Name        string
		DisplayName string
	}
	var opts []opt
	for _, t := range types {
		resolved, ok := services.ResolveType(t, app.Mode)
		if !ok {
			continue
		}
		meta, ok := providers.FindMeta(resolved)
		if !ok {
			continue
		}
		if _, ok := loadProvider(resolved); !ok {
			continue
		}
		opts = append(opts, opt{Name: resolved, DisplayName: meta.DisplayName})
	}

	base := c.Request.URL.Path + "?" + c.Request.URL.RawQuery
	var b strings.Builder
	b.WriteString("<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\">")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
	b.WriteString("<title>授权登录 · OauthGo</title><style>")
	b.WriteString("body{font-family:system-ui,-apple-system,'Segoe UI',sans-serif;background:#f4f4f5;color:#18181b;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;padding:20px}")
	b.WriteString(".card{background:#fff;border:1px solid #e4e4e7;border-radius:12px;padding:28px;width:100%;max-width:420px;box-shadow:0 10px 30px rgba(0,0,0,.06)}")
	b.WriteString("h1{font-size:18px;margin:0 0 4px}h2{font-size:13px;color:#71717a;font-weight:600;margin:22px 0 8px;text-transform:uppercase;letter-spacing:.05em}")
	b.WriteString(".sub{color:#71717a;font-size:13px;margin:0 0 18px}")
	b.WriteString("a.btn{display:flex;align-items:center;justify-content:center;padding:10px 14px;border:1px solid #e4e4e7;border-radius:8px;text-decoration:none;color:#18181b;font-size:14px;margin-bottom:8px;transition:background .15s}")
	b.WriteString("a.btn:hover{background:#f4f4f5}")
	b.WriteString("input{width:100%;box-sizing:border-box;padding:9px 11px;border:1px solid #d4d4d8;border-radius:8px;font-size:14px;margin-bottom:8px}")
	b.WriteString("button{width:100%;padding:10px 14px;border:none;border-radius:8px;background:#18181b;color:#fff;font-size:14px;cursor:pointer}")
	b.WriteString("button.ghost{background:#fff;color:#18181b;border:1px solid #e4e4e7;margin-top:6px}button:disabled{opacity:.5;cursor:not-allowed}")
	b.WriteString(".sep{height:1px;background:#e4e4e7;margin:22px 0 2px}")
	b.WriteString(".msg{font-size:12px;color:#dc2626;margin-top:6px;display:none}.hint{color:#a1a1aa;font-size:12px;margin-top:14px;line-height:1.7}")
	b.WriteString("</style></head><body><div class=\"card\"><h1>授权登录</h1>")
	b.WriteString("<p class=\"sub\">应用 <strong>" + htmlEscape(app.Name) + "</strong> 请求获取您的登录身份</p>")

	// 平台账号登录（IDP / CAS）
	b.WriteString("<h2>使用 OauthGo 账号</h2>")
	b.WriteString("<form id=\"pwdForm\" method=\"post\" action=\"/api/oauth2/platform-login\">")
	b.WriteString("<input type=\"hidden\" name=\"ctx_id\" value=\"" + htmlEscape(ctxID) + "\">")
	b.WriteString("<input id=\"username\" name=\"username\" autocomplete=\"username webauthn\" placeholder=\"用户名 / 邮箱 / 手机号\" required>")
	b.WriteString("<input id=\"password\" name=\"password\" type=\"password\" autocomplete=\"current-password\" placeholder=\"密码（未设置密码可改用 Passkey）\">")
	b.WriteString("<button type=\"submit\" id=\"pwdSubmit\">登录并授权</button>")
	b.WriteString("</form>")
	b.WriteString("<button class=\"ghost\" type=\"button\" id=\"passkeyBtn\">使用 Passkey / 安全密钥</button>")
	b.WriteString("<div class=\"msg\" id=\"msg\"></div>")

	// 第三方渠道
	b.WriteString("<h2>或使用第三方账号</h2>")
	if len(opts) == 0 {
		b.WriteString("<p style=\"color:#71717a;font-size:13px\">该应用未配置可用的第三方登录渠道。</p>")
	} else {
		for _, o := range opts {
			u := base
			sep := "&"
			if !strings.Contains(u, "?") {
				sep = "?"
			}
			u += sep + "type=" + url.QueryEscape(o.Name)
			b.WriteString("<a class=\"btn\" href=\"" + htmlEscape(u) + "\">通过 " + htmlEscape(o.DisplayName) + " 登录</a>")
		}
	}
	b.WriteString("<p class=\"hint\">登录后平台会向该应用返回一次性授权码（code），用于换取访问令牌。授权地址必须与此前发起时一致。</p>")
	b.WriteString("<script>")
	b.WriteString("function showMsg(s){var m=document.getElementById('msg');m.style.display='block';m.textContent=s;}")
	b.WriteString("function b64u(buf){var bytes=new Uint8Array(buf);var s='';for(var i=0;i<bytes.length;i++){s+=String.fromCharCode(bytes[i]);}return btoa(s).replace(/\\+/g,'-').replace(/\\//g,'_').replace(/=+$/g,'');}")
	b.WriteString("function b64toBytes(b64){b64=b64.replace(/\\-/g,'+').replace(/\\_/g,'/');while(b64.length%4){b64+='=';}var bin=atob(b64);var bytes=new Uint8Array(bin.length);for(var i=0;i<bin.length;i++){bytes[i]=bin.charCodeAt(i);}return bytes;}")
	b.WriteString("function credJSON(c){var r=c.response;var o={id:c.id,rawId:b64u(c.rawId),type:c.type,response:{clientDataJSON:b64u(r.clientDataJSON),attestationObject:b64u(r.attestationObject)}};if(r.getTransports){o.transports=r.getTransports();}return o;}")
	b.WriteString("document.getElementById('pwdForm').addEventListener('submit',function(){var sb=document.getElementById('pwdSubmit');sb.disabled=true;sb.textContent='登录中…';var p=document.getElementById('password').value;if(!p){showMsg('请输入密码');event.preventDefault();sb.disabled=false;sb.textContent='登录并授权';}});")
	b.WriteString("document.getElementById('passkeyBtn').addEventListener('click',function(){var btn=this;btn.disabled=true;var u=document.getElementById('username').value;if(!u){showMsg('请先在上方输入用户名');btn.disabled=false;return;}")
	b.WriteString("fetch('/api/auth/passkey/login/begin?username='+encodeURIComponent(u)).then(function(r){if(!r.ok)return r.json().then(function(j){throw new Error(j.message||'begin 失败')});return r.json();}).then(function(data){var opt=data.data.options;if(opt&&opt.challenge){opt.challenge=b64toBytes(opt.challenge);}if(opt&&opt.allowCredentials){for(var i=0;i<opt.allowCredentials.length;i++){opt.allowCredentials[i].id=b64toBytes(opt.allowCredentials[i].id);}}return navigator.credentials.get({publicKey:opt}).then(function(cred){")
	b.WriteString("if(!cred)throw new Error('已取消认证');var body={id:cred.id,rawId:b64u(cred.rawId),type:cred.type,response:{clientDataJSON:b64u(cred.response.clientDataJSON),authenticatorData:b64u(cred.response.authenticatorData),signature:b64u(cred.response.signature)});if(cred.response.userHandle){body.response.userHandle=b64u(cred.response.userHandle);}")
	b.WriteString("return fetch('/api/auth/passkey/login/finish?session_id='+encodeURIComponent(data.data.session_id),{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}).then(function(r){return r.json().then(function(j){if(!r.ok)throw new Error(j.message||'finish 失败');return j.data.token;});});}).then(function(token){")
	b.WriteString("var f=document.createElement('form');f.method='post';f.action='/api/oauth2/platform-login';function add(n,v){var i=document.createElement('input');i.type='hidden';i.name=n;i.value=v;f.appendChild(i);}add('ctx_id','" + htmlEscape(ctxID) + "');add('token',token);document.body.appendChild(f);f.submit();}).catch(function(e){showMsg(e.message||'Passkey 登录失败');btn.disabled=false;});});")
	b.WriteString("</script>")
	b.WriteString("</div></body></html>")
	return b.String()
}

// htmlEscape HTML 转义
func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")
	return r.Replace(s)
}

// oauth2ErrorHTML 在 redirect_uri 不可信时输出错误页
func oauth2ErrorHTML(c *gin.Context, status int, code, desc string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(status, "<!DOCTYPE html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>授权失败</title></head><body style=\"font-family:system-ui;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f4f4f5;color:#18181b\"><div style=\"background:#fff;border:1px solid #e4e4e7;border-radius:12px;padding:32px;max-width:420px\"><h1 style=\"font-size:18px;margin:0 0 8px\">授权失败 ("+htmlEscape(code)+")</h1><p style=\"color:#71717a;font-size:14px;margin:0\">"+htmlEscape(desc)+"</p></div></body></html>")
}

// oauth2ErrorRedirect 在 redirect_uri 可信时按 RFC 6749 以 query 参数回传错误
func oauth2ErrorRedirect(c *gin.Context, redirectURI, state, code, desc string) {
	q := url.Values{}
	q.Set("error", code)
	if desc != "" {
		q.Set("error_description", desc)
	}
	if state != "" {
		q.Set("state", state)
	}
	c.Redirect(http.StatusFound, redirectURI+"?"+q.Encode())
}

// ---------- 渠道授权回调 ----------

// handleOAuthCallback OAuth2/OIDC 授权回调：渠道授权完成后签发一次性授权码并跳回目标站点
func handleOAuthCallback(c *gin.Context, ctx services.OAuthCtx, providerName, providerCode string) {
	redirectErr := func(desc string) {
		oauth2ErrorRedirect(c, ctx.RedirectURI, ctx.State, "access_denied", desc)
	}
	if providerName != ctx.Provider {
		redirectErr("回调渠道与授权请求不一致")
		return
	}
	prov, ok := loadProvider(providerName)
	if !ok {
		redirectErr("登录渠道未配置或未启用")
		return
	}
	info, err := prov.GetUserInfo(providerCode)
	if err != nil {
		redirectErr("获取用户信息失败")
		return
	}
	user, err := bindProviderUser(providerName, info)
	if err != nil {
		redirectErr("登录失败：" + err.Error())
		return
	}
	// 记录登录日志
	_ = services.RecordLogin(0, "", models.LoginRecord{
		OpenID:    info.OpenID,
		Nickname:  info.Nickname,
		Avatar:    info.Avatar,
		Platform:  providerName,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    1,
	})

	code := models.OAuthCode{
		Code:          strings.ToUpper(utils.RandomString(32)),
		ClientID:      ctx.ClientID,
		UserID:        user.ID,
		Scope:         ctx.Scope,
		RedirectURI:   ctx.RedirectURI,
		PKCEChallenge: ctx.PKCEChallenge,
		PKCEMethod:    ctx.PKCEMethod,
		ExpiresAt:     time.Now().Add(services.OAuthCodeTTL),
	}
	if err := database.DB.Create(&code).Error; err != nil {
		redirectErr("签发授权码失败")
		return
	}

	q := url.Values{}
	q.Set("code", code.Code)
	if ctx.State != "" {
		q.Set("state", ctx.State)
	}
	c.Redirect(http.StatusFound, ctx.RedirectURI+"?"+q.Encode())
}

// ---------- Token 端点 ----------

// TokenOAuth2 标准 OAuth2 Token 端点（RFC 6749 §3.2）
// 支持 grant_type=authorization_code 与 grant_type=refresh_token（轮换制）
func TokenOAuth2(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_request", "无法解析请求体")
		return
	}
	clientID, clientSecret, ok := oauth2ClientCredentials(c)
	if !ok {
		oauth2TokenError(c, http.StatusUnauthorized, "invalid_client", "客户端认证失败")
		return
	}
	app, err := services.GetAppByID(clientID)
	if err != nil || app.AppKey != clientSecret {
		oauth2TokenError(c, http.StatusUnauthorized, "invalid_client", "客户端认证失败")
		return
	}
	if app.Mode != services.ModeOAuth2 && app.Mode != services.ModeCompat {
		oauth2TokenError(c, http.StatusForbidden, "unauthorized_client", "该应用未开启 OAuth2/OIDC 协议")
		return
	}

	grantType := c.Request.Form.Get("grant_type")
	switch grantType {
	case "authorization_code":
		oauth2TokenByCode(c, app)
	case "refresh_token":
		oauth2TokenByRefresh(c, app)
	default:
		oauth2TokenError(c, http.StatusBadRequest, "unsupported_grant_type", "不支持的 grant_type")
	}
}

// oauth2ClientCredentials 从 Authorization Basic 或表单提取 client_id / client_secret
func oauth2ClientCredentials(c *gin.Context) (string, string, bool) {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Basic ") {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err == nil {
			parts := strings.SplitN(string(raw), ":", 2)
			if len(parts) == 2 && parts[0] != "" {
				return parts[0], parts[1], true
			}
		}
		return "", "", false
	}
	id := c.Request.Form.Get("client_id")
	secret := c.Request.Form.Get("client_secret")
	if id == "" {
		return "", "", false
	}
	return id, secret, true
}

func oauth2TokenByCode(c *gin.Context, app *models.App) {
	codeVal := c.Request.Form.Get("code")
	if codeVal == "" {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_request", "缺少 code")
		return
	}
	var code models.OAuthCode
	if err := database.DB.Where("code = ? AND client_id = ? AND used = ?", codeVal, app.AppID, false).First(&code).Error; err != nil {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "授权码无效或已使用")
		return
	}
	if time.Now().After(code.ExpiresAt) {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "授权码已过期")
		return
	}
	if ru := c.Request.Form.Get("redirect_uri"); ru != "" && ru != code.RedirectURI {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "redirect_uri 不匹配")
		return
	}
	// PKCE 校验（可选）
	if code.PKCEChallenge != "" {
		verifier := c.Request.Form.Get("code_verifier")
		if verifier == "" {
			oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "缺少 code_verifier")
			return
		}
		if !oauth2VerifyPKCE(code.PKCEMethod, code.PKCEChallenge, verifier) {
			oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "code_verifier 校验失败")
			return
		}
	}
	// 原子占用：防止并发请求把同一个授权码兑换两次（校验全部通过后再占用，失败尝试不消耗 code）
	occupy := database.DB.Model(&models.OAuthCode{}).Where("id = ? AND used = ?", code.ID, false).Update("used", true)
	if occupy.Error != nil || occupy.RowsAffected == 0 {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "授权码已使用")
		return
	}
	issueOAuth2Tokens(c, app, code.UserID, code.Scope)
}

func oauth2VerifyPKCE(method, challenge, verifier string) bool {
	switch method {
	case "plain":
		return challenge == verifier
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		return challenge == base64.RawURLEncoding.EncodeToString(sum[:])
	default:
		return false
	}
}

func oauth2TokenByRefresh(c *gin.Context, app *models.App) {
	refresh := c.Request.Form.Get("refresh_token")
	if refresh == "" {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_request", "缺少 refresh_token")
		return
	}
	var rt models.OAuthRefreshToken
	if err := database.DB.Where("token = ? AND client_id = ?", refresh, app.AppID).First(&rt).Error; err != nil {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "refresh_token 无效")
		return
	}
	if time.Now().After(rt.ExpiresAt) {
		database.DB.Delete(&rt)
		oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "refresh_token 已过期")
		return
	}
	// 轮换：原子删除旧刷新令牌；删除数为 0 说明已被并发请求轮换，旧令牌立即失效
	del := database.DB.Where("id = ?", rt.ID).Delete(&models.OAuthRefreshToken{})
	if del.Error != nil || del.RowsAffected == 0 {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_grant", "refresh_token 已失效")
		return
	}
	issueOAuth2Tokens(c, app, rt.UserID, rt.Scope)
}

// issueOAuth2Tokens 为授权用户签发 access / refresh / id_token
func issueOAuth2Tokens(c *gin.Context, app *models.App, userID uint, scope string) {
	at := models.OAuthAccessToken{
		Token:     utils.RandomString(48),
		ClientID:  app.AppID,
		UserID:    userID,
		Scope:     scope,
		ExpiresAt: time.Now().Add(services.OAuthAccessTokenTTL),
	}
	if err := database.DB.Create(&at).Error; err != nil {
		oauth2TokenError(c, http.StatusInternalServerError, "server_error", "签发访问令牌失败")
		return
	}

	resp := gin.H{
		"access_token": at.Token,
		"token_type":   "Bearer",
		"expires_in":   int(services.OAuthAccessTokenTTL / time.Second),
		"scope":        scope,
	}
	// refresh_token：仅在应用开启「签发刷新令牌」时签发
	// （offline_access scope 不再绕过该开关，避免越权获得长期令牌）
	wantRefresh := app.EnableRefresh
	if wantRefresh {
		rt := models.OAuthRefreshToken{
			Token:     utils.RandomString(48),
			ClientID:  app.AppID,
			UserID:    userID,
			Scope:     scope,
			ExpiresAt: time.Now().Add(services.OAuthRefreshTokenTTL),
		}
		if err := database.DB.Create(&rt).Error; err != nil {
			oauth2TokenError(c, http.StatusInternalServerError, "server_error", "签发刷新令牌失败")
			return
		}
		resp["refresh_token"] = rt.Token
	}
	// id_token：scope 含 openid 时签发 RS256 JWT
	if strings.Contains(scope, "openid") {
		idToken, err := signOAuthIDToken(app.AppID, userID, scope, "")
		if err == nil {
			resp["id_token"] = idToken
		}
	}
	oauth2TokenOK(c, resp)
}

func oauth2TokenOK(c *gin.Context, data gin.H) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, data)
}

func oauth2TokenError(c *gin.Context, status int, code, desc string) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(status, gin.H{"error": code, "error_description": desc})
}

// ---------- id_token / JWKS / Discovery ----------

// signOAuthIDToken 签发 OIDC id_token（RS256，issuer=HOST）
func signOAuthIDToken(clientID string, userID uint, scope, nonce string) (string, error) {
	key, err := services.EnsureOAuthSigningKey()
	if err != nil {
		return "", err
	}
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":                services.OAuthIssuer(),
		"sub":                fmt.Sprintf("%d", user.ID),
		"aud":                clientID,
		"exp":                now.Add(time.Hour).Unix(),
		"iat":                now.Unix(),
		"auth_time":          now.Unix(),
		"name":               user.Nickname,
		"preferred_username": user.Username,
	}
	if user.Avatar != "" {
		claims["picture"] = user.Avatar
	}
	if user.Email != "" {
		claims["email"] = user.Email
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = services.OAuthKeyID
	return token.SignedString(key)
}

// OAuth2JWKS 返回平台 RSA 公钥（供第三方验证 id_token）
func OAuth2JWKS(c *gin.Context) {
	key, err := services.EnsureOAuthSigningKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": err.Error()})
		return
	}
	pub := &key.PublicKey
	c.JSON(http.StatusOK, gin.H{
		"keys": []gin.H{{
			"kty": "RSA",
			"use": "sig",
			"alg": "RS256",
			"kid": services.OAuthKeyID,
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString([]byte{0x01, 0x00, 0x01}),
		}},
	})
}

// OAuth2Discovery OIDC Discovery 文档（RFC 8414 / OIDC Discovery）
// 端点统一发布在 issuer（HOST）根路径下，保证 SDK 能按 issuer 自动发现。
func OAuth2Discovery(c *gin.Context) {
	iss := services.OAuthIssuer()
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, gin.H{
		"issuer":                                iss,
		"authorization_endpoint":                iss + "/authorize",
		"token_endpoint":                        iss + "/token",
		"userinfo_endpoint":                     iss + "/userinfo",
		"jwks_uri":                              iss + "/jwks",
		"revocation_endpoint":                   iss + "/revoke",
		"introspection_endpoint":                iss + "/introspect",
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
		"scopes_supported":                      []string{"openid", "profile", "email", "phone", "offline_access"},
		"claims_supported":                      []string{"sub", "name", "preferred_username", "nickname", "picture", "email", "email_verified", "phone", "phone_number_verified"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
	})
}

// OAuth2Userinfo 受保护资源端点（OIDC Core §5.3）：
// 按 Bearer access_token 返回用户信息，claims 依据授权 scope 过滤。
func OAuth2Userinfo(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	tokenStr := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if tokenStr == "" || tokenStr == c.GetHeader("Authorization") {
		oauth2TokenError(c, http.StatusUnauthorized, "invalid_token", "缺少 Bearer access_token")
		return
	}
	var at models.OAuthAccessToken
	if err := database.DB.Where("token = ?", tokenStr).First(&at).Error; err != nil {
		oauth2TokenError(c, http.StatusUnauthorized, "invalid_token", "access_token 无效")
		return
	}
	if time.Now().After(at.ExpiresAt) {
		database.DB.Delete(&at)
		oauth2TokenError(c, http.StatusUnauthorized, "invalid_token", "access_token 已过期")
		return
	}
	var user models.User
	if err := database.DB.First(&user, at.UserID).Error; err != nil {
		oauth2TokenError(c, http.StatusUnauthorized, "invalid_token", "用户不存在")
		return
	}

	scopeSet := oauth2ScopeSet(at.Scope)
	claims := gin.H{
		"sub":   fmt.Sprintf("%d", user.ID),
		"iss":   services.OAuthIssuer(),
		"aud":   []string{at.ClientID},
		"azp":   at.ClientID,
		"exp":   at.ExpiresAt.Unix(),
		"iat":   at.CreatedAt.Unix(),
		"scope": at.Scope,
	}
	// scope=profile：个人资料
	if scopeSet["profile"] {
		claims["name"] = user.Nickname
		claims["preferred_username"] = user.Username
		claims["nickname"] = user.Nickname
		if user.Avatar != "" {
			claims["picture"] = user.Avatar
		}
		// 附带绑定的第三方账号（本平台扩展，便于聚合场景获取 openid / unionid）
		var accounts []models.ProviderAccount
		database.DB.Where("user_id = ?", user.ID).Find(&accounts)
		if len(accounts) > 0 {
			list := make([]gin.H, 0, len(accounts))
			for _, a := range accounts {
				list = append(list, gin.H{
					"provider": a.Provider,
					"openid":   a.OpenID,
					"unionid":  a.UnionID,
					"nickname": a.Nickname,
					"avatar":   a.Avatar,
				})
			}
			claims["accounts"] = list
		}
	}
	// scope=email
	if scopeSet["email"] && user.Email != "" {
		// 平台不维护独立邮箱验证状态，故不输出 email_verified 以避免误导
		claims["email"] = user.Email
	}
	// scope=phone
	if scopeSet["phone"] && user.Phone != "" {
		claims["phone"] = user.Phone
	}
	c.JSON(http.StatusOK, claims)
}

// oauth2ScopeSet 解析空格分隔的 scope 为集合
func oauth2ScopeSet(scope string) map[string]bool {
	m := map[string]bool{}
	for _, s := range strings.Fields(scope) {
		m[s] = true
	}
	return m
}

// OAuth2Revoke 撤销 refresh_token / access_token（RFC 7009）
// 要求 client 认证；仅撤销属于该 client 的令牌（RFC 7009 §2.1 建议对不属于本 client 的令牌不执行撤销）
func OAuth2Revoke(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	if err := c.Request.ParseForm(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	clientID, clientSecret, ok := oauth2ClientCredentials(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client", "error_description": "客户端认证失败"})
		return
	}
	app, err := services.GetAppByID(clientID)
	if err != nil || app.AppKey != clientSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client", "error_description": "客户端认证失败"})
		return
	}
	tokenStr := c.Request.Form.Get("token")
	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "缺少 token"})
		return
	}
	hint := c.Request.Form.Get("token_type_hint")
	if hint != "access_token" {
		database.DB.Where("token = ? AND client_id = ?", tokenStr, app.AppID).Delete(&models.OAuthRefreshToken{})
	}
	if hint != "refresh_token" {
		database.DB.Where("token = ? AND client_id = ?", tokenStr, app.AppID).Delete(&models.OAuthAccessToken{})
	}
	c.JSON(http.StatusOK, gin.H{})
}

// OAuth2Introspect 令牌内省端点（RFC 7662）
// 允许受保护资源方使用 client 凭据校验 access_token / refresh_token 是否有效。
func OAuth2Introspect(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	if err := c.Request.ParseForm(); err != nil {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_request", "无法解析请求体")
		return
	}
	clientID, clientSecret, ok := oauth2ClientCredentials(c)
	if !ok {
		oauth2TokenError(c, http.StatusUnauthorized, "invalid_client", "客户端认证失败")
		return
	}
	app, err := services.GetAppByID(clientID)
	if err != nil || app.AppKey != clientSecret {
		oauth2TokenError(c, http.StatusUnauthorized, "invalid_client", "客户端认证失败")
		return
	}
	if app.Mode != services.ModeOAuth2 && app.Mode != services.ModeCompat {
		oauth2TokenError(c, http.StatusForbidden, "unauthorized_client", "该应用未开启 OAuth2/OIDC 协议")
		return
	}
	tokenStr := c.Request.Form.Get("token")
	if tokenStr == "" {
		oauth2TokenError(c, http.StatusBadRequest, "invalid_request", "缺少 token")
		return
	}
	_ = c.Request.Form.Get("token_type_hint")

	var user models.User
	now := time.Now()

	// RFC 7662 §2：授权服务器必须拒绝不属于该 client 的令牌查询（返回 active=false，不泄露令牌是否存在）
	var at models.OAuthAccessToken
	if err := database.DB.Where("token = ?", tokenStr).First(&at).Error; err == nil {
		if at.ClientID != app.AppID {
			oauth2TokenOK(c, gin.H{"active": false})
			return
		}
		active := !now.After(at.ExpiresAt)
		if !active {
			database.DB.Delete(&at)
		}
		if err := database.DB.First(&user, at.UserID).Error; err != nil {
			active = false
		}
		oauth2TokenOK(c, gin.H{
			"active":     active,
			"scope":      at.Scope,
			"client_id":  at.ClientID,
			"token_type": "access_token",
			"sub":        fmt.Sprintf("%d", at.UserID),
			"exp":        at.ExpiresAt.Unix(),
			"iat":        at.CreatedAt.Unix(),
			"username":   user.Username,
		})
		return
	}

	var rt models.OAuthRefreshToken
	if err := database.DB.Where("token = ?", tokenStr).First(&rt).Error; err == nil {
		if rt.ClientID != app.AppID {
			oauth2TokenOK(c, gin.H{"active": false})
			return
		}
		active := !now.After(rt.ExpiresAt)
		if !active {
			database.DB.Delete(&rt)
		}
		if err := database.DB.First(&user, rt.UserID).Error; err != nil {
			active = false
		}
		oauth2TokenOK(c, gin.H{
			"active":     active,
			"scope":      rt.Scope,
			"client_id":  rt.ClientID,
			"token_type": "refresh_token",
			"sub":        fmt.Sprintf("%d", rt.UserID),
			"exp":        rt.ExpiresAt.Unix(),
			"iat":        rt.CreatedAt.Unix(),
			"username":   user.Username,
		})
		return
	}

	oauth2TokenOK(c, gin.H{"active": false})
}

// OAuth2PlatformLogin 平台账号授权桥接（IDP / CAS 语义）：
// 使用平台账号（用户名/密码）或平台 JWT（Passkey 登录产物）换取 OAuth2/OIDC 授权码。
// POST /api/oauth2/platform-login  （application/x-www-form-urlencoded 或 JSON）
// 参数：ctx_id 必填 + (username+password) 或 token
func OAuth2PlatformLogin(c *gin.Context) {
	ctxID := strings.TrimSpace(c.Request.FormValue("ctx_id"))
	username := strings.TrimSpace(c.Request.FormValue("username"))
	password := c.Request.FormValue("password")
	tokenStr := strings.TrimSpace(c.Request.FormValue("token"))

	// JSON body 兼容
	if ctxID == "" && c.GetHeader("Content-Type") != "" && strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		var body struct {
			CtxID    string `json:"ctx_id"`
			Username string `json:"username"`
			Password string `json:"password"`
			Token    string `json:"token"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			ctxID = body.CtxID
			username = strings.TrimSpace(body.Username)
			password = body.Password
			tokenStr = body.Token
		}
	}
	if ctxID == "" {
		oauth2ErrorHTML(c, http.StatusBadRequest, "invalid_request", "缺少 ctx_id，请重新发起授权")
		return
	}
	ctx, ok := services.ResolveOAuthCtx(ctxID)
	if !ok {
		oauth2ErrorHTML(c, http.StatusBadRequest, "invalid_request", "授权会话无效或已过期，请重新发起授权")
		return
	}

	// 平台账号认证
	var user models.User
	if tokenStr != "" {
		claims, err := utils.ParseToken(tokenStr)
		if err != nil {
			oauth2ErrorHTML(c, http.StatusBadRequest, "access_denied", "登录令牌无效或已过期，请重新登录")
			return
		}
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			oauth2ErrorHTML(c, http.StatusBadRequest, "access_denied", "用户不存在")
			return
		}
	} else if username != "" && password != "" {
		u, err := findUserByAccount(username)
		if err != nil || !utils.CheckPassword(u.Password, password) {
			oauth2ErrorHTML(c, http.StatusBadRequest, "access_denied", "用户名或密码错误")
			return
		}
		user = *u
	} else {
		oauth2ErrorHTML(c, http.StatusBadRequest, "invalid_request", "缺少平台账号凭据（username/password 或 token）")
		return
	}

	// 应用仍有效
	app, err := services.GetAppByID(ctx.ClientID)
	if err != nil || (app.Mode != services.ModeOAuth2 && app.Mode != services.ModeCompat) {
		oauth2ErrorHTML(c, http.StatusBadRequest, "unauthorized_client", "应用不存在或未开启 OAuth2/OIDC 协议")
		return
	}

	code := models.OAuthCode{
		Code:          strings.ToUpper(utils.RandomString(32)),
		ClientID:      ctx.ClientID,
		UserID:        user.ID,
		Scope:         ctx.Scope,
		RedirectURI:   ctx.RedirectURI,
		PKCEChallenge: ctx.PKCEChallenge,
		PKCEMethod:    ctx.PKCEMethod,
		ExpiresAt:     time.Now().Add(services.OAuthCodeTTL),
	}
	if err := database.DB.Create(&code).Error; err != nil {
		oauth2ErrorHTML(c, http.StatusBadRequest, "server_error", "签发授权码失败")
		return
	}
	// 审计：平台账号授权（IDP/CAS）登录记录
	_ = services.RecordLogin(0, "", models.LoginRecord{
		Username:  user.Username,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Platform:  "platform_account",
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Status:    1,
	})

	q := url.Values{}
	q.Set("code", code.Code)
	if ctx.State != "" {
		q.Set("state", ctx.State)
	}
	c.Redirect(http.StatusFound, ctx.RedirectURI+"?"+q.Encode())
}

// findUserByAccount 按用户名/邮箱/手机号查找平台账号
func findUserByAccount(account string) (*models.User, error) {
	var user models.User
	for _, field := range []string{"username", "email", "phone"} {
		if err := database.DB.Where(field+" = ?", account).First(&user).Error; err == nil {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("账号不存在")
}
