package dbmgmt

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

type dbMetaAccess struct {
	allTables bool
	tables    map[string]struct{}
}

type metadataScope struct {
	unrestricted bool
	// allDatabases：授权未指定 database_name 时允许枚举全部库，但仍受表级限制（不再等同 manage 级 unrestricted）。
	allDatabases bool
	dbs          map[string]*dbMetaAccess
}

func (s *Service) resolveMetadataScope(ctx context.Context, projectID uint, inst *model.DbInstance, actor *auth.CurrentUser) (*metadataScope, error) {
	out := &metadataScope{dbs: map[string]*dbMetaAccess{}}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		out.unrestricted = true
		return out, nil
	}
	if inst.OwnerUserID != nil && *inst.OwnerUserID == actorUserID(actor) {
		out.unrestricted = true
		return out, nil
	}
	perm, err := s.GetEffectivePermission(ctx, projectID, inst.ID, actor)
	if err != nil {
		return nil, err
	}
	if perm.CanManage {
		out.unrestricted = true
		return out, nil
	}

	now := time.Now()
	for _, k := range principalRefs(actor) {
		grants, err := s.repo.ListGrantsForPrincipal(ctx, projectID, inst.ID, k.kind, k.ref)
		if err != nil {
			return nil, err
		}
		for _, g := range grants {
			if !grantStillValid(g, now) {
				continue
			}
			if !g.CanQuery && !g.CanManage && !hasPrivilege(parsePrivilegesJSON(g.PrivilegesJSON), "select") {
				continue
			}
			db := strings.TrimSpace(g.DatabaseName)
			if db == "" {
				out.allDatabases = true
				m := out.dbs["*"]
				if m == nil {
					m = &dbMetaAccess{}
					out.dbs["*"] = m
				}
				tables := parseTableNamesJSON(g.TableNamesJSON)
				if tablesAllowAll(tables) {
					m.allTables = true
					m.tables = nil
				} else if !m.allTables {
					if m.tables == nil {
						m.tables = map[string]struct{}{}
					}
					for _, t := range tables {
						m.tables[strings.ToLower(t)] = struct{}{}
					}
				}
				continue
			}
			key := strings.ToLower(db)
			m := out.dbs[key]
			if m == nil {
				m = &dbMetaAccess{}
				out.dbs[key] = m
			}
			tables := parseTableNamesJSON(g.TableNamesJSON)
			if tablesAllowAll(tables) {
				m.allTables = true
				m.tables = nil
				continue
			}
			if m.allTables {
				continue
			}
			if m.tables == nil {
				m.tables = map[string]struct{}{}
			}
			for _, t := range tables {
				m.tables[strings.ToLower(t)] = struct{}{}
			}
		}
	}
	return out, nil
}

func (sc *metadataScope) allowsDatabase(name string) bool {
	if sc == nil || sc.unrestricted || sc.allDatabases {
		return true
	}
	_, ok := sc.dbs[strings.ToLower(strings.TrimSpace(name))]
	return ok
}

func (sc *metadataScope) filterDatabases(list []DatabaseInfo) []DatabaseInfo {
	if sc == nil || sc.unrestricted || sc.allDatabases {
		return list
	}
	out := make([]DatabaseInfo, 0, len(list))
	for _, d := range list {
		if sc.allowsDatabase(d.Name) {
			out = append(out, d)
		}
	}
	return out
}

func (sc *metadataScope) filterTables(database string, list []TableInfo) []TableInfo {
	if sc == nil || sc.unrestricted {
		return list
	}
	m := sc.dbs[strings.ToLower(strings.TrimSpace(database))]
	if m == nil && sc.allDatabases {
		m = sc.dbs["*"]
	}
	if m == nil {
		return nil
	}
	if m.allTables {
		return list
	}
	out := make([]TableInfo, 0, len(list))
	for _, t := range list {
		if _, ok := m.tables[strings.ToLower(t.Name)]; ok {
			out = append(out, t)
		}
	}
	return out
}

func (sc *metadataScope) allowsTable(database, table string) bool {
	if sc == nil || sc.unrestricted {
		return true
	}
	m := sc.dbs[strings.ToLower(strings.TrimSpace(database))]
	if m == nil && sc.allDatabases {
		m = sc.dbs["*"]
	}
	if m == nil {
		return false
	}
	if m.allTables {
		return true
	}
	_, ok := m.tables[strings.ToLower(strings.TrimSpace(table))]
	return ok
}

func (s *Service) requireMetadataDatabaseAccess(ctx context.Context, projectID uint, inst *model.DbInstance, database string, actor *auth.CurrentUser) error {
	sc, err := s.resolveMetadataScope(ctx, projectID, inst, actor)
	if err != nil {
		return err
	}
	if !sc.allowsDatabase(database) {
		return constants.ErrForbiddenWithMsg("你无该库的元数据访问权限")
	}
	return nil
}

func (s *Service) requireMetadataTableAccess(ctx context.Context, projectID uint, inst *model.DbInstance, database, table string, actor *auth.CurrentUser) error {
	sc, err := s.resolveMetadataScope(ctx, projectID, inst, actor)
	if err != nil {
		return err
	}
	if !sc.allowsTable(database, table) {
		return constants.ErrForbiddenWithMsg("你无该表的元数据访问权限")
	}
	return nil
}
