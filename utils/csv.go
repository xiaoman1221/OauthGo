package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

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

// ImportLoginRecordsFromCSV 从 CSV 文件导入登录记录
func ImportLoginRecordsFromCSV(path string) ([]models.LoginRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) <= 1 {
		return nil, nil
	}

	records := make([]models.LoginRecord, 0, len(rows)-1)
	for i, row := range rows[1:] {
		if len(row) < 8 {
			return nil, fmt.Errorf("第 %d 行数据格式错误", i+2)
		}
		status, _ := strconv.Atoi(row[8])
		loginTime, _ := time.ParseInLocation(loginTimeFormat, row[7], time.Local)
		records = append(records, models.LoginRecord{
			AppName:   row[1],
			Username:  row[2],
			Nickname:  row[3],
			Platform:  row[4],
			IP:        row[5],
			Location:  row[6],
			LoginTime: loginTime,
			Status:    status,
		})
	}
	return records, nil
}
