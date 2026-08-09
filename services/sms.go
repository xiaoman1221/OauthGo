package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// SMSProviderName 当前短信服务商（none / aliyun / tencent / smsbao）
func SMSProviderName() string {
	return GetSetting("sms_provider", "none")
}

// SendVerifyCodeSMS 发送验证码短信
func SendVerifyCodeSMS(phone, code string) error {
	switch SMSProviderName() {
	case "aliyun":
		return sendAliyunSMS(phone, code)
	case "tencent":
		return sendTencentSMS(phone, code)
	case "smsbao":
		content := fmt.Sprintf("【%s】您的验证码是%s，请在%s分钟内完成验证，请勿泄露给他人。",
			GetSetting("sms_sign_name", ""), code, GetSetting("code_expire_minutes", "10"))
		return sendSMSBao(phone, content)
	default:
		return fmt.Errorf("未配置短信服务商")
	}
}

// SendTestSMS 发送测试短信（用于系统设置中的发信测试）
func SendTestSMS(phone string) error {
	code := "123456"
	switch SMSProviderName() {
	case "aliyun":
		return sendAliyunSMS(phone, code)
	case "tencent":
		return sendTencentSMS(phone, code)
	case "smsbao":
		content := fmt.Sprintf("【%s】OauthGo 短信发送测试：如果您收到此短信，说明短信服务配置正常。",
			GetSetting("sms_sign_name", ""))
		return sendSMSBao(phone, content)
	default:
		return fmt.Errorf("未配置短信服务商")
	}
}

// sendAliyunSMS 阿里云短信（RPC 签名）
func sendAliyunSMS(phone, code string) error {
	params := map[string]string{
		"AccessKeyId":      GetSetting("sms_access_key_id", ""),
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     phone,
		"RegionId":         GetSetting("sms_region_id", "cn-hangzhou"),
		"SignName":         GetSetting("sms_sign_name", ""),
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   randomHex(16),
		"SignatureVersion": "1.0",
		"TemplateCode":     GetSetting("sms_aliyun_template_code", ""),
		"TemplateParam":    fmt.Sprintf(`{"code":"%s"}`, code),
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Version":          "2017-05-25",
	}
	if params["AccessKeyId"] == "" || params["SignName"] == "" || params["TemplateCode"] == "" {
		return fmt.Errorf("阿里云短信配置不完整")
	}

	params["Signature"] = aliyunSign(params, GetSetting("sms_access_key_secret", ""))

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	var resp struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
	}
	if err := postJSONForm("https://dysmsapi.aliyuncs.com/", form, &resp); err != nil {
		return err
	}
	if resp.Code != "OK" {
		return fmt.Errorf("阿里云短信发送失败: %s %s", resp.Code, resp.Message)
	}
	return nil
}

func aliyunSign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, aliyunEncode(k)+"="+aliyunEncode(params[k]))
	}
	query := strings.Join(parts, "&")

	strToSign := "POST&" + aliyunEncode("/") + "&" + aliyunEncode(query)
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte(strToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func aliyunEncode(s string) string {
	escaped := url.QueryEscape(s)
	escaped = strings.ReplaceAll(escaped, "+", "%20")
	escaped = strings.ReplaceAll(escaped, "*", "%2A")
	return escaped
}

// sendTencentSMS 腾讯云短信（TC3-HMAC-SHA256 签名）
func sendTencentSMS(phone, code string) error {
	secretID := GetSetting("sms_access_key_id", "")
	secretKey := GetSetting("sms_access_key_secret", "")
	sdkAppID := GetSetting("sms_tencent_sdk_app_id", "")
	templateID := GetSetting("sms_tencent_template_id", "")
	signName := GetSetting("sms_sign_name", "")

	if secretID == "" || sdkAppID == "" || templateID == "" || signName == "" {
		return fmt.Errorf("腾讯云短信配置不完整")
	}

	host := "sms.tencentcloudapi.com"
	service := "sms"
	action := "SendSms"
	version := "2021-01-11"
	region := GetSetting("sms_region_id", "ap-guangzhou")
	now := time.Now()
	date := now.Format("2006-01-02")
	timestamp := fmt.Sprintf("%d", now.Unix())

	payloadBytes, err := json.Marshal(map[string]interface{}{
		"PhoneNumberSet":   []string{"+86" + phone},
		"SmsSdkAppId":      sdkAppID,
		"SignName":         signName,
		"TemplateId":       templateID,
		"TemplateParamSet": []string{code},
	})
	if err != nil {
		return err
	}

	canonicalRequest := strings.Join([]string{
		"POST",
		"/",
		"",
		"content-type:application/json; charset=utf-8\nhost:" + host + "\n",
		"",
		sha256Hex(payloadBytes),
	}, "\n")

	credentialScope := date + "/" + service + "/tc3_request"
	stringToSign := "TC3-HMAC-SHA256\n" + timestamp + "\n" + credentialScope + "\n" + sha256Hex([]byte(canonicalRequest))

	secretDate := hmacSHA256([]byte(secretKey), date)
	secretService := hmacSHA256(secretDate, service)
	secretSigning := hmacSHA256(secretService, "tc3_request")
	signature := hex.EncodeToString(hmacSHA256(secretSigning, stringToSign))

	authorization := fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=content-type;host, Signature=%s",
		secretID, credentialScope, signature)

	req, err := http.NewRequest(http.MethodPost, "https://"+host+"/", bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", host)
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", timestamp)
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Region", region)
	req.Header.Set("Authorization", authorization)

	httpClient2 := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient2.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Response struct {
			Error struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error"`
			SendStatusSet []struct {
				Code string `json:"Code"`
			} `json:"SendStatusSet"`
		} `json:"Response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Response.Error.Code != "" {
		return fmt.Errorf("腾讯云短信发送失败: %s %s", result.Response.Error.Code, result.Response.Error.Message)
	}
	return nil
}

// sendSMSBao 短信宝
func sendSMSBao(phone, content string) error {
	user := GetSetting("smsbao_username", "")
	pass := GetSetting("smsbao_password", "")
	if user == "" {
		return fmt.Errorf("短信宝配置不完整")
	}

	api := fmt.Sprintf("http://api.smsbao.com/sms?u=%s&p=%s&m=%s&c=%s",
		url.QueryEscape(user), md5Hex(pass), phone, url.QueryEscape(content))

	resp, err := http.Get(api)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// 返回 0 表示成功
	if strings.TrimSpace(string(body)) != "0" {
		return fmt.Errorf("短信宝发送失败: %s", strings.TrimSpace(string(body)))
	}
	return nil
}

// postJSONForm POST 表单并解析 JSON 响应
func postJSONForm(rawURL string, form url.Values, target interface{}) error {
	resp, err := http.Post(rawURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("请求 %s 返回 %d: %s", rawURL, resp.StatusCode, truncateStr(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func truncateStr(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}
