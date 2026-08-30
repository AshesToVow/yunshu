package cicd

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/jenkins"
)

// --- Deploy Config ---

type DeployConfigUpsertRequest struct {
	Name                 string `json:"name" binding:"required,max=128"`
	DeployKind           string `json:"deploy_kind" binding:"required,oneof=regular container"`
	Tenv                 string `json:"tenv" binding:"required,max=16"`
	AuditEnabled         bool   `json:"audit_enabled"`
	Importance           string `json:"importance" binding:"omitempty,max=32"`
	DestPath             string `json:"dest_path" binding:"omitempty,max=512"`
	ServerIDs            []uint `json:"server_ids"`
	DeployUser           string `json:"deploy_user" binding:"omitempty,max=64"`
	DeployGroup          string `json:"deploy_group" binding:"omitempty,max=64"`
	ArtifactRetainCount  int    `json:"artifact_retain_count"`
	RunUser              string `json:"run_user" binding:"omitempty,max=64"`
	StartScriptType      string `json:"start_script_type" binding:"omitempty,max=32"`
	CustomScriptContent  string `json:"custom_script_content"`
	CleanDeployDir       bool   `json:"clean_deploy_dir"`
	JVMOpts              string `json:"jvm_opts"`
	ServerPort           int    `json:"server_port"`
	DeployMethod         string `json:"deploy_method" binding:"omitempty,max=16"`
	DeployAction         string `json:"deploy_action" binding:"omitempty,max=32"`
	DeployConfigType     string `json:"deploy_config_type" binding:"omitempty,max=64"`
	DeployConfigTemplate string `json:"deploy_config_template" binding:"omitempty,max=64"`
	K8sNamespace         string `json:"k8s_namespace" binding:"omitempty,max=128"`
	K8sClusterID         *uint  `json:"k8s_cluster_id"`
	ImageName            string `json:"image_name" binding:"omitempty,max=128"`
	ImageTag             string `json:"image_tag" binding:"omitempty,max=128"`
	Replicas             int    `json:"replicas"`
	ContainerPort        int    `json:"container_port"`
	DeployStrategy       string `json:"deploy_strategy" binding:"omitempty,oneof=rolling canary blue_green"`
	CanaryReplicas       int    `json:"canary_replicas"`
	CanaryPercent        int    `json:"canary_percent"`
	CanaryStepsJSON      string `json:"canary_steps_json" binding:"omitempty,max=128"`
	BlueGreenService     string `json:"blue_green_service" binding:"omitempty,max=128"`
	Status               *int   `json:"status" binding:"omitempty,oneof=0 1"`
}

type DeployConfigListItem struct {
	model.CicdDeployConfig
	ServerCount int    `json:"server_count"`
	NodesStatus string `json:"nodes_status"`
}

// DeployConfigUpsertResult 保存发布配置结果（DB 保存与 Jenkins 同步解耦）。
type DeployConfigUpsertResult struct {
	Config           *model.CicdDeployConfig `json:"config"`
	JenkinsSync      *JenkinsSyncResult      `json:"jenkins_sync,omitempty"`
	JenkinsSyncError string                  `json:"jenkins_sync_error,omitempty"`
}

func (s *Service) ListDeployConfigs(ctx context.Context, projectID, serviceID uint) ([]DeployConfigListItem, error) {
	if _, err := s.loadService(ctx, projectID, serviceID); err != nil {
		return nil, err
	}
	var rows []model.CicdDeployConfig
	if err := s.db.WithContext(ctx).Where("service_id = ?", serviceID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	serverMap := s.loadProjectServerMap(ctx, projectID)
	items := make([]DeployConfigListItem, 0, len(rows))
	for _, row := range rows {
		cnt, status := summarizeDeployNodes(row, serverMap)
		items = append(items, DeployConfigListItem{
			CicdDeployConfig: row,
			ServerCount:      cnt,
			NodesStatus:      status,
		})
	}
	return items, nil
}

func (s *Service) loadProjectServerMap(ctx context.Context, projectID uint) map[uint]model.Server {
	out := make(map[uint]model.Server)
	if projectID == 0 {
		return out
	}
	var servers []model.Server
	_ = s.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&servers).Error
	for _, srv := range servers {
		out[srv.ID] = srv
	}
	return out
}

