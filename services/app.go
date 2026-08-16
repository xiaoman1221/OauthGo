package services

import (
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/providers"
	"OauthGo/utils"
)

// App 模式常量
const (
	ModeRainbow = "rainbow" // 仅彩虹聚合登录协议
	ModeREST    = "rest"    // 仅 REST 风格接口
	ModeCompat  = "compat"  // 兼容（两种接口均支持）
)

// rainbowTypeMap 彩虹聚合登录类型 -> 平台 provider name
// 参考彩虹官方文档（u.cccyun.cc/doc.php）：qq / wx / alipay / sina / baidu / douyin / dingtalk / gitee / wework ...
var rainbowTypeMap = map[string]string{
	"qq":       "qq",
	"wx":       "wechat",
	"alipay":   "alipay",
	"sina":     "weibo",
	"baidu":    "baidu",
	"douyin":   "douyin",
	"dingtalk": "dingtalk",
	"gitee":    "gitee",
	"wework":   "wecom",
}

var providerToRainbowType = func() map[string]string {
	m := make(map[string]string, len(rainbowTypeMap))
	for k, v := range rainbowTypeMap {
		m[v] = k
	}
	return m
}()

// ResolveType 解析登录类型为 provider name。
// rainbow 模式按彩虹类型名映射；rest 模式直接使用 provider name；compat 两种都尝试。
func ResolveType(typeName, mode string) (providerName string, ok bool) {
	if typeName == "" {
		return "", false
	}
	if mode == ModeREST {
		_, ok := providers.FindMeta(typeName)
		return typeName, ok
	}
	if p, exists := rainbowTypeMap[typeName]; exists {
		return p, true
	}
	if mode == ModeCompat {
		if _, ok := providers.FindMeta(typeName); ok {
			return typeName, true
		}
	}
	return "", false
}

// RainbowTypeName 返回 provider name 对应的彩虹类型名（无对应时返回 provider name 本身）
func RainbowTypeName(providerName string) string {
	if t, ok := providerToRainbowType[providerName]; ok {
		return t
	}
	return providerName
}

// AppSession 一次应用授权登录会话
type AppSession struct {
	AppID       string
	Provider    string
	Type        string // 请求中的类型名（与协议一致）
	RedirectURI string
	CreatedAt   time.Time
}

var appSessionStore = struct {
	sync.RWMutex
	m map[string]AppSession
}{m: map[string]AppSession{}}

const appSessionTTL = 10 * time.Minute

// startAppSessionJanitor 定期清理过期会话，避免未完成的登录会话无限累积
func startAppSessionJanitor() {
	go func() {
		ticker := time.NewTicker(appSessionTTL)
		defer ticker.Stop()
		for range ticker.C {
			appSessionStore.Lock()
			for s, sess := range appSessionStore.m {
				if time.Since(sess.CreatedAt) > appSessionTTL {
					delete(appSessionStore.m, s)
				}
			}
			appSessionStore.Unlock()
		}
	}()
}

func init() {
	startAppSessionJanitor()
}

// CreateAppSession 创建应用登录会话并返回 state（由渠道回调带回）
func CreateAppSession(appID, providerName, typeName, redirectURI string) string {
	state := utils.RandomString(32)
	appSessionStore.Lock()
	appSessionStore.m[state] = AppSession{
		AppID:       appID,
		Provider:    providerName,
		Type:        typeName,
		RedirectURI: redirectURI,
		CreatedAt:   time.Now(),
	}
	appSessionStore.Unlock()
	return state
}

// ResolveAppSession 按 state 查找并消费应用登录会话（一次性）
func ResolveAppSession(state string) (AppSession, bool) {
	appSessionStore.Lock()
	defer appSessionStore.Unlock()
	s, ok := appSessionStore.m[state]
	if !ok {
		return AppSession{}, false
	}
	delete(appSessionStore.m, state)
	if time.Since(s.CreatedAt) > appSessionTTL {
		return AppSession{}, false
	}
	return s, true
}

// GetAppByID 按 appid 查找启用中的应用
func GetAppByID(appID string) (*models.App, error) {
	if appID == "" {
		return nil, errors.New("缺少 appid")
	}
	var app models.App
	if err := database.DB.Where("app_id = ? AND status = ?", appID, 1).First(&app).Error; err != nil {
		return nil, errors.New("应用不存在或已禁用")
	}
	return &app, nil
}

