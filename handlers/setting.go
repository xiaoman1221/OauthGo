package handlers

import (
	"OauthGo/models"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// ListSettings 获取全部系统设置（按分组返回，含默认值与敏感标记）
func ListSettings(c *gin.Context) {
	current := services.AllSettings()
	defs := models.DefaultSettingDefs()
	byGroup := map[string][]gin.H{}

	for _, def := range defs {
		value := current[def.Key]
		if def.Sensitive && value != "" {
			// 敏感字段回传占位符，避免泄露真实值
			value = "********"
		}
		item := gin.H{
			"key":         def.Key,
			"value":       value,
			"description": def.Description,
			"group":       def.Group,
			"sensitive":   def.Sensitive,
		}
		byGroup[def.Group] = append(byGroup[def.Group], item)
	}
	utils.Success(c, gin.H{"groups": byGroup})
}

// UpdateSettings 批量更新系统设置（key-value）
func UpdateSettings(c *gin.Context) {
	var req struct {
		Items []models.Setting `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	for _, item := range req.Items {
		if item.Key == "" {
			continue
		}
		value := item.Value
		// 前端未修改的敏感字段以 ******** 回传，跳过更新
		if value == "********" {
			continue
		}
		services.SetSetting(item.Key, value)
	}
	utils.SuccessMsg(c, "保存成功")
}
