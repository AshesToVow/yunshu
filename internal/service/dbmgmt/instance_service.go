package dbmgmt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/dbconn"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
)

// --- Instance DTOs ---

type InstanceItem struct {
	ID            uint    `json:"id"`
	ProjectID     uint    `json:"project_id"`
	Name          string  `json:"name"`
	Env           string  `json:"env"`
	Driver        string  `json:"driver"`
	ConnectMode   string  `json:"connect_mode"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	Database      string  `json:"database"`
	ServerID      *uint   `json:"server_id,omitempty"`
	ServerName    string  `json:"server_name,omitempty"`
	Username      string  `json:"username"`
	SSLMode       string  `json:"ssl_mode"`
	ReadOnly            bool    `json:"read_only"`
	Role                string  `json:"role"`
	PrimaryInstanceID   *uint   `json:"primary_instance_id,omitempty"`
	PrimaryInstanceName string  `json:"primary_instance_name,omitempty"`
	RequireTicket       bool    `json:"require_ticket_for_dml"`
	OwnerUserID   *uint   `json:"owner_user_id,omitempty"`
	Status        string  `json:"status"`
	LastPingAt    *string `json:"last_ping_at,omitempty"`
	LastPingOK    bool    `json:"last_ping_ok"`
	Tags          string  `json:"tags"`
	Remark        string  `json:"remark"`
	BackupLink    string  `json:"backup_link,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type InstanceUpsertRequest struct {
	ProjectID             uint   `json:"project_id"`
	Name                  string `json:"name" binding:"required"`
	Env                   string `json:"env"`
	Driver                string `json:"driver"`
	ConnectMode           string `json:"connect_mode"`
	Host                  string `json:"host"`
	Port                  int    `json:"port"`
	Database              string `json:"database"`
	ServerID              *uint  `json:"server_id"`
	Username              string `json:"username" binding:"required"`
	Password              string `json:"password"`
	SSLMode               string `json:"ssl_mode"`
	ReadOnly              bool   `json:"read_only"`
	Role                  string `json:"role"`
	PrimaryInstanceID     *uint  `json:"primary_instance_id"`
	RequireTicketForDML   *bool  `json:"require_ticket_for_dml"`
	OwnerUserID           *uint  `json:"owner_user_id"`
	Tags                  string `json:"tags"`
	Remark                string `json:"remark"`
}

type InstanceListQuery struct {
	ProjectID uint
	Env       string
	Keyword   string
	Page      int `form:"page"`
	PageSize  int `form:"page_size"`
	Actor     *auth.CurrentUser `form:"-"`
}

func normalizeInstanceRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case model.DbInstanceRoleReplica, "slave":
		return model.DbInstanceRoleReplica
	default:
		return model.DbInstanceRolePrimary
	}
}

func baseInstanceItem(inst model.DbInstance) InstanceItem {
	item := InstanceItem{
		ID: inst.ID, ProjectID: inst.ProjectID, Name: inst.Name, Env: inst.Env,
		Driver: inst.Driver, ConnectMode: inst.ConnectMode, Host: inst.Host, Port: inst.Port,
		Database: inst.Database, ServerID: inst.ServerID, Username: inst.Username,
		SSLMode: inst.SSLMode, ReadOnly: inst.ReadOnly, Role: inst.Role,
		PrimaryInstanceID: inst.PrimaryInstanceID, RequireTicket: inst.RequireTicketForDML,
		OwnerUserID: inst.OwnerUserID, Status: inst.Status, LastPingOK: inst.LastPingOK,
		Tags: inst.Tags, Remark: inst.Remark,
	}
	if item.Role == "" {
		item.Role = model.DbInstanceRolePrimary
	}
	if inst.LastPingAt != nil {
		ts := inst.LastPingAt.Format(time.RFC3339)
		item.LastPingAt = &ts
	}
	item.CreatedAt = inst.CreatedAt.Format(time.RFC3339)
	if strings.EqualFold(inst.Driver, model.DbDriverMySQL) {
		item.BackupLink = fmt.Sprintf("/mysql-backup?project_id=%d", inst.ProjectID)
	}
	return item
}

func (s *Service) toInstanceItem(ctx context.Context, inst model.DbInstance) InstanceItem {
	item := baseInstanceItem(inst)
	if inst.PrimaryInstanceID != nil && *inst.PrimaryInstanceID > 0 {
		if primary, err := s.repo.GetInstanceInProject(ctx, inst.ProjectID, *inst.PrimaryInstanceID); err == nil && primary != nil {
			item.PrimaryInstanceName = primary.Name
		}
	}
	if inst.ServerID != nil && *inst.ServerID > 0 {
		if sv, err := s.serverRepo.GetByID(ctx, *inst.ServerID); err == nil && sv != nil {
			item.ServerName = sv.Name
		}
	}
	return item
}

