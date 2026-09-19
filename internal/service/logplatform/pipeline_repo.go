package logplatform

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

// LogPipelineUpsert 创建/更新 Pipeline 仓库条目。
type LogPipelineUpsert struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"` // host|k8s|template
	ClusterID    uint   `json:"cluster_id"`
	ServerID     uint   `json:"server_id"`
	ParseProfile string `json:"parse_profile"`
	ContentYAML  string `json:"content_yml"`
	Status       string `json:"status"` // draft|published
	Remark       string `json:"remark"`
	SourceRef    string `json:"source_ref"`
}

// LogPipelineApplyRequest 将仓库内容应用到运行中的采集配置。
type LogPipelineApplyRequest struct {
	ApplyDeploy bool   `json:"apply_deploy"` // k8s 时是否立刻 Deploy DaemonSet
	Namespace   string `json:"namespace"`
}

// ListLogPipelines 列出项目下 Pipeline 仓库。
func (s *ClusterLogService) ListLogPipelines(ctx context.Context, projectID uint, kind string) ([]model.LogPipeline, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	return s.pipelineRepo.List(ctx, projectID, kind)
}

// GetLogPipeline 获取单条。
func (s *ClusterLogService) GetLogPipeline(ctx context.Context, projectID, id uint) (*model.LogPipeline, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	row, err := s.pipelineRepo.GetByIDInProject(ctx, projectID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFoundWithMsg("Pipeline 不存在")
		}
		return nil, err
	}
	return row, nil
}

// UpsertLogPipeline 新建或更新（id=0 新建）。
func (s *ClusterLogService) UpsertLogPipeline(ctx context.Context, projectID, id, userID uint, req LogPipelineUpsert) (*model.LogPipeline, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, constants.ErrBadRequestWithMsg("名称不能为空")
	}
	kind := normalizePipelineKind(req.Kind)
	yml := strings.TrimSpace(req.ContentYAML)
	if yml != "" && !strings.Contains(yml, "pipelines:") {
		return nil, constants.ErrBadRequestWithMsg("content_yml 须包含 pipelines: 根节点")
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "draft"
	}
	if status != "draft" && status != "published" {
		return nil, constants.ErrBadRequestWithMsg("status 仅支持 draft|published")
	}

	if id > 0 {
		row, err := s.pipelineRepo.GetByIDInProject(ctx, projectID, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFoundWithMsg("Pipeline 不存在")
			}
			return nil, err
		}
		row.Name = name
		row.Kind = kind
		row.ClusterID = req.ClusterID
		row.ServerID = req.ServerID
		row.ParseProfile = strings.TrimSpace(req.ParseProfile)
		if yml != "" && yml != strings.TrimSpace(row.ContentYAML) {
			_ = s.snapshotPipelineVersion(ctx, row, userID, "auto before update")
			row.ContentYAML = req.ContentYAML
			row.Version++
		}
		row.Status = status
		row.Remark = strings.TrimSpace(req.Remark)
		if ref := strings.TrimSpace(req.SourceRef); ref != "" {
			row.SourceRef = ref
		}
		row.UpdatedBy = userID
		if err := s.pipelineRepo.Save(ctx, row); err != nil {
			return nil, err
		}
		return row, nil
	}

	row := model.LogPipeline{
		ProjectID:    projectID,
		Name:         name,
		Kind:         kind,
		ClusterID:    req.ClusterID,
		ServerID:     req.ServerID,
		ParseProfile: strings.TrimSpace(req.ParseProfile),
		ContentYAML:  req.ContentYAML,
		Status:       status,
		Version:      1,
		SourceRef:    strings.TrimSpace(req.SourceRef),
		Remark:       strings.TrimSpace(req.Remark),
		UpdatedBy:    userID,
	}
	// 软删后同名仍占 uk_log_pipeline_proj_name，新建会 500；先回收再写。
	ghost, gerr := s.pipelineRepo.GetUnscopedByProjectAndName(ctx, projectID, name)
	if gerr == nil {
		if ghost.DeletedAt.Valid {
			ghost.DeletedAt = gorm.DeletedAt{}
			ghost.Kind = kind
			ghost.ClusterID = req.ClusterID
			ghost.ServerID = req.ServerID
			ghost.ParseProfile = strings.TrimSpace(req.ParseProfile)
			if yml != "" {
				_ = s.snapshotPipelineVersion(ctx, ghost, userID, "restore before sync")
				ghost.ContentYAML = req.ContentYAML
				ghost.Version++
			}
			ghost.Status = status
			ghost.Remark = strings.TrimSpace(req.Remark)
			if ref := strings.TrimSpace(req.SourceRef); ref != "" {
				ghost.SourceRef = ref
			}
			ghost.UpdatedBy = userID
			if err := s.pipelineRepo.UnscopedSave(ctx, ghost); err != nil {
				return nil, err
			}
			return ghost, nil
		}
		return nil, constants.ErrBadRequestWithMsg("Pipeline 名称已存在: " + name)
	}
	if gerr != nil && !errors.Is(gerr, gorm.ErrRecordNotFound) {
		return nil, gerr
	}
	if err := s.pipelineRepo.Create(ctx, &row); err != nil {
		if isDuplicateKeyErr(err) {
			return nil, constants.ErrBadRequestWithMsg("Pipeline 名称已存在: " + name)
		}
		return nil, err
	}
	if strings.TrimSpace(row.ContentYAML) != "" {
		_ = s.snapshotPipelineVersion(ctx, &row, userID, "initial")
	}
	return &row, nil
}

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "1062") || strings.Contains(msg, "unique")
}