// AppSupportsType 判断应用是否支持该类型（provider name）
func AppSupportsType(app *models.App, providerName string) bool {
	var types []string
	if err := json.Unmarshal([]byte(app.Types), &types); err != nil {
		return false
	}
	for _, t := range types {
		if t == providerName {
			return true
		}
	}
	return false
}

// NormalizeDomains 规整回调白名单域名列表：按换行/逗号/分号/空格分隔，去重并校验。
// 返回规范化后的字符串（每行一个域名）与错误信息。
func NormalizeDomains(raw string) (string, string) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';' || r == ' ' || r == '，' || r == '；'
	})
	seen := map[string]bool{}
	list := make([]string, 0, len(parts))
	for _, p := range parts {
		d := utils.ExtractDomain(p)
		if d == "" {
			return "", "回调域名不合法: " + p
		}
		if !seen[d] {
			seen[d] = true
			list = append(list, d)
		}
	}
	return strings.Join(list, "\n"), ""
}

// ValidateRedirect 校验回跳地址是否在白名单域名内（区分子域名）。
// 域名匹配规则：redirect_uri 的 host 等于白名单域名，或以「.域名」结尾（即其子域名）；
// 未配置白名单时允许任意 http(s) 地址。
func ValidateRedirect(redirectURI string, app *models.App) bool {
	if redirectURI == "" {
		return false
	}
	u, err := url.Parse(redirectURI)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	domains := []string{}
	for _, line := range strings.Split(app.Domains, "\n") {
		line = strings.ToLower(strings.TrimSpace(line))
		if line != "" {
			domains = append(domains, line)
		}
	}
	if len(domains) == 0 {
		return true
	}
	for _, d := range domains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

// VerifySign 校验服务端签名。
// 签名规则：将除 sign 外的所有参数按 key 升序排列为 k1=v1&k2=v2，拼接 &key=AppKey 后取 MD5。
func VerifySign(params map[string]string, appKey, sign string) bool {
	if sign == "" || appKey == "" {
		return false
	}
	return sign == ComputeSign(params, appKey)
}

// ComputeSign 计算签名
func ComputeSign(params map[string]string, appKey string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sign" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	raw := strings.Join(parts, "&") + "&key=" + appKey
	return utils.MD5(raw)
}

// IssueLoginCode 为用户认证结果签发一次性授权码（code）
func IssueLoginCode(appID, typeName, providerName string, info *providers.UserInfo, ip string) (*models.LoginCode, error) {
	code := strings.ToUpper(utils.RandomString(24))
	record := models.LoginCode{
		Code:        code,
		AppID:       appID,
		Type:        typeName,
		Provider:    providerName,
		OpenID:      info.OpenID,
		UnionID:     info.UnionID,
		Nickname:    info.Nickname,
		Avatar:      info.Avatar,
		Email:       info.Email,
		Gender:      extraString(info.Extra, "gender"),
		Location:    extraString(info.Extra, "location"),
		AccessToken: info.AccessToken,
		IP:          ip,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
	if err := database.DB.Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// ExchangeCode 使用授权码换取用户信息（一次性，成功后标记已使用）
func ExchangeCode(appID, code string) (*models.LoginCode, error) {
	if code == "" {
		return nil, errors.New("缺少 code")
	}
	var record models.LoginCode
	if err := database.DB.Where("code = ? AND app_id = ?", code, appID).First(&record).Error; err != nil {
		return nil, errors.New("code 无效")
	}
	if record.Used {
		return nil, errors.New("code 已使用")
	}
	if time.Now().After(record.ExpiresAt) {
		return nil, errors.New("code 已过期")
	}
	database.DB.Model(&record).Update("used", true)
	return &record, nil
}

// QueryUserBySocialUID 通过第三方 UID 查询最近一次成功登录的用户信息
func QueryUserBySocialUID(appID, typeName, socialUID string) (*models.LoginRecord, error) {
	if socialUID == "" {
		return nil, errors.New("缺少 social_uid")
	}
	var app models.App
	if err := database.DB.Where("app_id = ?", appID).First(&app).Error; err != nil {
		return nil, errors.New("应用不存在")
	}

	var record models.LoginRecord
	if err := database.DB.Where("app_id = ? AND open_id = ?", app.ID, socialUID).
		Order("id desc").First(&record).Error; err != nil {
		return nil, errors.New("未查询到用户登录记录")
	}
	return &record, nil
}

func extraString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
