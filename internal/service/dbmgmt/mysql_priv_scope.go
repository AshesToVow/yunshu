package dbmgmt

import (
	"fmt"
	"strings"

	"yunshu/internal/model"
)

// MySQL 仅能在 *.* 上授予的权限；库级申请中须拆成单独 GRANT 语句。
// UI 中的 GRANT 表示 GRANT OPTION，不能出现在 GRANT priv_list 里。
var mysqlGlobalOnlyPrivs = map[string]struct{}{
	"SUPER":              {},
	"PROCESS":            {},
	"RELOAD":             {},
	"SHUTDOWN":           {},
	"SHOW DATABASES":     {},
	"REPLICATION CLIENT": {},
	"REPLICATION SLAVE":  {},
	"CREATE USER":        {},
}

type mysqlPrivSplit struct {
	Database []string
	Global   []string
	WithGrantOption bool
}

func splitMySQLPrivileges(privs []string, privLevel string) mysqlPrivSplit {
	var split mysqlPrivSplit
	isGlobal := strings.EqualFold(strings.TrimSpace(privLevel), model.DbAppUserPrivGlobal)
	for _, p := range privs {
		up := strings.ToUpper(strings.TrimSpace(p))
		if up == "" {
			continue
		}
		if up == "GRANT" {
			split.WithGrantOption = true
			continue
		}
		if isGlobal {
			split.Global = append(split.Global, up)
			continue
		}
		if _, globalOnly := mysqlGlobalOnlyPrivs[up]; globalOnly {
			split.Global = append(split.Global, up)
		} else {
			split.Database = append(split.Database, up)
		}
	}
	return split
}

func joinMySQLPrivs(privs []string, fallback string) string {
	if len(privs) == 0 {
		return fallback
	}
	return strings.Join(privs, ", ")
}

func buildGrantStmtsForHost(req *model.DbAppUserRequest, host string, privs []string) []string {
	split := splitMySQLPrivileges(privs, req.PrivLevel)
	user := escapeMySQLString(req.MySQLUser)
	hostEsc := escapeMySQLString(host)
	var stmts []string

	if strings.EqualFold(req.PrivLevel, model.DbAppUserPrivGlobal) {
		clause := joinMySQLPrivs(split.Global, "USAGE")
		if clause == "" {
			return stmts
		}
		stmt := fmtGrant(clause, "*.*", user, hostEsc, split.WithGrantOption)
		return []string{stmt}
	}

	dbObj := grantObject(req)
	if len(split.Database) > 0 {
		withGrant := split.WithGrantOption && len(split.Global) == 0
		stmts = append(stmts, fmtGrant(joinMySQLPrivs(split.Database, "SELECT"), dbObj, user, hostEsc, withGrant))
	} else if split.WithGrantOption && len(split.Global) == 0 {
		stmts = append(stmts, fmtGrant("USAGE", dbObj, user, hostEsc, true))
	}
	if len(split.Global) > 0 {
		stmts = append(stmts, fmtGrant(joinMySQLPrivs(split.Global, "USAGE"), "*.*", user, hostEsc, split.WithGrantOption))
	}
	return stmts
}

func buildRevokeStmtsForHost(req *model.DbAppUserRequest, host string, privs []string) []string {
	split := splitMySQLPrivileges(privs, req.PrivLevel)
	user := escapeMySQLString(req.MySQLUser)
	hostEsc := escapeMySQLString(host)
	var stmts []string

	if strings.EqualFold(req.PrivLevel, model.DbAppUserPrivGlobal) {
		clause := joinMySQLPrivs(split.Global, "")
		if clause == "" {
			return stmts
		}
		return []string{fmt.Sprintf("REVOKE %s ON *.* FROM '%s'@'%s'", clause, user, hostEsc)}
	}

	dbObj := grantObject(req)
	if len(split.Database) > 0 {
		stmts = append(stmts, fmt.Sprintf("REVOKE %s ON %s FROM '%s'@'%s'",
			joinMySQLPrivs(split.Database, "SELECT"), dbObj, user, hostEsc))
	}
	if len(split.Global) > 0 {
		stmts = append(stmts, fmt.Sprintf("REVOKE %s ON *.* FROM '%s'@'%s'",
			joinMySQLPrivs(split.Global, "USAGE"), user, hostEsc))
	}
	if split.WithGrantOption {
		stmts = append(stmts, fmt.Sprintf("REVOKE GRANT OPTION ON %s FROM '%s'@'%s'", dbObj, user, hostEsc))
	}
	return stmts
}

func fmtGrant(privClause, obj, user, host string, withGrantOption bool) string {
	stmt := fmt.Sprintf("GRANT %s ON %s TO '%s'@'%s'", privClause, obj, user, host)
	if withGrantOption {
		stmt += " WITH GRANT OPTION"
	}
	return stmt
}
