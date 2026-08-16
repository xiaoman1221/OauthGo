package utils

import (
	"encoding/csv"
	"os"
	"strconv"

	"OauthGo/models"
)

const loginTimeFormat = "2006-01-02 15:04:05"

// ExportLoginRecordsToCSV 将登录记录导出为 CSV 文件
func ExportLoginRecordsToCSV(path string, records []models.LoginRecord) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"ID", "AppName", "Username", "Nickname", "Platform", "IP", "Location", "LoginTime", "Status"}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range records {
		row := []string{
			strconv.FormatUint(uint64(r.ID), 10),
			r.AppName,
			r.Username,
			r.Nickname,
			r.Platform,
			r.IP,
			r.Location,
			r.LoginTime.Format(loginTimeFormat),
			strconv.Itoa(r.Status),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}
