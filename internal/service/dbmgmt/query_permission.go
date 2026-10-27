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
	reExplainPrefix = regexp.MustCompile(`(?is)^\s*EXPLAIN\s+(?:(?:ANALYZE|EXTENDED|PARTITIONS)\s+|FORMAT\s*=\s*[A-Za-z0-9_]+\s+)*`)
	reStatementHead = regexp.MustCompile(`(?i)^(SELECT|INSERT|UPDATE|DELETE|REPLACE|TABLE|VALUES)\b`)
	reShowCreateObj = regexp.MustCompile("(?is)^\\s*SHOW\\s+CREATE\\s+(?:TEMPORARY\\s+)?(?:TABLE|VIEW)\\s+(.+?)\\s*;?\\s*$")
	reShowCreateDB  = regexp.MustCompile(`(?is)^\s*SHOW\s+CREATE\s+(?:DATABASE|SCHEMA)\s+(.+?)\s*;?\s*$`)
	reQualifiedName = regexp.MustCompile("^`?([A-Za-z0-9_$]+)`?(?:\\.`?([A-Za-z0-9_$]+)`?)?$")
	reFromTable     = regexp.MustCompile("(?i)\\bFROM\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reJoinTable     = regexp.MustCompile("(?i)\\bJOIN\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
)

type queryTableRef struct {
	Schema string
	Table  string
}

// parseQualifiedName 解析 `schema`.`table` / `table` 形式的对象名。
func parseQualifiedName(raw, defaultDB string) (queryTableRef, bool) {
	raw = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(raw), ";"))
	m := reQualifiedName.FindStringSubmatch(raw)
	if m == nil {
		return queryTableRef{}, false
	}
	if m[2] != "" {
		return queryTableRef{Schema: m[1], Table: m[2]}, true
	}
	return queryTableRef{Schema: defaultDB, Table: m[1]}, true
}

func extractQueryTableRefs(sqlText, defaultDB string) []queryTableRef {
	sqlText = stripSQLComments(sqlText)
	trimmed := strings.TrimSpace(sqlText)
	// SHOW CREATE TABLE/VIEW：被查看对象即受控资源，按表级授权校验。
	if m := reShowCreateObj.FindStringSubmatch(trimmed); m != nil {
		if ref, ok := parseQualifiedName(m[1], defaultDB); ok {
			return []queryTableRef{ref}
		}
		return nil
	}
	// SHOW CREATE DATABASE/SCHEMA：整库对象，交给整库授权判定。
	if m := reShowCreateDB.FindStringSubmatch(trimmed); m != nil {
		name := strings.Trim(strings.TrimSpace(m[1]), "`")
		if name == "" {
			return nil
		}
		return []queryTableRef{{Schema: name}}
	}
	// EXPLAIN [options] <statement>：剥离前缀后按被解释语句涉及的表校验，
	// 防止 EXPLAIN/SHOW CREATE 成为跨库探测通道。
	if reExplainPrefix.MatchString(trimmed) {
		rest := strings.TrimSpace(reExplainPrefix.ReplaceAllString(trimmed, ""))
		if rest == "" {
			return nil
		}
		if !reStatementHead.MatchString(rest) {
			// EXPLAIN db.t 简写：rest 即对象名；无法识别则按“未知对象”处理。
			if ref, ok := parseQualifiedName(rest, defaultDB); ok {
				return []queryTableRef{ref}
			}
			return nil
		}
		sqlText = rest
	}
	if refs, ok := extractTableRefsAST(sqlText, defaultDB); ok {
		return refs
	}
	seen := map[string]struct{}{}
	var refs []queryTableRef
	add := func(schema, table string) {
		if table == "" {
			return
		}
		if schema == "" {
			schema = defaultDB
		}
		key := schema + "." + table
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		refs = append(refs, queryTableRef{Schema: schema, Table: table})
	}
	for _, re := range []*regexp.Regexp{reFromTable, reJoinTable} {
		for _, m := range re.FindAllStringSubmatch(sqlText, -1) {
			if len(m) >= 3 && m[2] != "" {
				add(m[1], m[2])
			} else if len(m) >= 2 {
				add(defaultDB, m[1])
			}
		}
	}
	return refs
}

