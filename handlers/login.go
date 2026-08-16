package handlers

import (
	"os"
	"path/filepath"
	"strconv"

	"OauthGo/database"
	"OauthGo/models"
	"OauthGo/utils"

	"github.com/gin-gonic/gin"
)

// platformAccountLabel 根据登录方式返回账号标识名称
func platformAccountLabel(platform string) string {
	switch platform {
	case "qq":
		return "QQ号"
	case "wechat", "wechat_miniprogram":
		return "OpenID"
	case "weibo":
		return "微博UID"
	case "gitee":
		return "Gitee UID"
	case "douyin":
		return "抖音OpenID"
	case "baidu":
		return "百度UID"
	case "alipay":
		return "支付宝用户ID"
	case "dingtalk":
		return "钉钉UserID"
	case "wecom":
		return "企业微信UserID"
	case "lark":
		return "飞书OpenID"
	case "infoflow":
		return "如流UID"
	default:
		return "OpenID"
	}
}

// loginAccountLabelValue 返回登录记录账号的展示标签与值
// 有用户名（如 CSV 导入）时优先显示用户名，否则按登录方式显示对应的标识
func loginAccountLabelValue(r models.LoginRecord) (string, string) {
	if r.Username != "" {
		return "用户名", r.Username
	}
	return platformAccountLabel(r.Platform), r.OpenID
}

// loginRecordItem 登录记录列表项（附带按登录方式展示的用户名字段）
type loginRecordItem struct {
	models.LoginRecord
	UIDLabel string `json:"uid_label"`
	UIDValue string `json:"uid_value"`
}

// ListLoginRecords 登录记录分页查询
func ListLoginRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 基础查询
	query := database.DB.Model(&models.LoginRecord{})
	// 非管理员限制：仅显示当前用户所拥有应用产生的登录记录
	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	if role != "admin" {
		uid, _ := userAny.(uint)
		var appIDs []uint
		database.DB.Model(&models.App{}).Where("owner_id = ?", uid).Pluck("id", &appIDs)
		if len(appIDs) == 0 {
			utils.Success(c, gin.H{"list": []loginRecordItem{}, "total": 0})
			return
		}
		query = query.Where("app_id IN ?", appIDs)
	}

	if appID := c.Query("app_id"); appID != "" {
		// 若非管理员且指定 app_id，不属于用户的 app 则返回空
		if role != "admin" {
			uid, _ := userAny.(uint)
			var cnt int64
			database.DB.Model(&models.App{}).Where("id = ? AND owner_id = ?", appID, uid).Count(&cnt)
			if cnt == 0 {
				utils.Success(c, gin.H{"list": []loginRecordItem{}, "total": 0})
				return
			}
		}
		query = query.Where("app_id = ?", appID)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ? OR ip LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var records []models.LoginRecord
	query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records)

	items := make([]loginRecordItem, 0, len(records))
	for _, r := range records {
		label, value := loginAccountLabelValue(r)
		items = append(items, loginRecordItem{LoginRecord: r, UIDLabel: label, UIDValue: value})
	}

	utils.Success(c, gin.H{"list": items, "total": total})
}

// DeleteLoginRecord 删除单条登录记录
func DeleteLoginRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}

	var rec models.LoginRecord
	if err := database.DB.First(&rec, id).Error; err != nil {
		utils.FailNotFound(c, "记录不存在")
		return
	}

	// 权限校验：非管理员仅能删除属于自己应用的记录
	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	if role != "admin" {
		uid, _ := userAny.(uint)
		if rec.AppID == 0 {
			utils.FailForbidden(c)
			return
		}
		var cnt int64
		database.DB.Model(&models.App{}).Where("id = ? AND owner_id = ?", rec.AppID, uid).Count(&cnt)
		if cnt == 0 {
			utils.FailForbidden(c)
			return
		}
	}

	if err := database.DB.Delete(&models.LoginRecord{}, id).Error; err != nil {
		utils.FailInternal(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "删除成功")
}

// BatchDeleteLoginRecords 批量删除登录记录
func BatchDeleteLoginRecords(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.FailBadRequest(c, "参数错误")
		return
	}
	if len(req.IDs) == 0 {
		utils.FailBadRequest(c, "请选择要删除的记录")
		return
	}

	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	if role != "admin" {
		uid, _ := userAny.(uint)
		var appIDs []uint
		database.DB.Model(&models.App{}).Where("owner_id = ?", uid).Pluck("id", &appIDs)
		if len(appIDs) == 0 {
			utils.FailForbidden(c)
			return
		}
		var cnt int64
		database.DB.Model(&models.LoginRecord{}).Where("id IN ? AND app_id IN ?", req.IDs, appIDs).Count(&cnt)
		if cnt != int64(len(req.IDs)) {
			utils.FailForbidden(c)
			return
		}
	}

	if err := database.DB.Delete(&models.LoginRecord{}, req.IDs).Error; err != nil {
		utils.FailInternal(c, "删除失败")
		return
	}
	utils.SuccessMsg(c, "批量删除成功")
}

// serveEmptyLoginCSV 导出空 CSV（无权限/无数据时使用）
func serveEmptyLoginCSV(c *gin.Context) {
	tmpPath := filepath.Join(os.TempDir(), "login_records.csv")
	_ = utils.ExportLoginRecordsToCSV(tmpPath, nil)
	defer os.Remove(tmpPath)
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=login_records.csv")
	c.File(tmpPath)
}

// ExportLoginRecords CSV 导出登录记录
func ExportLoginRecords(c *gin.Context) {
	query := database.DB.Model(&models.LoginRecord{})
	// 非管理员限制：仅导出当前用户所拥有应用产生的登录记录
	roleAny, _ := c.Get("role")
	userAny, _ := c.Get("user_id")
	role, _ := roleAny.(string)
	if role != "admin" {
		uid, _ := userAny.(uint)
		var appIDs []uint
		database.DB.Model(&models.App{}).Where("owner_id = ?", uid).Pluck("id", &appIDs)
		if len(appIDs) == 0 {
			serveEmptyLoginCSV(c)
			return
		}
		query = query.Where("app_id IN ?", appIDs)
	}

	if appID := c.Query("app_id"); appID != "" {
		if role != "admin" {
			uid, _ := userAny.(uint)
			var cnt int64
			database.DB.Model(&models.App{}).Where("id = ? AND owner_id = ?", appID, uid).Count(&cnt)
			if cnt == 0 {
				serveEmptyLoginCSV(c)
				return
			}
		}
		query = query.Where("app_id = ?", appID)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ? OR ip LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var records []models.LoginRecord
	query.Order("id desc").Find(&records)

	tmpPath := filepath.Join(os.TempDir(), "login_records.csv")
	if err := utils.ExportLoginRecordsToCSV(tmpPath, records); err != nil {
		utils.FailInternal(c, "导出失败")
		return
	}
	defer os.Remove(tmpPath)

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=login_records.csv")
	c.File(tmpPath)
}