func summarizeDeployNodes(dc model.CicdDeployConfig, servers map[uint]model.Server) (int, string) {
	if dc.Status != 1 {
		return 0, "已停用"
	}
	if dc.DeployKind == model.CicdDeployKindContainer {
		cnt := dc.Replicas
		if cnt <= 0 {
			cnt = 1
		}
		return cnt, "启用"
	}
	var ids []uint
	if strings.TrimSpace(dc.ServerIDsJSON) != "" {
		_ = json.Unmarshal([]byte(dc.ServerIDsJSON), &ids)
	}
	if len(ids) == 0 {
		return 0, "未配置节点"
	}
	healthy := 0
	for _, id := range ids {
		srv, ok := servers[id]
		if !ok {
			continue
		}
		errText := ""
		if srv.LastTestError != nil {
			errText = strings.TrimSpace(*srv.LastTestError)
		}
		if srv.Status == 1 && errText == "" {
			healthy++
		}
	}
	if healthy == len(ids) {
		return len(ids), "正常"
	}
	if healthy == 0 {
		return len(ids), "异常"
	}
	return len(ids), "部分异常"
}

func (s *Service) UpsertDeployConfig(ctx context.Context, projectID, serviceID, configID uint, req DeployConfigUpsertRequest) (*DeployConfigUpsertResult, error) {
	if _, err := s.loadService(ctx, projectID, serviceID); err != nil {
		return nil, err
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	serverJSON, _ := json.Marshal(req.ServerIDs)
	var row model.CicdDeployConfig
	if configID > 0 {
		if err := s.db.WithContext(ctx).Where("id = ? AND service_id = ?", configID, serviceID).First(&row).Error; err != nil {
			return nil, constants.ErrNotFound
		}
	}
	row.ServiceID = serviceID
	row.Name = strings.TrimSpace(req.Name)
	row.DeployKind = strings.TrimSpace(req.DeployKind)
	row.Tenv = strings.TrimSpace(req.Tenv)
	if row.Tenv == "" {
		return nil, constants.ErrBadRequestWithMsg("发布环境不能为空")
	}
	dupQ := s.db.WithContext(ctx).Model(&model.CicdDeployConfig{}).
		Where("service_id = ? AND deploy_kind = ? AND tenv = ?", serviceID, strings.TrimSpace(req.DeployKind), row.Tenv)
	if configID > 0 {
		dupQ = dupQ.Where("id <> ?", configID)
	}
	var dupCnt int64
	if err := dupQ.Count(&dupCnt).Error; err != nil {
		return nil, err
	}
	if dupCnt > 0 {
		return nil, constants.ErrBadRequestWithMsg("该应用在此环境下已存在同类型发布配置，请编辑已有配置或选择其他环境")
	}
	row.AuditEnabled = req.AuditEnabled
	if s.enforceProdDeployAudit(ctx, row.Tenv, &row.AuditEnabled) {
		row.AuditEnabled = true
	}
	row.Importance = strings.TrimSpace(req.Importance)
	row.DestPath = strings.TrimSpace(req.DestPath)
	row.ServerIDsJSON = string(serverJSON)
	row.DeployUser = strings.TrimSpace(req.DeployUser)
	if row.DeployUser == "" {
		row.DeployUser = "root"
	}
	row.DeployGroup = strings.TrimSpace(req.DeployGroup)
	if row.DeployGroup == "" {
		row.DeployGroup = row.DeployUser
	}
	row.ArtifactRetainCount = req.ArtifactRetainCount
	if row.ArtifactRetainCount <= 0 {
		row.ArtifactRetainCount = s.resolvedConfig(ctx).DefaultArtifactRetain
	}
	row.RunUser = strings.TrimSpace(req.RunUser)
	if row.RunUser == "" {
		row.RunUser = "app"
	}
	row.StartScriptType = strings.TrimSpace(req.StartScriptType)
	if row.StartScriptType == "" {
		row.StartScriptType = "脚本模板"
	}
	row.CustomScriptContent = req.CustomScriptContent
	row.CleanDeployDir = req.CleanDeployDir
	row.JVMOpts = req.JVMOpts
	row.ServerPort = req.ServerPort
	if row.ServerPort <= 0 {
		row.ServerPort = 8080
	}
	row.DeployMethod = strings.TrimSpace(req.DeployMethod)
	if row.DeployMethod == "" {
		row.DeployMethod = "kubectl"
	}
	row.DeployAction = strings.TrimSpace(req.DeployAction)
	if row.DeployAction == "" {
		row.DeployAction = "服务更新"
	}
	row.DeployConfigType = strings.TrimSpace(req.DeployConfigType)
	if row.DeployConfigType == "" {
		row.DeployConfigType = "使用deployment模板"
	}
	row.DeployConfigTemplate = strings.TrimSpace(req.DeployConfigTemplate)
	if row.DeployConfigTemplate == "" {
		row.DeployConfigTemplate = "基础模板"
	}
	row.K8sNamespace = strings.TrimSpace(req.K8sNamespace)
	row.K8sClusterID = req.K8sClusterID
	row.ImageName = strings.TrimSpace(req.ImageName)
	row.ImageTag = strings.TrimSpace(req.ImageTag)
	row.Replicas = req.Replicas
	if row.Replicas <= 0 {
		row.Replicas = 1
	}
	row.ContainerPort = req.ContainerPort
	if row.ContainerPort <= 0 {
		row.ContainerPort = 8080
	}
	row.DeployStrategy = normalizeDeployStrategy(req.DeployStrategy)
	row.CanaryReplicas = req.CanaryReplicas
	if row.CanaryReplicas <= 0 {
		row.CanaryReplicas = 1
	}
	row.CanaryPercent = req.CanaryPercent
	if row.CanaryPercent <= 0 {
		row.CanaryPercent = 10
	}
	if row.CanaryPercent > 100 {
		row.CanaryPercent = 100
	}
	row.CanaryStepsJSON = strings.TrimSpace(req.CanaryStepsJSON)
	if row.CanaryStepsJSON == "" {
		row.CanaryStepsJSON = "10,50,100"
	}
	row.BlueGreenService = strings.TrimSpace(req.BlueGreenService)
	row.Status = status
	if configID > 0 {
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	}
	result := &DeployConfigUpsertResult{Config: &row}
	if strings.EqualFold(row.DeployKind, model.CicdDeployKindContainer) {
		svc, svcErr := s.loadService(ctx, projectID, serviceID)
		if svcErr != nil {
			slog.Default().With("component", "cicd").Warn("skip jenkins sync after deploy config save: load service failed",
				"service_id", serviceID, "config_id", row.ID, "error", svcErr)
		} else if ci, ciErr := s.requireCiConfig(ctx, projectID, serviceID); ciErr != nil {
			slog.Default().With("component", "cicd").Warn("skip jenkins sync after deploy config save: ci config missing",
				"service_id", serviceID, "config_id", row.ID, "error", ciErr)
		} else if syncResult, syncErr := s.syncJenkinsJob(ctx, svc, ci); syncErr != nil {
			result.JenkinsSyncError = jenkins.HumanizeAPIError(syncErr)
			slog.Default().With("component", "cicd").Warn("jenkins job sync failed after deploy config save",
				"service_id", serviceID, "config_id", row.ID, "error", syncErr)
		} else {
			result.JenkinsSync = syncResult
		}
	}
	return result, nil
}

func (s *Service) DeleteDeployConfig(ctx context.Context, projectID, serviceID, configID uint) error {
	if _, err := s.loadService(ctx, projectID, serviceID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Where("id = ? AND service_id = ?", configID, serviceID).Delete(&model.CicdDeployConfig{}).Error
}
