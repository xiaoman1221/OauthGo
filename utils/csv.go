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
			// 防 CSV 公式注入：以 = + - @ 或控制字符开头的单元格加前导单引号
			csvSafe(r.AppName),
			csvSafe(r.Username),
			csvSafe(r.Nickname),
			csvSafe(r.Platform),
			csvSafe(r.IP),
			csvSafe(r.Location),
			r.LoginTime.Format(loginTimeFormat),
			strconv.Itoa(r.Status),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

// csvSafe 对可能被电子表格当作公式的单元格加前导单引号
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '	', '':
		return "'" + s
	}
	return s
}
