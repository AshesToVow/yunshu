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
	reExplainShow = regexp.MustCompile(`(?i)^\s*(EXPLAIN|SHOW\s+CREATE)\b`)
	reFromTable   = regexp.MustCompile("(?i)\\bFROM\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
	reJoinTable   = regexp.MustCompile("(?i)\\bJOIN\\s+`?([a-zA-Z0-9_]+)`?(?:\\.`?([a-zA-Z0-9_]+)`?)?")
)

type queryTableRef struct {
	Schema string
	Table  string
}

func extractQueryTableRefs(sqlText, defaultDB string) []queryTableRef {
	sqlText = stripSQLComments(sqlText)
	if reExplainShow.MatchString(strings.TrimSpace(sqlText)) {
		return nil
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
	if auth.IsSuperAdminRole(actor.RoleCodes) {
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

	// EXPLAIN / SHOW CREATE：允许任意命中该库的查询授权（含表级）。
	if reExplainShow.MatchString(strings.TrimSpace(sqlText)) {
		for _, k := range principalRefs(actor) {
			grants, err := s.repo.ListGrantsForPrincipal(ctx, projectID, inst.ID, k.kind, k.ref)
			if err != nil {
				return 0, err
			}
			for _, g := range grants {
				if !grantStillValid(g, now) || !grantMatchesDatabase(g, database) {
					continue
				}
				if !g.CanQuery && !hasPrivilege(parsePrivilegesJSON(g.PrivilegesJSON), "select") {
					continue
				}
				hasAnyGrant = true
				if g.QueryLimitNum > 0 && g.QueryLimitNum < limit {
					limit = g.QueryLimitNum
				}
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
