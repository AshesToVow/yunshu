package dbmgmt

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
)

// EffectivePermission 用户对实例的有效权限。
type EffectivePermission struct {
	CanConnect bool     `json:"can_connect"`
	CanQuery   bool     `json:"can_query"`
	CanDML     bool     `json:"can_dml"`
	CanDDL     bool     `json:"can_ddl"`
	CanExport  bool     `json:"can_export"`
	CanImport  bool     `json:"can_import"`
	CanManage  bool     `json:"can_manage"`
	Databases  []string `json:"databases,omitempty"`
}

type GrantItem struct {
	ID             uint    `json:"id"`
	ProjectID      uint    `json:"project_id"`
	InstanceID     uint    `json:"instance_id"`
	PrincipalKind  string  `json:"principal_kind"`
	PrincipalRef   string  `json:"principal_ref"`
	DatabaseName   string  `json:"database_name"`
	TableNames     []string `json:"table_names"`
	CanConnect     bool    `json:"can_connect"`
	CanQuery       bool    `json:"can_query"`
	CanDML         bool    `json:"can_dml"`
	CanDDL         bool    `json:"can_ddl"`
	CanExport      bool    `json:"can_export"`
	CanImport      bool    `json:"can_import"`
	CanManage      bool     `json:"can_manage"`
	QueryLimitNum  int      `json:"query_limit_num"`
	Privileges     []string `json:"privileges,omitempty"`
	ExpiresAt      *string  `json:"expires_at,omitempty"`
	Remark         string  `json:"remark"`
}

type GrantUpsertRequest struct {
	InstanceID    uint     `json:"instance_id" binding:"required"`
	PrincipalKind string   `json:"principal_kind" binding:"required"`
	PrincipalRef  string   `json:"principal_ref" binding:"required"`
	DatabaseName  string   `json:"database_name"`
	TableNames    []string `json:"table_names"`
	CanConnect    bool     `json:"can_connect"`
	CanQuery      bool     `json:"can_query"`
	CanDML        bool     `json:"can_dml"`
	CanDDL        bool     `json:"can_ddl"`
	CanExport     bool     `json:"can_export"`
	CanImport     bool     `json:"can_import"`
	CanManage     bool     `json:"can_manage"`
	QueryLimitNum int      `json:"query_limit_num"`
	Privileges    []string `json:"privileges,omitempty"`
	ExpiresAt     *string  `json:"expires_at"`
	Remark        string   `json:"remark"`
}

type GrantUpdateRequest struct {
	QueryLimitNum *int    `json:"query_limit_num"`
	ExpiresAt     *string `json:"expires_at"`
	Remark        *string `json:"remark"`
}

func parseTableNamesJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var arr []string
	_ = json.Unmarshal([]byte(raw), &arr)
	return arr
}

func parseOptionalExpiresAt(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("无效的过期时间格式")
	}
	if !t.After(time.Now()) {
		return nil, constants.ErrBadRequestWithMsg("过期时间须晚于当前时间")
	}
	return &t, nil
}

