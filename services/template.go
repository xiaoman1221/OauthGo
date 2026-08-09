package services

import "strings"

// RenderTemplate 渲染 {{key}} 占位符模板
func RenderTemplate(tpl string, data map[string]interface{}) string {
	for k, v := range data {
		val := ""
		if s, ok := v.(string); ok {
			val = s
		}
		tpl = strings.ReplaceAll(tpl, "{{"+k+"}}", val)
	}
	return tpl
}
