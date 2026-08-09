package docs

import "embed"

// FS 内嵌的接口文档资源
//
//go:embed index.html openapi.yaml swagger.html
var FS embed.FS