// toInstanceItems 批量填充主库名与服务器名，避免列表接口逐条查询（N+1）。
func (s *Service) toInstanceItems(ctx context.Context, projectID uint, list []model.DbInstance) []InstanceItem {
	primaryNames := make(map[uint]string, len(list))
	needAllInstances := false
	for i := range list {
		primaryNames[list[i].ID] = list[i].Name
		if pid := list[i].PrimaryInstanceID; pid != nil && *pid > 0 {
			if _, ok := primaryNames[*pid]; !ok {
				needAllInstances = true
			}
		}
	}
	if needAllInstances {
		if all, err := s.repo.ListInstancesForVisibility(ctx, repository.DbInstanceListParams{ProjectID: projectID}); err == nil {
			for i := range all {
				primaryNames[all[i].ID] = all[i].Name
			}
		}
	}
	serverNames := make(map[uint]string)
	needServers := false
	for i := range list {
		if sid := list[i].ServerID; sid != nil && *sid > 0 {
			needServers = true
			break
		}
	}
	if needServers && s.serverRepo != nil {
		if servers, err := s.serverRepo.ListByProject(ctx, projectID); err == nil {
			for i := range servers {
				serverNames[servers[i].ID] = servers[i].Name
			}
		}
	}
	items := make([]InstanceItem, 0, len(list))
	for i := range list {
		item := baseInstanceItem(list[i])
		if pid := list[i].PrimaryInstanceID; pid != nil && *pid > 0 {
			item.PrimaryInstanceName = primaryNames[*pid]
		}
		if sid := list[i].ServerID; sid != nil && *sid > 0 {
			item.ServerName = serverNames[*sid]
		}
		items = append(items, item)
	}
	return items
}

func (s *Service) buildOpenParams(inst *model.DbInstance, password string) dbconn.OpenParams {
	return dbconn.OpenParams{
		Driver:      inst.Driver,
		Host:        inst.Host,
		Port:        inst.Port,
		Database:    inst.Database,
		Username:    inst.Username,
		Password:    password,
		SSLMode:     inst.SSLMode,
		ConnectMode: inst.ConnectMode,
		ServerID:    inst.ServerID,
	}
}

func (s *Service) openSession(ctx context.Context, inst *model.DbInstance) (*dbconn.Session, error) {
	pw, err := cryptox.DecryptString(s.aead, inst.EncPassword)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgDbInstancePasswordDecryptFailed)
	}
	return dbconn.OpenSession(ctx, s.buildOpenParams(inst, pw), sshDialer{s: s})
}

