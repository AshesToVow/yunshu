package dbmgmt

import (
	"fmt"
	"strings"

	"yunshu/internal/model"
)

func formatMySQLGrantExecError(err error, inst *model.DbInstance, req *model.DbAppUserRequest, stmtIndex int, stmt string) string {
	if err == nil {
		return ""
	}
	raw := err.Error()
	adminUser := strings.TrimSpace(inst.Username)
	if adminUser == "" {
		adminUser = "实例管理员"
	}
	targetDB := strings.TrimSpace(req.DatabaseName)
	grantObj := grantObject(req)
	prefix := fmt.Sprintf("MySQL 执行失败（第 %d 条）", stmtIndex)

	switch {
	case strings.Contains(raw, "1044"):
		clientUser, clientHost := parseMySQLErrorUserHost(raw)
		if clientUser == "" {
			clientUser = adminUser
		}
		if clientHost == "" {
			clientHost = "%"
		}
		dbName := pickDBName(targetDB, raw)
		hint := fmt.Sprintf(
			"%s：实际连接身份为 '%s'@'%s'，对库 %q 无 GRANT 权限。",
			prefix, clientUser, clientHost, dbName,
		)
		if clientHost != "%" {
			hint += fmt.Sprintf(" 注意：授予 root@'%%' 不会覆盖已存在的 root@'%s'，需对该 host 单独授权。", clientHost)
		}
		hint += fmt.Sprintf(
			" 请在 MySQL 执行：GRANT ALL PRIVILEGES ON %s TO '%s'@'%s' WITH GRANT OPTION; FLUSH PRIVILEGES;",
			grantObj, clientUser, clientHost,
		)
		return hint
	case strings.Contains(raw, "1045"):
		return fmt.Sprintf("%s：实例连接账号 %q 认证失败，请检查数据库实例中配置的用户名/密码", prefix, adminUser)
	case strings.Contains(raw, "1141"):
		return fmt.Sprintf("%s：实例连接账号 %q 缺少 GRANT OPTION，无法为其他用户授权", prefix, adminUser)
	case strings.Contains(raw, "1142"):
		return fmt.Sprintf("%s：实例连接账号 %q 无法授予请求中的权限类型，请提升管理员账号权限或调整申请的权限范围", prefix, adminUser)
	case strings.Contains(raw, "1396"):
		return fmt.Sprintf("%s：目标用户 %q@主机不存在，请确认用户与授权主机是否正确", prefix, req.MySQLUser)
	default:
		return fmt.Sprintf("%s：%s\nSQL: %s", prefix, raw, strings.TrimSpace(stmt))
	}
}

func pickDBName(fromReq, errMsg string) string {
	if fromReq != "" {
		return fromReq
	}
	// Error 1044 (...): Access denied for user 'root'@'host' to database 'test'
	if i := strings.LastIndex(errMsg, "to database '"); i >= 0 {
		rest := errMsg[i+len("to database '"):]
		if j := strings.Index(rest, "'"); j > 0 {
			return rest[:j]
		}
	}
	return "目标库"
}

// parseMySQLErrorUserHost 从 MySQL 错误信息解析 user@host，如 'root'@'10.10.10.1'。
func parseMySQLErrorUserHost(errMsg string) (user, host string) {
	const marker = "for user '"
	i := strings.Index(errMsg, marker)
	if i < 0 {
		return "", ""
	}
	rest := errMsg[i+len(marker):]
	at := strings.Index(rest, "'@'")
	if at <= 0 {
		return "", ""
	}
	user = rest[:at]
	hostPart := rest[at+len("'@'"):]
	if j := strings.Index(hostPart, "'"); j >= 0 {
		host = hostPart[:j]
	}
	return user, host
}