// DeleteLogPipeline 删除仓库条目。
func (s *ClusterLogService) DeleteLogPipeline(ctx context.Context, projectID, id uint) error {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return err
	}
	n, err := s.pipelineRepo.DeleteByIDInProject(ctx, projectID, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return constants.ErrNotFoundWithMsg("Pipeline 不存在")
	}
	return nil
}

// SyncLogPipelinesFromCluster 从集群当前 pipelines 同步进仓库（有则更新同名条目）。
func (s *ClusterLogService) SyncLogPipelinesFromCluster(ctx context.Context, projectID, clusterID, userID uint) (*model.LogPipeline, error) {
	prev, err := s.PreviewPipelines(ctx, projectID, clusterID)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("k8s-cluster-%d", clusterID)
	existing, err := s.pipelineRepo.GetByProjectKindClusterName(ctx, projectID, "k8s", clusterID, name)
	req := LogPipelineUpsert{
		Name:        name,
		Kind:        "k8s",
		ClusterID:   clusterID,
		ContentYAML: prev.PipelinesYAML,
		Status:      "published",
		SourceRef:   fmt.Sprintf("cluster:%d", clusterID),
		Remark:      "从集群采集配置同步",
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.UpsertLogPipeline(ctx, projectID, 0, userID, req)
	}
	if err != nil {
		return nil, err
	}
	return s.UpsertLogPipeline(ctx, projectID, existing.ID, userID, req)
}

