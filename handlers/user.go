package handlers

import (
	"strconv"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// ListUsers 用户列表（管理员）
func ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	var users []models.User
	database.DB.Model(&models.User{}).Count(&total)
	database.DB.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users)

	utils.Success(c, gin.H{"list": users, "total": total})
}

// CreateUserRequest 管理员创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

// CreateUser 创建用户（管理员）
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误："+err.Error())
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	if req.Role != "admin" && req.Role != "user" {
		utils.FailBadRequest(c, "角色不合法（admin / user）")
		return
	}

	var count int64
	database.DB.Model(&models.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		utils.FailBadRequest(c, "用户名已存在")
		return
	}
	if req.Email != "" {
		database.DB.Model(&models.User{}).Where("email = ?", req.Email).Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "邮箱已被注册")
			return
		}
	}
	if req.Phone != "" {
		database.DB.Model(&models.User{}).Where("phone = ?", req.Phone).Count(&count)
		if count > 0 {
			utils.FailBadRequest(c, "手机号已被注册")
			return
		}
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.FailInternal(c, "密码加密失败")
		return
	}

	user := models.User{
		Username:    req.Username,
		Nickname:    req.Username,
		Password:    hash,
		PasswordSet: true,
		Email:       req.Email,
		Phone:       req.Phone,
		Role:        req.Role,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		utils.FailInternal(c, "创建用户失败")
		return
	}
	utils.Success(c, user)
}

// UpdateUser 更新用户信息（管理员）
func UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		utils.FailNotFound(c, "用户不存在")
		return
	}

	updates := map[string]interface{}{}
	if req.Username != "" {
		var cnt int64
		database.DB.Model(&models.User{}).Where("username = ? AND id != ?", req.Username, user.ID).Count(&cnt)
		if cnt > 0 {
			utils.FailBadRequest(c, "用户名已被使用")
			return
		}
		updates["username"] = req.Username
	}
	if req.Email != "" {
		var cnt int64
		database.DB.Model(&models.User{}).Where("email = ? AND id != ?", req.Email, user.ID).Count(&cnt)
		if cnt > 0 {
			utils.FailBadRequest(c, "邮箱已被其他账号使用")
			return
		}
		updates["email"] = req.Email
	}
	if req.Role != "" {
		if req.Role != "admin" && req.Role != "user" {
			utils.FailBadRequest(c, "角色不合法（admin / user）")
			return
		}
		updates["role"] = req.Role
	}
	if req.Password != "" {
		hash, err := utils.HashPassword(req.Password)
		if err != nil {
			utils.FailInternal(c, "密码加密失败")
			return
		}
		updates["password"] = hash
	}
	if len(updates) == 0 {
		utils.FailBadRequest(c, "没有需要更新的字段")
		return
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		utils.FailInternal(c, "更新用户失败")
		return
	}
	utils.Success(c, user)
}

// DeleteUser 删除用户（管理员）
func DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	if id == 1 {
		utils.FailBadRequest(c, "不能删除默认管理员")
		return
	}
	if err := database.DB.Delete(&models.User{}, id).Error; err != nil {
		utils.FailInternal(c, "删除用户失败")
		return
	}
	utils.SuccessMsg(c, "删除成功")
}
