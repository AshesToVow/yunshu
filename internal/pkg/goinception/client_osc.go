package goinception

import (
	"context"
	"fmt"
	"strings"
)

// OSC 控制命令：get（进度）、kill、pause、resume。
func (c *Client) OSCControl(ctx context.Context, sqlsha1, command string) (*ReviewSet, error) {
	sqlsha1 = strings.TrimSpace(sqlsha1)
	command = strings.TrimSpace(strings.ToLower(command))
	if sqlsha1 == "" {
		return nil, fmt.Errorf("sqlsha1 不能为空")
	}
	var sql string
	switch command {
	case "get":
		sql = fmt.Sprintf("inception get osc_percent '%s';", escapeInceptionValue(sqlsha1))
	default:
		sql = fmt.Sprintf("inception %s osc '%s';", command, escapeInceptionValue(sqlsha1))
	}
	return c.run(ctx, sql)
}

// QueryAdmin 执行 goInception 管理命令（如 get variables）。
func (c *Client) QueryAdmin(ctx context.Context, sql string) (*ReviewSet, error) {
	return c.run(ctx, strings.TrimSpace(sql))
}

// ExtractOSCJobs 从执行结果中提取带 sqlsha1 的 OSC 任务。
func ExtractOSCJobs(rows []ReviewRow) []ReviewRow {
	var out []ReviewRow
	seen := map[string]struct{}{}
	for _, row := range rows {
		sha := strings.TrimSpace(row.SQLSHA1)
		if sha == "" {
			continue
		}
		if _, ok := seen[sha]; ok {
			continue
		}
		seen[sha] = struct{}{}
		out = append(out, row)
	}
	return out
}
