package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"strings"
)

// RandomString 生成指定长度的随机十六进制字符串（长度为 n）
func RandomString(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

// MD5 计算字符串 MD5 值（小写十六进制）
func MD5(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ExtractDomain 从字符串中提取并规整域名（小写、去除协议/路径/端口），非法时返回空串
func ExtractDomain(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return ""
	}
	// 校验：各标签非空且仅含字母/数字/连字符
	labels := strings.Split(host, ".")
	if len(labels) < 2 && host != "localhost" {
		return ""
	}
	for _, label := range labels {
		if label == "" {
			return ""
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return ""
			}
		}
	}
	return host
}
