package dbmgmt

import (
	"context"
	"encoding/json"
	"fmt"
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
}

func normalizeInstanceRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case model.DbInstanceRoleReplica, "slave":
		return model.DbInstanceRoleReplica
	default:
		return model.DbInstanceRolePrimary
	}
}

func (s *Service) toInstanceItem(ctx context.Context, inst model.DbInstance) InstanceItem {
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
	if inst.PrimaryInstanceID != nil && *inst.PrimaryInstanceID > 0 {
		if primary, err := s.repo.GetInstanceInProject(ctx, inst.ProjectID, *inst.PrimaryInstanceID); err == nil && primary != nil {
			item.PrimaryInstanceName = primary.Name
		}
	}
	if inst.LastPingAt != nil {
		ts := inst.LastPingAt.Format(time.RFC3339)
		item.LastPingAt = &ts
	}
	item.CreatedAt = inst.CreatedAt.Format(time.RFC3339)
	if inst.ServerID != nil && *inst.ServerID > 0 {
		if sv, err := s.serverRepo.GetByID(ctx, *inst.ServerID); err == nil && sv != nil {
			item.ServerName = sv.Name
		}
	}
	if strings.EqualFold(inst.Driver, model.DbDriverMySQL) {
		item.BackupLink = fmt.Sprintf("/mysql-backup?project_id=%d", inst.ProjectID)
	}
	return item
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
	list, total, err := s.repo.ListInstances(ctx, repository.DbInstanceListParams{
		ProjectID: q.ProjectID, Env: q.Env, Keyword: q.Keyword, Page: q.Page, PageSize: q.PageSize,
	})
	if err != nil {
		return nil, err
	}
	items := make([]InstanceItem, 0, len(list))
	for _, inst := range list {
		items = append(items, s.toInstanceItem(ctx, inst))
	}
	return paginate(items, total, q.Page, q.PageSize), nil
}

func (s *Service) GetInstance(ctx context.Context, projectID, id uint) (*InstanceItem, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, id)
	if err != nil {
		return nil, err
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
	_ = s.repo.UpdateInstance(ctx, inst)
	if err != nil {
		return &PingResult{OK: false, Message: err.Error()}, nil
	}
	return &PingResult{OK: true, Message: "ok"}, nil
}

func (s *Service) writeAudit(ctx context.Context, projectID uint, instanceID *uint, actor *auth.CurrentUser, action string, detail map[string]any) error {
	b, _ := json.Marshal(detail)
	log := &model.DbAuditLog{
		ProjectID: projectID, InstanceID: instanceID,
		ActorUserID: actorUserID(actor), ActorName: actorUsername(actor),
		Action: action, DetailJSON: b,
	}
	return s.repo.CreateAuditLog(ctx, log)
}
