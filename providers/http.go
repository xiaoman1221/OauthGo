package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// getJSON 发起 GET 请求并解析 JSON 响应
func getJSON(rawURL string, target interface{}) error {
	return getJSONWithHeaders(rawURL, nil, target)
}

// getJSONWithHeaders 发起带请求头的 GET 请求并解析 JSON 响应
func getJSONWithHeaders(rawURL string, headers map[string]string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s 返回 %d: %s", rawURL, resp.StatusCode, truncate(string(body)))
	}
	return decodeJSON(resp.Body, target)
}

// postForm 发起表单 POST 请求并解析 JSON 响应
func postForm(rawURL string, form url.Values, target interface{}) error {
	resp, err := httpClient.PostForm(rawURL, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s 返回 %d: %s", rawURL, resp.StatusCode, truncate(string(body)))
	}
	return decodeJSON(resp.Body, target)
}

// postJSON 发起 JSON POST 请求并解析 JSON 响应
func postJSON(rawURL string, payload interface{}, target interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s 返回 %d: %s", rawURL, resp.StatusCode, truncate(string(body)))
	}
	return decodeJSON(resp.Body, target)
}

// getText 发起 GET 请求，接收文本响应（如 QQ 的 access_token 查询串）
func getText(rawURL string) (string, error) {
	resp, err := httpClient.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return readBody(resp.StatusCode, rawURL, resp.Body)
}

// postText 发起 POST 请求，接收文本响应（如 QQ 的 access_token 查询串）
func postText(rawURL string, form url.Values) (string, error) {
	resp, err := httpClient.PostForm(rawURL, form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return readBody(resp.StatusCode, rawURL, resp.Body)
}

func readBody(statusCode int, rawURL string, r io.Reader) (string, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if statusCode >= 400 {
		return "", fmt.Errorf("请求 %s 返回 %d: %s", rawURL, statusCode, truncate(string(body)))
	}
	return string(body), nil
}

func decodeJSON(r io.Reader, target interface{}) error {
	return json.NewDecoder(r).Decode(target)
}

func truncate(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}

// parseQueryString 解析形如 a=1&b=2 的响应文本
func parseQueryString(s string) url.Values {
	v, _ := url.ParseQuery(s)
	return v
}

// extractJSONCallback 解析 JSONP 回调，如 callback({"openid":"xxx"})
func extractJSONCallback(s string) (string, error) {
	start := strings.Index(s, "(")
	end := strings.LastIndex(s, ")")
	if start < 0 || end <= start {
		return "", fmt.Errorf("无效的 JSONP 响应: %s", truncate(s))
	}
	return strings.TrimSpace(s[start+1 : end]), nil
}
