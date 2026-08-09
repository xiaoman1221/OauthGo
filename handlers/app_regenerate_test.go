package handlers_test

import (
	"fmt"
	"testing"
)

func TestUpdateAppRegenerateKey(t *testing.T) {
	app := seedApp(t)
	_, token := testUser(t)

	_, m := doAuthedJSON(t, "PUT", fmt.Sprintf("/api/apps/%d", app.ID),
		`{"name":"测试站点","platform":"web","mode":"compat","types":["gitee"],"domains":"target.example.com","regenerate_key":true,"status":1}`,
		token)
	data, _ := m["data"].(map[string]interface{})

	key, _ := data["app_key"].(string)
	if key == "" {
		t.Fatalf("响应缺少 app_key: %v", m)
	}
	if key == app.AppKey {
		t.Fatalf("regenerate_key 未生效，app_key 未变化: %s", key)
	}
}

func TestUpdateAppKeepKeyWithoutFlag(t *testing.T) {
	app := seedApp(t)
	_, token := testUser(t)

	_, m := doAuthedJSON(t, "PUT", fmt.Sprintf("/api/apps/%d", app.ID),
		`{"name":"新名称","platform":"web","mode":"compat","types":["gitee"],"domains":"target.example.com","status":1}`,
		token)
	data, _ := m["data"].(map[string]interface{})

	if data["name"] != "新名称" {
		t.Fatalf("名称未更新: %v", m)
	}
	key, _ := data["app_key"].(string)
	if key != app.AppKey {
		t.Fatalf("未带 regenerate_key 时不应变更 app_key: %s", key)
	}
}