func (s *Service) ListInstances(ctx context.Context, q InstanceListQuery) (*pagination.Result[InstanceItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	params := repository.DbInstanceListParams{ProjectID: q.ProjectID, Env: q.Env, Keyword: q.Keyword}
	// 项目管理员/超管可见全部实例，可直接在数据库分页。
	if err := s.requireProjectAdminOrOwner(ctx, q.ProjectID, q.Actor); err == nil {
		params.Page, params.PageSize = page, pageSize
		list, total, err := s.repo.ListInstances(ctx, params)
		if err != nil {
			return nil, err
		}
		return paginate(s.toInstanceItems(ctx, q.ProjectID, list), total, page, pageSize), nil
	}
	// 普通成员：先取全部匹配项，按实例可见性过滤后再分页，避免页内条数与 total 失真。
	all, err := s.repo.ListInstancesForVisibility(ctx, params)
	if err != nil {
		return nil, err
	}
	visible := make([]model.DbInstance, 0, len(all))
	for i := range all {
		if s.canSeeInstance(ctx, q.ProjectID, &all[i], q.Actor) {
			visible = append(visible, all[i])
		}
	}
	total := int64(len(visible))
	start := (page - 1) * pageSize
	if start > len(visible) {
		start = len(visible)
	}
	end := start + pageSize
	if end > len(visible) {
		end = len(visible)
	}
	return paginate(s.toInstanceItems(ctx, q.ProjectID, visible[start:end]), total, page, pageSize), nil
}

// canSeeInstance 项目管理员/超管/实例 Owner 可见全部；其余需至少有连接/查询/写/管理任一授权。
func (s *Service) canSeeInstance(ctx context.Context, projectID uint, inst *model.DbInstance, actor *auth.CurrentUser) bool {
	if actor == nil {
		return false
	}
	if auth.IsSuperAdminRole(actor.RoleCodes) {
		return true
	}
	if inst.OwnerUserID != nil && *inst.OwnerUserID == actorUserID(actor) {
		return true
	}
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err == nil {
		return true
	}
	perm, err := s.GetEffectivePermission(ctx, projectID, inst.ID, actor)
	if err != nil || perm == nil {
		return false
	}
	return perm.CanManage || perm.CanQuery || perm.CanConnect || perm.CanDML || perm.CanDDL || perm.CanImport || perm.CanExport
}

func (s *Service) GetInstance(ctx context.Context, projectID, id uint, actor *auth.CurrentUser) (*InstanceItem, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	if !s.canSeeInstance(ctx, projectID, inst, actor) {
		return nil, constants.ErrForbidden
	}
	item := s.toInstanceItem(ctx, *inst)
	return &item, nil
}

func (s *Service) UpsertInstance(ctx context.Context, id uint, req InstanceUpsertRequest, actor *auth.CurrentUser) (*InstanceItem, error) {
	if err := s.requireProjectAdminOrOwner(ctx, req.ProjectID, actor); err != nil {
		return nil, err
	}
	cfg := s.resolvedConfig(ctx)
	driver := strings.ToLower(strings.TrimSpace(req.Driver))
	if driver == "" {
		driver = model.DbDriverMySQL
	}
	if !slices.Contains(cfg.AllowedDrivers, driver) {
		return nil, constants.ErrBadRequestWithMsg("不支持的驱动: " + driver)
	}
	mode := strings.ToLower(strings.TrimSpace(req.ConnectMode))
	if mode == "" {
		mode = model.DbConnectDirect
	}
	if mode == model.DbConnectSSHTunnel && (req.ServerID == nil || *req.ServerID == 0) {
		return nil, constants.ErrBadRequestWithMsg("SSH 隧道模式须绑定 CMDB 服务器")
	}

	var inst model.DbInstance
	if id > 0 {
		ex, err := s.repo.GetInstanceInProject(ctx, req.ProjectID, id)
		if err != nil {
			return nil, err
		}
		inst = *ex
	} else {
		inst.ProjectID = req.ProjectID
		inst.Status = "unknown"
		inst.RequireTicketForDML = true
	}
	inst.Name = strings.TrimSpace(req.Name)
	inst.Env = strings.TrimSpace(req.Env)
	if inst.Env == "" {
		inst.Env = model.DbEnvDev
	}
	inst.Driver = driver
	inst.ConnectMode = mode
	inst.Host = strings.TrimSpace(req.Host)
	if inst.Host == "" {
		inst.Host = "127.0.0.1"
	}
	inst.Port = req.Port
	inst.Database = strings.TrimSpace(req.Database)
	inst.ServerID = req.ServerID
	inst.Username = strings.TrimSpace(req.Username)
	inst.SSLMode = strings.TrimSpace(req.SSLMode)
	role := normalizeInstanceRole(req.Role)
	inst.Role = role
	if role == model.DbInstanceRoleReplica {
		if req.PrimaryInstanceID == nil || *req.PrimaryInstanceID == 0 {
			return nil, constants.ErrBadRequestWithMsg("从库须选择关联主库")
		}
		if id > 0 && *req.PrimaryInstanceID == id {
			return nil, constants.ErrBadRequestWithMsg("从库不能关联自身")
		}
		primary, err := s.repo.GetInstanceInProject(ctx, req.ProjectID, *req.PrimaryInstanceID)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg("关联主库不存在或不属于当前项目")
		}
		if normalizeInstanceRole(primary.Role) != model.DbInstanceRolePrimary {
			return nil, constants.ErrBadRequestWithMsg("关联目标须为主库实例")
		}
		inst.PrimaryInstanceID = req.PrimaryInstanceID
		inst.ReadOnly = true
	} else {
		inst.PrimaryInstanceID = nil
		inst.ReadOnly = req.ReadOnly
	}
	if req.RequireTicketForDML != nil {
		inst.RequireTicketForDML = *req.RequireTicketForDML
	}
	// 生产环境强制写操作走工单，禁止关闭。
	if inst.Env == model.DbEnvProd {
		inst.RequireTicketForDML = true
	}
	inst.OwnerUserID = req.OwnerUserID
	inst.Tags = strings.TrimSpace(req.Tags)
	inst.Remark = strings.TrimSpace(req.Remark)

	if pw := strings.TrimSpace(req.Password); pw != "" {
		enc, err := cryptox.EncryptString(s.aead, pw)
		if err != nil {
			return nil, err
		}
		inst.EncPassword = enc
	} else if id == 0 {
		return nil, constants.ErrBadRequestWithMsg("新建实例须填写密码")
	}

	if id > 0 {
		if err := s.repo.UpdateInstance(ctx, &inst); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.CreateInstance(ctx, &inst); err != nil {
			return nil, err
		}
	}
	_ = s.writeAudit(ctx, req.ProjectID, &inst.ID, actor, "instance_upsert", map[string]any{"name": inst.Name})
	item := s.toInstanceItem(ctx, inst)
	return &item, nil
}

