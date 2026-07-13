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
	if len(refs) == 0 && defaultDB != "" {
		refs = append(refs, queryTableRef{Schema: defaultDB, Table: ""})
	}
	return refs
}

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
	if gdb == "" {
		return true, limit
	}
	if gdb != db {
		return false, 0
	}
	if len(tables) == 0 || table == "" {
		return true, limit
	}
	for _, t := range tables {
		if t == table || t == "*" {
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
	perm, err := s.GetEffectivePermission(ctx, projectID, inst.ID, actor)
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

	for _, k := range principalRefs(actor) {
		grants, err := s.repo.ListGrantsForPrincipal(ctx, projectID, inst.ID, k.kind, k.ref)
		if err != nil {
			return 0, err
		}
		for _, g := range grants {
			if !grantStillValid(g, now) {
				continue
			}
			if len(refs) == 0 {
				if ok, l := grantCoversQuery(g, database, ""); ok {
					hasAnyGrant = true
					if l > 0 && l < limit {
						limit = l
					}
				}
				continue
			}
			for _, ref := range refs {
				db := ref.Schema
				if db == "" {
					db = database
				}
				if ok, l := grantCoversQuery(g, db, ref.Table); ok {
					hasAnyGrant = true
					if l > 0 && l < limit {
						limit = l
					}
				} else if ref.Table != "" {
					return 0, constants.ErrForbiddenWithMsg("你无 " + db + "." + ref.Table + " 表的查询权限，请先申请平台查询权限")
				}
			}
		}
	}

	if !hasAnyGrant {
		if perm.CanQuery {
			return cfgMaxRows, nil
		}
		return 0, constants.ErrForbiddenWithMsg("你无该库的查询权限，请先申请平台查询权限")
	}
	if limit <= 0 {
		limit = 1000
	}
	return limit, nil
}
