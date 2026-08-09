package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// httpClient 默认 HTTP 客户端（直连）
var httpClient = &http.Client{Timeout: 15 * time.Second}

// ProxyConfig SOCKS5 代理配置
type ProxyConfig struct {
	Address  string
	Username string
	Password string
}

// clientFor 根据是否启用代理返回对应的 HTTP 客户端
func clientFor(useProxy bool, cfg ProxyConfig) *http.Client {
	if !useProxy || strings.TrimSpace(cfg.Address) == "" {
		return httpClient
	}
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				d, err := socks5Dialer(cfg)
				if err != nil {
					return nil, err
				}
				return d.Dial(network, addr)
			},
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}
}

// socks5Dialer 创建 SOCKS5 拨号器
func socks5Dialer(cfg ProxyConfig) (proxy.Dialer, error) {
	var auth *proxy.Auth
	if cfg.Username != "" {
		auth = &proxy.Auth{User: cfg.Username, Password: cfg.Password}
	}
	return proxy.SOCKS5("tcp", cfg.Address, auth, proxy.Direct)
}

// getJSON 发起 GET 请求并解析 JSON 响应（直连）
func getJSON(rawURL string, target interface{}) error {
	return getJSONClient(httpClient, rawURL, nil, target)
}

// getJSONWithHeaders 发起带请求头的 GET 请求并解析 JSON 响应（直连）
func getJSONWithHeaders(rawURL string, headers map[string]string, target interface{}) error {
	return getJSONClient(httpClient, rawURL, headers, target)
}

// getJSONClient 使用指定客户端发起 GET 请求并解析 JSON 响应
func getJSONClient(client *http.Client, rawURL string, headers map[string]string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
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

// postForm 发起表单 POST 请求并解析 JSON 响应（直连）
func postForm(rawURL string, form url.Values, target interface{}) error {
	return postFormClient(httpClient, rawURL, nil, form, target)
}

// postFormClient 使用指定客户端发起表单 POST 请求并解析 JSON 响应
func postFormClient(client *http.Client, rawURL string, headers map[string]string, form url.Values, target interface{}) error {
	req, err := http.NewRequest(http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
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

// postJSON 发起 JSON POST 请求并解析 JSON 响应（直连）
func postJSON(rawURL string, payload interface{}, target interface{}) error {
	return postJSONClient(httpClient, rawURL, payload, target)
}

// postJSONClient 使用指定客户端发起 JSON POST 请求并解析 JSON 响应
func postJSONClient(client *http.Client, rawURL string, payload interface{}, target interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
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

// getText 发起 GET 请求，接收文本响应（直连）
func getText(rawURL string) (string, error) {
	return getTextClient(httpClient, rawURL)
}

// getTextClient 使用指定客户端发起 GET 请求，接收文本响应
func getTextClient(client *http.Client, rawURL string) (string, error) {
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return readBody(resp.StatusCode, rawURL, resp.Body)
}

// postText 发起 POST 请求，接收文本响应（直连）
func postText(rawURL string, form url.Values) (string, error) {
	return postTextClient(httpClient, rawURL, form)
}

// postTextClient 使用指定客户端发起 POST 请求，接收文本响应
func postTextClient(client *http.Client, rawURL string, form url.Values) (string, error) {
	resp, err := client.PostForm(rawURL, form)
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
