package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestAppOIDCDiscoveryURL：应用详情响应应携带本平台 OIDC Discovery URL
func TestAppOIDCDiscoveryURL(t *testing.T) {
	user, token := testUser(t)
	app := seedOwnedApp(t, user.ID)

	code, m := doAuthedJSON(t, http.MethodGet, fmt.Sprintf("/api/apps/%d", app.ID), "", token)
	if code != 200 || int(m["code"].(float64)) != 0 {
		t.Fatalf("获取应用详情失败: %d %v", code, m)
	}
	data, _ := m["data"].(map[string]interface{})
	disc, _ := data["oidc_discovery_url"].(string)
	if disc == "" || !strings.HasSuffix(disc, "/.well-known/openid-configuration") {
		t.Fatalf("缺少 oidc_discovery_url: %v", data)
	}
}
