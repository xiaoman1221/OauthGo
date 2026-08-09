package handlers

import (
	"net/http"

	"OauthGo/docs"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// serveDoc 输出内嵌的接口文档资源
func serveDoc(c *gin.Context, name, contentType string) {
	data, err := docs.FS.ReadFile(name)
	if err != nil {
		utils.FailInternal(c, "文档加载失败")
		return
	}
	c.Data(http.StatusOK, contentType, data)
}

// DocsIndex 接口文档首页（彩虹协议 + REST 接口说明）
func DocsIndex(c *gin.Context) {
	serveDoc(c, "index.html", "text/html; charset=utf-8")
}

// DocsSwagger Swagger UI 在线调试页
func DocsSwagger(c *gin.Context) {
	serveDoc(c, "swagger.html", "text/html; charset=utf-8")
}

// DocsOpenAPI OpenAPI 规范文件（yaml）
func DocsOpenAPI(c *gin.Context) {
	serveDoc(c, "openapi.yaml", "application/yaml; charset=utf-8")
}
