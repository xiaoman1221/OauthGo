package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// bingDailyURL Bing 每日一图查询接口
const bingDailyURL = "https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=1"

// BingDaily 登录页「Bing 每日一图」背景：代理 Bing 每日壁纸并 302 跳转到原图地址
func BingDaily(c *gin.Context) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(bingDailyURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "获取 Bing 每日一图失败"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "获取 Bing 每日一图失败"})
		return
	}

	var data struct {
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&data); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "获取 Bing 每日一图失败"})
		return
	}
	if len(data.Images) == 0 || data.Images[0].URL == "" {
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "获取 Bing 每日一图失败"})
		return
	}

	// Bing 返回相对路径（/th?id=...），拼接为完整地址
	imageURL := data.Images[0].URL
	if imageURL[0] == '/' {
		imageURL = fmt.Sprintf("https://www.bing.com%s", imageURL)
	}
	c.Redirect(http.StatusFound, imageURL)
}