func formatTimeRFC3339(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func grantStillValid(g model.DbAccessGrant, now time.Time) bool {
	return g.ExpiresAt == nil || g.ExpiresAt.After(now)
}

func (s *Service) toGrantItem(g model.DbAccessGrant) GrantItem {
	privs := parsePrivilegesJSON(g.PrivilegesJSON)
	if len(privs) == 0 {
		privs = privilegesFromFlags(g.CanQuery, g.CanDML, g.CanDDL, g.CanExport, g.CanImport)
	}
	item := GrantItem{
		ID: g.ID, ProjectID: g.ProjectID, InstanceID: g.InstanceID,
		PrincipalKind: g.PrincipalKind, PrincipalRef: g.PrincipalRef,
		DatabaseName: g.DatabaseName, TableNames: parseTableNamesJSON(g.TableNamesJSON),
		CanConnect: g.CanConnect, CanQuery: g.CanQuery, CanDML: g.CanDML, CanDDL: g.CanDDL,
		CanExport: g.CanExport, CanImport: g.CanImport, CanManage: g.CanManage,
		QueryLimitNum: g.QueryLimitNum,
		Privileges: privs, Remark: g.Remark,
	}
	item.ExpiresAt = formatTimeRFC3339(g.ExpiresAt)
	return item
}

func (s *Service) ListGrants(ctx context.Context, projectID uint, instanceID uint) ([]GrantItem, error) {
	list, err := s.repo.ListGrants(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	out := make([]GrantItem, 0, len(list))
	for _, g := range list {
		out = append(out, s.toGrantItem(g))
	}
	return out, nil
}

func (s *Service) CreateGrant(ctx context.Context, projectID uint, req GrantUpsertRequest, actor *auth.CurrentUser) (*GrantItem, error) {
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetInstanceInProject(ctx, projectID, req.InstanceID); err != nil {
		return nil, err
	}
	expiresAt, err := parseOptionalExpiresAt(req.ExpiresAt)
	if err != nil {
		return nil, err
	}
	privs := normalizePrivileges(req.Privileges)
	canQuery, canDML, canDDL, canExport, canImport := flagsFromPrivileges(privs)
	if len(privs) == 0 {
		canQuery, canDML, canDDL, canExport, canImport = req.CanQuery, req.CanDML, req.CanDDL, req.CanExport, req.CanImport
		privs = privilegesFromFlags(canQuery, canDML, canDDL, canExport, canImport)
	}
	if len(privs) == 0 && !req.CanManage {
		return nil, constants.ErrBadRequestWithMsg("请至少选择一项权限")
	}
	queryLimit := req.QueryLimitNum
	if queryLimit <= 0 {
		queryLimit = 1000
	}
	if !canQuery || canDML || canDDL || canExport || canImport {
		queryLimit = 0
	}
	tb, _ := json.Marshal(req.TableNames)
	g := &model.DbAccessGrant{
		ProjectID: projectID, InstanceID: req.InstanceID,
		PrincipalKind: req.PrincipalKind, PrincipalRef: req.PrincipalRef,
		DatabaseName: req.DatabaseName, TableNamesJSON: string(tb),
		CanConnect: req.CanConnect, CanQuery: canQuery, CanDML: canDML, CanDDL: canDDL,
		CanExport: canExport, CanImport: canImport, CanManage: req.CanManage,
		QueryLimitNum: queryLimit,
		PrivilegesJSON: marshalPrivilegesJSON(privs),
		ExpiresAt: expiresAt, Remark: req.Remark,
		CreatedByUserID: ptrUint(actorUserID(actor)),
	}
	if err := s.repo.CreateGrant(ctx, g); err != nil {
		return nil, err
	}
	iid := req.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "grant_create", map[string]any{
		"grant_id": g.ID, "principal_kind": req.PrincipalKind, "principal_ref": req.PrincipalRef,
		"database": req.DatabaseName, "privileges": privs,
	})
	item := s.toGrantItem(*g)
	return &item, nil
}

func (s *Service) DeleteGrant(ctx context.Context, projectID, id uint, actor *auth.CurrentUser) error {
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err != nil {
		return err
	}
	g, err := s.repo.GetGrantInProject(ctx, projectID, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteGrant(ctx, g.ID); err != nil {
		return err
	}
	iid := g.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "grant_delete", map[string]any{
		"grant_id": g.ID, "principal_kind": g.PrincipalKind, "principal_ref": g.PrincipalRef,
		"database": g.DatabaseName,
	})
	return nil
}

func (s *Service) UpdateGrant(ctx context.Context, projectID, id uint, req GrantUpdateRequest, actor *auth.CurrentUser) (*GrantItem, error) {
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err != nil {
		return nil, err
	}
	g, err := s.repo.GetGrantInProject(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if req.QueryLimitNum != nil {
		limit := *req.QueryLimitNum
		if limit <= 0 {
			limit = 1000
		}
		g.QueryLimitNum = limit
	}
	if req.ExpiresAt != nil {
		expiresAt, err := parseOptionalExpiresAt(req.ExpiresAt)
		if err != nil {
			return nil, err
		}
		g.ExpiresAt = expiresAt
	}
	if req.Remark != nil {
		g.Remark = strings.TrimSpace(*req.Remark)
	}
	if err := s.repo.UpdateGrant(ctx, g); err != nil {
		return nil, err
	}
	iid := g.InstanceID
	_ = s.writeAudit(ctx, projectID, &iid, actor, "grant_update", map[string]any{
		"grant_id": g.ID, "principal_kind": g.PrincipalKind, "principal_ref": g.PrincipalRef,
		"database": g.DatabaseName,
	})
	item := s.toGrantItem(*g)
	return &item, nil
}

func (s *Service) mergeGrant(dst *EffectivePermission, g model.DbAccessGrant) {
	if g.CanConnect {
		dst.CanConnect = true
	}
	if g.CanQuery {
		dst.CanQuery = true
	}
	if g.CanDML {
		dst.CanDML = true
	}
	if g.CanDDL {
		dst.CanDDL = true
	}
	if g.CanExport {
		dst.CanExport = true
	}
	if g.CanImport {
		dst.CanImport = true
	}
	if g.CanManage {
		dst.CanManage = true
	}
}

func (s *Service) GetEffectivePermission(ctx context.Context, projectID, instanceID uint, actor *auth.CurrentUser) (*EffectivePermission, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	out := &EffectivePermission{}
	if inst.OwnerUserID != nil && *inst.OwnerUserID == actorUserID(actor) {
		out.CanConnect, out.CanQuery, out.CanDML, out.CanDDL = true, true, true, true
		out.CanExport, out.CanImport, out.CanManage = true, true, true
		return out, nil
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		out.CanConnect, out.CanQuery, out.CanDML, out.CanDDL = true, true, true, true
		out.CanExport, out.CanImport, out.CanManage = true, true, true
		return out, nil
	}
	now := time.Now()
	for _, k := range principalRefs(actor) {
		grants, err := s.repo.ListGrantsForPrincipal(ctx, projectID, instanceID, k.kind, k.ref)
		if err != nil {
			return nil, err
		}
		for _, g := range grants {
			if !grantStillValid(g, now) {
				continue
			}
			s.mergeGrant(out, g)
			if db := strings.TrimSpace(g.DatabaseName); db != "" {
				out.Databases = appendUniqueString(out.Databases, db)
			}
		}
	}
	if inst.ReadOnly {
		out.CanDML, out.CanDDL, out.CanImport = false, false, false
	}
	return out, nil
}

func appendUniqueString(list []string, v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return list
	}
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return list
		}
	}
	return append(list, v)
}