// ApplyLogPipeline 将仓库 YAML 写回集群自定义 pipelines（host 仅标记 published）。
func (s *ClusterLogService) ApplyLogPipeline(ctx context.Context, projectID, id, userID uint, req LogPipelineApplyRequest) (*model.LogPipeline, error) {
	row, err := s.GetLogPipeline(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	yml := strings.TrimSpace(row.ContentYAML)
	if yml == "" {
		return nil, constants.ErrBadRequestWithMsg("Pipeline 内容为空")
	}
	switch row.Kind {
	case "k8s":
		if row.ClusterID == 0 {
			return nil, constants.ErrBadRequestWithMsg("该条目未绑定 cluster_id")
		}
		if _, err := s.SavePipelines(ctx, projectID, ClusterPipelinesUpsert{
			ClusterID:     row.ClusterID,
			PipelinesYAML: row.ContentYAML,
			Namespace:     req.Namespace,
			Apply:         req.ApplyDeploy,
		}); err != nil {
			return nil, err
		}
	case "host":
		// 主机下发由 Loggie DeployCustomPipelinesYAML API 完成；此处仅标记 published。
		if row.ServerID == 0 {
			return nil, constants.ErrBadRequestWithMsg("该条目未绑定 server_id，无法标记下发")
		}
	case "template":
		return nil, constants.ErrBadRequestWithMsg("模板类型请先复制为 host/k8s 条目再下发")
	default:
		return nil, constants.ErrBadRequestWithMsg("不支持的 kind")
	}
	row.Status = "published"
	row.UpdatedBy = userID
	if err := s.pipelineRepo.Save(ctx, row); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *ClusterLogService) snapshotPipelineVersion(ctx context.Context, row *model.LogPipeline, userID uint, remark string) error {
	if row == nil || row.ID == 0 {
		return nil
	}
	yml := strings.TrimSpace(row.ContentYAML)
	if yml == "" {
		return nil
	}
	ver := model.LogPipelineVersion{
		PipelineID:  row.ID,
		ProjectID:   row.ProjectID,
		Version:     row.Version,
		ContentYAML: row.ContentYAML,
		Remark:      remark,
		CreatedBy:   userID,
	}
	return s.pipelineRepo.CreateVersion(ctx, &ver)
}

// ListLogPipelineVersions 列出历史版本。
func (s *ClusterLogService) ListLogPipelineVersions(ctx context.Context, projectID, pipelineID uint) ([]model.LogPipelineVersion, error) {
	if _, err := s.GetLogPipeline(ctx, projectID, pipelineID); err != nil {
		return nil, err
	}
	return s.pipelineRepo.ListVersions(ctx, projectID, pipelineID, 50)
}

// RollbackLogPipelineVersion 回滚到指定历史版本（会再产生新 version）。
func (s *ClusterLogService) RollbackLogPipelineVersion(ctx context.Context, projectID, pipelineID, versionID, userID uint) (*model.LogPipeline, error) {
	row, err := s.GetLogPipeline(ctx, projectID, pipelineID)
	if err != nil {
		return nil, err
	}
	ver, err := s.pipelineRepo.GetVersion(ctx, projectID, pipelineID, versionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFoundWithMsg("版本不存在")
		}
		return nil, err
	}
	return s.UpsertLogPipeline(ctx, projectID, row.ID, userID, LogPipelineUpsert{
		Name:         row.Name,
		Kind:         row.Kind,
		ClusterID:    row.ClusterID,
		ServerID:     row.ServerID,
		ParseProfile: row.ParseProfile,
		ContentYAML:  ver.ContentYAML,
		Status:       row.Status,
		Remark:       fmt.Sprintf("rollback to v%d", ver.Version),
		SourceRef:    row.SourceRef,
	})
}

// ListParseProfiles 暴露解析档选项给前端。
func (s *ClusterLogService) ListParseProfiles() []ParseProfileOption {
	return ListParseProfileOptions()
}

func normalizePipelineKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "host", "agent":
		return "host"
	case "template":
		return "template"
	default:
		return "k8s"
	}
}

// SaveHostPipelineSnapshot 将主机生成的 pipelines.yml 写入仓库。
func (s *ClusterLogService) SaveHostPipelineSnapshot(ctx context.Context, projectID, serverID, userID uint, name, yml, remark string) (*model.LogPipeline, error) {
	if err := s.ensureProject(ctx, projectID); err != nil {
		return nil, err
	}
	if serverID == 0 {
		return nil, constants.ErrBadRequestWithMsg("server_id 无效")
	}
	yml = strings.TrimSpace(yml)
	if yml == "" || !strings.Contains(yml, "pipelines:") {
		return nil, constants.ErrBadRequestWithMsg("pipelines.yml 无效")
	}
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("host-server-%d", serverID)
	}
	req := LogPipelineUpsert{
		Name:        name,
		Kind:        "host",
		ServerID:    serverID,
		ContentYAML: yml,
		Status:      "draft",
		SourceRef:   fmt.Sprintf("server:%d", serverID),
		Remark:      remark,
	}
	// 优先按 server_id 定位；再按名称回收（含软删）误绑/已删条目
	existing, err := s.pipelineRepo.GetHostByServer(ctx, projectID, serverID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		existing, err = s.pipelineRepo.GetUnscopedHostByName(ctx, projectID, name)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.UpsertLogPipeline(ctx, projectID, 0, userID, req)
	}
	if err != nil {
		return nil, err
	}
	if existing.DeletedAt.Valid {
		existing.DeletedAt = gorm.DeletedAt{}
		if err := s.pipelineRepo.UnscopedSave(ctx, existing); err != nil {
			return nil, err
		}
	}
	return s.UpsertLogPipeline(ctx, projectID, existing.ID, userID, req)
}
