package cmdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/exportutil"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"

	"github.com/xuri/excelize/v2"
)

// ServerImportRowError 服务器导入单行错误。
type ServerImportRowError struct {
	Row     int    `json:"row"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Message string `json:"message"`
}

// ServerImportResult 服务器批量导入结果。
type ServerImportResult struct {
	Imported int                    `json:"imported"`
	Skipped  int                    `json:"skipped"`
	Errors   []ServerImportRowError `json:"errors,omitempty"`
}

// ImportServersFromExcel 从 Excel 批量导入服务器。
func (s *Service) ImportServersFromExcel(ctx context.Context, projectID uint, r io.Reader) (*ServerImportResult, error) {
	limited := exportutil.LimitedImportReader(r)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ImportServersFromExcel", err)
	}
	if err := exportutil.CheckImportReadSize(int64(len(data))); err != nil {
		return nil, err
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg04d13e805997)
	}
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ImportServersFromExcel", err)
	}
	result := &ServerImportResult{}
	if len(rows) <= 1 {
		return result, nil
	}
	if len(rows) > exportutil.MaxExcelImportRows+1 {
		return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf("导入行数超过 %d 行上限", exportutil.MaxExcelImportRows))
	}
	header := map[string]int{}
	for i, h := range rows[0] {
		header[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(row []string, key string) string {
		idx, ok := header[key]
		if !ok || idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}
	for i, row := range rows[1:] {
		rowNum := i + 2
		name := get(row, "name")
		host := get(row, "host")
		if name == "" && host == "" {
			continue
		}
		if name == "" || host == "" {
			result.Skipped++
			continue
		}
		portStr := get(row, "port")
		port, portErr := strconv.Atoi(portStr)
		if portStr != "" && portErr != nil {
			result.Errors = append(result.Errors, ServerImportRowError{Row: rowNum, Name: name, Host: host, Message: "port 格式无效"})
			continue
		}
		if port <= 0 {
			port = 22
		}
		upReq := ServerUpsertRequest{
			ProjectID: projectID,
			Name:      name,
			Host:      host,
			Port:      port,
			OSType:    get(row, "os_type"),
			Tags:      get(row, "tags"),
			Status:    model.StatusEnabled,
			AuthType:  get(row, "auth_type"),
			Username:  get(row, "username"),
		}
		if pw := get(row, "password"); pw != "" {
			upReq.Password = &pw
		}
		if pk := get(row, "private_key"); pk != "" {
			upReq.PrivateKey = &pk
		}
		if pp := get(row, "passphrase"); pp != "" {
			upReq.Passphrase = &pp
		}
		if _, upsertErr := s.UpsertServer(ctx, upReq); upsertErr != nil {
			result.Errors = append(result.Errors, ServerImportRowError{Row: rowNum, Name: name, Host: host, Message: upsertErr.Error()})
			continue
		}
		result.Imported++
	}
	return result, nil
}

// ExportServersToExcel 导出项目服务器列表为 Excel。
func (s *Service) ExportServersToExcel(ctx context.Context, projectID uint, keyword string) (*excelize.File, error) {
	list, _, err := s.serverRepo.List(ctx, repository.ServerListParams{ProjectID: projectID, Keyword: strings.TrimSpace(keyword), Page: 1, PageSize: 10000})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "cmdb", "ExportServersToExcel", err)
	}
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	headers := []string{"name", "host", "port", "os_type", "os_arch", "tags", "status", "last_test_at", "last_test_error"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	for r, sv := range list {
		values := []any{sv.Name, sv.Host, sv.Port, sv.OSType, sv.OSArch, sv.Tags, sv.Status, "", ""}
		if sv.LastTestAt != nil {
			values[7] = sv.LastTestAt.Format(time.RFC3339)
		}
		if sv.LastTestError != nil {
			values[8] = *sv.LastTestError
		}
		for c, v := range values {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			_ = f.SetCellValue(sheet, cell, v)
		}
	}
	return f, nil
}

// ServersImportTemplateExcel 生成服务器批量导入模板。
func (s *Service) ServersImportTemplateExcel() (*excelize.File, error) {
	f := excelize.NewFile()

	// Template sheet
	sheet := f.GetSheetName(0)
	f.SetSheetName(sheet, "servers")
	sheet = "servers"

	headers := []string{"name", "host", "port", "os_type", "tags", "auth_type", "username", "password", "private_key", "passphrase"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	// Example row (placeholders only)
	example := []any{
		"app-01",
		"10.0.0.12",
		22,
		"linux",
		"prod,web",
		"password",
		"root",
		"",
		"",
		"",
	}
	for c, v := range example {
		cell, _ := excelize.CoordinatesToCellName(c+1, 2)
		_ = f.SetCellValue(sheet, cell, v)
	}

	// Notes sheet
	noteSheet := "填写说明"
	_, _ = f.NewSheet(noteSheet)
	notes := [][]any{
		{"字段", "说明", "必填", "示例/备注"},
		{"name", "服务器名称", "是", "app-01"},
		{"host", "主机名/IP", "是", "10.0.0.12 / example.com"},
		{"port", "SSH 端口，默认 22", "否", "22"},
		{"os_type", "操作系统类型，默认 linux", "否", "linux / windows / darwin"},
		{"tags", "标签，逗号分隔", "否", "prod,web"},
		{"auth_type", "认证方式，默认 password", "否", "password / key"},
		{"username", "SSH 用户名", "是", "root"},
		{"password", "SSH 密码（不推荐在 Excel 中填写，建议导入后在平台录入）", "否", ""},
		{"private_key", "SSH 私钥（不推荐在 Excel 中填写）", "否", ""},
		{"passphrase", "私钥口令", "否", ""},
		{"提示", "导入后会写入当前项目；凭证请在平台「服务器详情」中安全配置。", "", ""},
	}
	for r, row := range notes {
		cell, _ := excelize.CoordinatesToCellName(1, r+1)
		_ = f.SetSheetRow(noteSheet, cell, &row)
	}

	// Basic styling
	_ = f.SetColWidth(sheet, "A", "A", 16)
	_ = f.SetColWidth(sheet, "B", "B", 18)
	_ = f.SetColWidth(sheet, "C", "C", 8)
	_ = f.SetColWidth(sheet, "D", "D", 10)
	_ = f.SetColWidth(sheet, "E", "E", 18)
	_ = f.SetColWidth(sheet, "F", "F", 12)
	_ = f.SetColWidth(sheet, "G", "G", 12)
	_ = f.SetColWidth(sheet, "H", "H", 16)
	_ = f.SetColWidth(sheet, "I", "J", 18)

	_ = f.SetColWidth(noteSheet, "A", "A", 16)
	_ = f.SetColWidth(noteSheet, "B", "B", 48)
	_ = f.SetColWidth(noteSheet, "C", "C", 10)
	_ = f.SetColWidth(noteSheet, "D", "D", 26)

	f.SetActiveSheet(0)
	return f, nil
}
