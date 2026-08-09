package utils

import "github.com/gin-gonic/gin"

// 统一响应格式
func Success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": data})
}

func SuccessMsg(c *gin.Context, message string) {
	c.JSON(200, gin.H{"code": 0, "message": message})
}

func Fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"code": status, "message": message})
}

func FailBadRequest(c *gin.Context, message string) {
	Fail(c, 400, message)
}

func FailUnauthorized(c *gin.Context) {
	Fail(c, 401, "未授权，请先登录")
}

func FailForbidden(c *gin.Context) {
	Fail(c, 403, "无权限访问")
}

func FailNotFound(c *gin.Context, message string) {
	Fail(c, 404, message)
}

func FailInternal(c *gin.Context, message string) {
	Fail(c, 500, message)
}