func (s *Service) checkMetadataPermission(ctx context.Context, projectID uint, inst *model.DbInstance, actor *auth.CurrentUser) error {
	perm, err := s.GetEffectivePermission(ctx, projectID, inst.ID, actor)
	if err != nil {
		return err
	}
	if perm.CanManage || perm.CanQuery {
		return nil
	}
	return constants.ErrForbidden
}

func (s *Service) checkQueryPermission(ctx context.Context, projectID uint, inst *model.DbInstance, database string, actor *auth.CurrentUser) error {
	perm, err := s.effectivePermissionForDatabase(ctx, projectID, inst.ID, database, actor)
	if err != nil {
		return err
	}
	if perm.CanManage || perm.CanQuery {
		return nil
	}
	return constants.ErrForbiddenWithMsg("你无该库的查询权限，请先申请平台查询权限")
}

func (s *Service) checkWritePermission(ctx context.Context, projectID uint, inst *model.DbInstance, database string, needDDL bool, actor *auth.CurrentUser) error {
	perm, err := s.effectivePermissionForDatabase(ctx, projectID, inst.ID, database, actor)
	if err != nil {
		return err
	}
	if perm.CanManage {
		return nil
	}
	if inst.ReadOnly {
		return constants.ErrBadRequestWithMsg("只读实例禁止写操作")
	}
	if needDDL {
		if !perm.CanDDL {
			return constants.ErrForbidden
		}
	} else if !perm.CanDML {
		return constants.ErrForbidden
	}
	return nil
}

func ptrUint(v uint) *uint {
	if v == 0 {
		return nil
	}
	return &v
}
