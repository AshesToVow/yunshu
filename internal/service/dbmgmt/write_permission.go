package dbmgmt

import (
	"context"
	"regexp"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

var (
	reInsertInto  = regexp.MustCompile("(?i)\\bINSERT\\s+(?:IGNORE\\s+)?INTO\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reReplaceInto = regexp.MustCompile("(?i)\\bREPLACE\\s+INTO\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reUpdateTable = regexp.MustCompile("(?i)\\bUPDATE\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reDeleteFrom  = regexp.MustCompile("(?i)\\bDELETE\\s+FROM\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reDDLTable    = regexp.MustCompile("(?i)\\b(?:ALTER|DROP|TRUNCATE)\\s+TABLE\\s+(?:IF\\s+EXISTS\\s+)?`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reCreateTable = regexp.MustCompile("(?i)\\bCREATE\\s+TABLE\\s+(?:IF\\s+NOT\\s+EXISTS\\s+)?`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reCreateIndex = regexp.MustCompile("(?i)\\b(?:CREATE|DROP)\\s+(?:UNIQUE\\s+)?INDEX\\s+\\S+\\s+ON\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reInstanceDDL = regexp.MustCompile(`(?i)^\s*(CREATE|DROP)\s+DATABASE\b`)
)

// extractWriteTableRefs 从变更 SQL 中提取目标表；解析不到时返回空，由调用方要求整库写授权。
func extractWriteTableRefs(sqlText, defaultDB string) []queryTableRef {
	sqlText = stripSQLComments(sqlText)
	if reInstanceDDL.MatchString(strings.TrimSpace(sqlText)) {
		return nil
	}
	seen := map[string]struct{}{}
	var refs []queryTableRef
	add := func(schema, table string) {
		table = strings.TrimSpace(table)
		if table == "" {
			return
		}
		// 避免把 DATABASE / INDEX 等关键字当成表名。
		switch strings.ToUpper(table) {
		case "DATABASE", "SCHEMA", "INDEX", "UNIQUE", "TEMPORARY", "VIEW", "IF":
			return
		}
		schema = strings.TrimSpace(schema)
		if schema == "" {
			schema = defaultDB
		}
		key := strings.ToLower(schema) + "." + strings.ToLower(table)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		refs = append(refs, queryTableRef{Schema: schema, Table: table})
	}
	addFromMatch := func(m []string) {
		if len(m) >= 3 && m[2] != "" {
			add(m[1], m[2])
		} else if len(m) >= 2 {
			add(defaultDB, m[1])
		}
	}
	for _, re := range []*regexp.Regexp{
		reInsertInto, reReplaceInto, reUpdateTable, reDeleteFrom, reDDLTable, reCreateTable, reCreateIndex, reJoinTable,
	} {
		for _, m := range re.FindAllStringSubmatch(sqlText, -1) {
			addFromMatch(m)
		}
	}
	return refs
}

func grantCoversWrite(g model.DbAccessGrant, db, table string, needDDL bool) bool {
	if needDDL {
		if !g.CanDDL {
			return false
		}
	} else if !g.CanDML {
		return false
	}
	gdb := strings.TrimSpace(g.DatabaseName)
	// 写操作禁止依赖空库名「整实例」授权，避免误授后 blast radius 过大；CanManage 已在上层短路。
	if gdb == "" {
		return false
	}
	tables := parseTableNamesJSON(g.TableNamesJSON)
	if !strings.EqualFold(gdb, db) {
		return false
	}
	if tablesAllowAll(tables) {
		return true
	}
	if table == "" {
		return false
	}
	for _, t := range tables {
		if strings.EqualFold(t, table) {
			return true
		}
	}
	return false
}

// resolveWriteAccess 校验写操作是否覆盖 SQL 涉及的全部表；无法解析表时仅整库/实例级写授权可放行。
func (s *Service) resolveWriteAccess(
	ctx context.Context,
	projectID uint,
	inst *model.DbInstance,
	database, sqlText string,
	needDDL bool,
	actor *auth.CurrentUser,
) error {
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return nil
	}
	if inst.OwnerUserID != nil && *inst.OwnerUserID == actorUserID(actor) {
		return nil
	}
	perm, err := s.effectivePermissionForDatabase(ctx, projectID, inst.ID, database, actor)
	if err != nil {
		return err
	}
	if perm.CanManage {
		return nil
	}

	refs := extractWriteTableRefs(sqlText, database)
	now := time.Now()
	coverAny := func(db, table string) (bool, error) {
		for _, k := range principalRefs(actor) {
			grants, gerr := s.repo.ListGrantsForPrincipal(ctx, projectID, inst.ID, k.kind, k.ref)
			if gerr != nil {
				return false, gerr
			}
			for _, g := range grants {
				if !grantStillValid(g, now) || !grantMatchesDatabase(g, db) {
					continue
				}
				if grantCoversWrite(g, db, table, needDDL) {
					return true, nil
				}
			}
		}
		return false, nil
	}

	if len(refs) == 0 {
		ok, err := coverAny(database, "")
		if err != nil {
			return err
		}
		if !ok {
			return constants.ErrForbiddenWithMsg("无法解析 SQL 涉及的表，且你没有该库的整库变更权限")
		}
		return nil
	}
	for _, ref := range refs {
		db := ref.Schema
		if db == "" {
			db = database
		}
		ok, err := coverAny(db, ref.Table)
		if err != nil {
			return err
		}
		if !ok {
			return constants.ErrForbiddenWithMsg("你无 " + db + "." + ref.Table + " 表的变更权限")
		}
	}
	return nil
}