func (s *Service) DeleteInstance(ctx context.Context, projectID, id uint, actor *auth.CurrentUser) error {
	if err := s.requireProjectAdminOrOwner(ctx, projectID, actor); err != nil {
		return err
	}
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, id)
	if err != nil {
		return err
	}
	if normalizeInstanceRole(inst.Role) == model.DbInstanceRolePrimary {
		n, err := s.repo.CountReplicasByPrimary(ctx, projectID, id)
		if err != nil {
			return err
		}
		if n > 0 {
			return constants.ErrBadRequestWithMsg("该主库仍有关联从库，请先删除或改绑从库")
		}
	}
	if s.workflow != nil {
		ticketIDs, accessIDs, appUserIDs, lerr := s.repo.ListInstanceLinkedRefIDs(ctx, id)
		if lerr != nil {
			return lerr
		}
		closeLinked := func(refType string, ids []uint) {
			for _, refID := range ids {
				if cerr := s.workflow.CloseLinkedTicket(ctx, refType, refID); cerr != nil {
					slog.Default().With("component", "dbmgmt").Warn("close linked workflow ticket failed",
						"ref_type", refType, "ref_id", refID, "error", cerr)
				}
			}
		}
		closeLinked(model.WorkflowRefDbSqlTicket, ticketIDs)
		closeLinked(model.WorkflowRefDbAccessRequest, accessIDs)
		closeLinked(model.WorkflowRefDbAppUserRequest, appUserIDs)
	}
	if err := s.repo.DeleteInstance(ctx, id); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, projectID, &id, actor, "instance_delete", map[string]any{"name": inst.Name})
	return nil
}

type PingResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func (s *Service) PingInstance(ctx context.Context, projectID, id uint, actor *auth.CurrentUser) (*PingResult, error) {
	if err := s.requireInstanceManage(ctx, projectID, id, actor); err != nil {
		return nil, err
	}
	return s.pingInstance(ctx, projectID, id)
}

func (s *Service) pingInstance(ctx context.Context, projectID, id uint) (*PingResult, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	pw, err := cryptox.DecryptString(s.aead, inst.EncPassword)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgDbInstancePasswordDecryptFailed)
	}
	p := s.buildOpenParams(inst, pw)
	err = dbconn.Ping(ctx, p, sshDialer{s: s})
	now := time.Now()
	inst.LastPingAt = &now
	inst.LastPingOK = err == nil
	if err != nil {
		inst.Status = "offline"
	} else {
		inst.Status = "online"
	}
	if uerr := s.repo.UpdateInstance(ctx, inst); uerr != nil {
		slog.Default().With("component", "dbmgmt").Warn("persist instance ping status failed", "instance_id", inst.ID, "error", uerr)
	}
	if err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	return &PingResult{OK: true, Message: "ok"}, nil
}

func (s *Service) writeAudit(ctx context.Context, projectID uint, instanceID *uint, actor *auth.CurrentUser, action string, detail map[string]any) error {
	b, err := json.Marshal(detail)
	if err != nil {
		// 详情序列化失败不应阻断主流程，但必须留下痕迹，避免审计静默缺失。
		slog.Default().With("component", "dbmgmt").Warn("marshal audit detail failed", "action", action, "error", err)
		b = nil
	}
	log := &model.DbAuditLog{
		ProjectID: projectID, InstanceID: instanceID,
		ActorUserID: actorUserID(actor), ActorName: actorUsername(actor),
		Action: action, DetailJSON: b,
	}
	if err := s.repo.CreateAuditLog(ctx, log); err != nil {
		slog.Default().With("component", "dbmgmt").Error("write audit log failed", "action", action, "error", err)
		return err
	}
	return nil
}
