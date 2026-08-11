package handlers

import (
	"strconv"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/services"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// ListChannels 通知渠道列表
func ListChannels(c *gin.Context) {
	// 非管理员仅列出已启用的渠道，同时对 config 字段脱敏
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)

	var channels []models.NotificationChannel
	q := database.DB.Order("id desc")
	if role != "admin" {
		q = q.Where("enabled = ?", true)
	}
	if err := q.Find(&channels).Error; err != nil {
		utils.FailInternal(c, "查询失败")
		return
	}

	result := make([]gin.H, 0, len(channels))
	for _, ch := range channels {
		item := gin.H{
			"id":         ch.ID,
			"name":       ch.Name,
			"type":       ch.Type,
			"enabled":    ch.Enabled,
			"created_at": ch.CreatedAt,
		}
		if role == "admin" {
			item["config"] = ch.Config
		} else {
			// 非管理员不返回凭据/配置
			item["config"] = ""
		}
		result = append(result, item)
	}

	utils.Success(c, gin.H{"list": result})
}

// CreateChannel 创建通知渠道（管理员）
func CreateChannel(c *gin.Context) {
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	if role != "admin" {
		utils.FailForbidden(c)
		return
	}

	var req models.NotificationChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误："+err.Error())
		return
	}
	if err := database.DB.Create(&req).Error; err != nil {
		utils.FailInternal(c, "创建渠道失败")
		return
	}
	utils.Success(c, req)
}

// UpdateChannel 更新通知渠道（管理员）
func UpdateChannel(c *gin.Context) {
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	if role != "admin" {
		utils.FailForbidden(c)
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var req models.NotificationChannel
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var channel models.NotificationChannel
	if err := database.DB.First(&channel, id).Error; err != nil {
		utils.FailNotFound(c, "渠道不存在")
		return
	}

	if err := database.DB.Model(&channel).Updates(map[string]interface{}{
		"name":    req.Name,
		"type":    req.Type,
		"config":  req.Config,
		"enabled": req.Enabled,
	}).Error; err != nil {
		utils.FailInternal(c, "更新渠道失败")
		return
	}
	utils.Success(c, channel)
}

// DeleteChannel 删除通知渠道（管理员）
func DeleteChannel(c *gin.Context) {
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	if role != "admin" {
		utils.FailForbidden(c)
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}
	if err := database.DB.Delete(&models.NotificationChannel{}, id).Error; err != nil {
		utils.FailInternal(c, "删除渠道失败")
		return
	}
	utils.SuccessMsg(c, "删除成功")
}

// TestChannel 测试发送通知（管理员）
func TestChannel(c *gin.Context) {
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	if role != "admin" {
		utils.FailForbidden(c)
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var channel models.NotificationChannel
	if err := database.DB.First(&channel, id).Error; err != nil {
		utils.FailNotFound(c, "渠道不存在")
		return
	}

	log := models.NotificationLog{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
		Subject:     "OauthGo 测试通知",
		Content:     "这是一条测试通知，发送成功！",
	}

	sendErr := services.SendByChannel(channel, log.Subject, log.Content)
	if sendErr != nil {
		log.Status = 2
		log.Error = sendErr.Error()
		database.DB.Create(&log)
		utils.Fail(c, 500, "发送失败："+sendErr.Error())
		return
	}

	log.Status = 1
	database.DB.Create(&log)
	utils.SuccessMsg(c, "发送成功")
}

// ListNotificationLogs 通知日志列表（管理员）
func ListNotificationLogs(c *gin.Context) {
	roleAny, _ := c.Get("role")
	role, _ := roleAny.(string)
	if role != "admin" {
		utils.FailForbidden(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := database.DB.Model(&models.NotificationLog{})
	if channelID := c.Query("channel_id"); channelID != "" {
		query = query.Where("channel_id = ?", channelID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var logs []models.NotificationLog
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs)

	utils.Success(c, gin.H{"list": logs, "total": total})
}