func tablesAllowAll(tables []string) bool {
	if len(tables) == 0 {
		return true
	}
	for _, t := range tables {
		if t == "*" {
			return true
		}
	}
	return false
}

// grantCoversQuery 判断授权是否覆盖指定库表。
// table == "" 表示「整库或无法解析到具体表」：仅整库授权（空表列表或 *）可通过。
func grantCoversQuery(g model.DbAccessGrant, db, table string) (bool, int) {
	if !g.CanQuery && !hasPrivilege(parsePrivilegesJSON(g.PrivilegesJSON), "select") {
		return false, 0
	}
	gdb := strings.TrimSpace(g.DatabaseName)
	tables := parseTableNamesJSON(g.TableNamesJSON)
	limit := g.QueryLimitNum
	if limit <= 0 {
		limit = 1000
	}
	if gdb != "" && !strings.EqualFold(gdb, db) {
		return false, 0
	}
	if tablesAllowAll(tables) {
		return true, limit
	}
	// 表级授权：未知/未解析表名不得放行。
	if table == "" {
		return false, 0
	}
	for _, t := range tables {
		if strings.EqualFold(t, table) {
			return true, limit
		}
	}
	return false, 0
}

func (s *Service) resolveQueryAccess(ctx context.Context, projectID uint, inst *model.DbInstance, database, sqlText string, actor *auth.CurrentUser, cfgMaxRows int) (int, error) {
	if isSuperAdmin(actor) {
		return cfgMaxRows, nil
	}
	if inst.OwnerUserID != nil && *inst.OwnerUserID == actorUserID(actor) {
		return cfgMaxRows, nil
	}
	perm, err := s.effectivePermissionForDatabase(ctx, projectID, inst.ID, database, actor)
	if err != nil {
		return 0, err
	}
	if perm.CanManage {
		return cfgMaxRows, nil
	}

	refs := extractQueryTableRefs(sqlText, database)
	now := time.Now()
	limit := cfgMaxRows
	hasAnyGrant := false

	// 解析不到表：仅整库（或整实例）查询授权可放行，表级授权一律拒绝，避免绕过。
	if len(refs) == 0 {
		for _, k := range principalRefs(actor) {
			grants, err := s.repo.ListGrantsForPrincipal(ctx, projectID, inst.ID, k.kind, k.ref)
			if err != nil {
				return 0, err
			}
			for _, g := range grants {
				if !grantStillValid(g, now) || !grantMatchesDatabase(g, database) {
					continue
				}
				if ok, l := grantCoversQuery(g, database, ""); ok {
					hasAnyGrant = true
					if l > 0 && l < limit {
						limit = l
					}
				}
			}
		}
		if !hasAnyGrant {
			return 0, constants.ErrForbiddenWithMsg("无法解析 SQL 涉及的表，且你没有该库的整库查询权限；请申请库级查询权限或改用简单 SELECT")
		}
		if limit <= 0 {
			limit = 1000
		}
		return limit, nil
	}

	for _, ref := range refs {
		db := ref.Schema
		if db == "" {
			db = database
		}
		covered := false
		for _, k := range principalRefs(actor) {
			grants, err := s.repo.ListGrantsForPrincipal(ctx, projectID, inst.ID, k.kind, k.ref)
			if err != nil {
				return 0, err
			}
			for _, g := range grants {
				if !grantStillValid(g, now) {
					continue
				}
				if ok, l := grantCoversQuery(g, db, ref.Table); ok {
					covered = true
					hasAnyGrant = true
					if l > 0 && l < limit {
						limit = l
					}
				}
			}
		}
		if !covered {
			if ref.Table == "" {
				return 0, constants.ErrForbiddenWithMsg("你无 " + db + " 库的查询权限，请先申请平台查询权限")
			}
			return 0, constants.ErrForbiddenWithMsg("你无 " + db + "." + ref.Table + " 表的查询权限，请先申请平台查询权限")
		}
	}
	if !hasAnyGrant {
		return 0, constants.ErrForbiddenWithMsg("你无该库的查询权限，请先申请平台查询权限")
	}
	if limit <= 0 {
		limit = 1000
	}
	return limit, nil
}
