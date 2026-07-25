package inspect

import (
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
)

func renderExcel(data ReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	summary := "概览"
	_ = f.SetSheetName("Sheet1", summary)
	_ = f.SetCellValue(summary, "A1", "项目")
	_ = f.SetCellValue(summary, "B1", data.Project)
	_ = f.SetCellValue(summary, "A2", "时间")
	_ = f.SetCellValue(summary, "B2", data.Timestamp.Format("2006-01-02 15:04:05"))
	_ = f.SetCellValue(summary, "A3", "数据源")
	_ = f.SetCellValue(summary, "B3", data.Datasource)
	_ = f.SetCellValue(summary, "A4", "执行人")
	_ = f.SetCellValue(summary, "B4", data.InspectionUser)
	_ = f.SetCellValue(summary, "A5", "健康分")
	_ = f.SetCellValue(summary, "B5", data.Score)
	_ = f.SetCellValue(summary, "A6", "等级")
	_ = f.SetCellValue(summary, "B6", data.Grade)
	_ = f.SetCellValue(summary, "A7", "摘要")
	_ = f.SetCellValue(summary, "B7", data.Summary)
	_ = f.SetCellValue(summary, "A8", "总计/严重/警告/正常")
	_ = f.SetCellValue(summary, "B8", fmt.Sprintf("%d / %d / %d / %d", data.Total, data.Critical, data.Warning, data.Normal))

	findings := "发现"
	_, _ = f.NewSheet(findings)
	_ = f.SetCellValue(findings, "A1", "严重级别")
	_ = f.SetCellValue(findings, "B1", "分类")
	_ = f.SetCellValue(findings, "C1", "名称")
	_ = f.SetCellValue(findings, "D1", "次数")
	_ = f.SetCellValue(findings, "E1", "建议")
	for i, it := range data.Findings {
		row := i + 2
		_ = f.SetCellValue(findings, "A"+strconv.Itoa(row), it.Severity)
		_ = f.SetCellValue(findings, "B"+strconv.Itoa(row), it.Type)
		_ = f.SetCellValue(findings, "C"+strconv.Itoa(row), it.Name)
		_ = f.SetCellValue(findings, "D"+strconv.Itoa(row), it.Count)
		_ = f.SetCellValue(findings, "E"+strconv.Itoa(row), it.Hint)
	}

	detail := "明细"
	_, _ = f.NewSheet(detail)
	_ = f.SetCellValue(detail, "A1", "分类")
	_ = f.SetCellValue(detail, "B1", "名称")
	_ = f.SetCellValue(detail, "C1", "实例")
	_ = f.SetCellValue(detail, "D1", "当前值")
	_ = f.SetCellValue(detail, "E1", "阈值")
	_ = f.SetCellValue(detail, "F1", "单位")
	_ = f.SetCellValue(detail, "G1", "状态")
	_ = f.SetCellValue(detail, "H1", "错误")
	row := 2
	for _, g := range data.Groups {
		for _, m := range g.Metrics {
			_ = f.SetCellValue(detail, "A"+strconv.Itoa(row), g.Type)
			_ = f.SetCellValue(detail, "B"+strconv.Itoa(row), m.Name)
			_ = f.SetCellValue(detail, "C"+strconv.Itoa(row), m.Instance)
			_ = f.SetCellValue(detail, "D"+strconv.Itoa(row), m.Value)
			_ = f.SetCellValue(detail, "E"+strconv.Itoa(row), m.Threshold)
			_ = f.SetCellValue(detail, "F"+strconv.Itoa(row), m.Unit)
			_ = f.SetCellValue(detail, "G"+strconv.Itoa(row), m.Status)
			_ = f.SetCellValue(detail, "H"+strconv.Itoa(row), m.Error)
			row++
		}
	}

	f.SetActiveSheet(0)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
