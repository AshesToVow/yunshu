package cicd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/jenkins"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/pkg/pagination"

	"gorm.io/gorm"
)

type Service struct {
	db            *gorm.DB
	serverRepo    interfaces.ServerRepository
	projectRepo   interfaces.ProjectRepository
	userGroupRepo interfaces.UserGroupRepository
	userRepo      interfaces.UserRepository
	nsEnsurer     K8sNamespaceEnsurer
	mailer        mailer.Sender
	appName       string
	yamlCicd      config.CicdConfig
	syncMu        sync.Mutex
}

func NewService(db *gorm.DB, serverRepo interfaces.ServerRepository, projectRepo interfaces.ProjectRepository, userGroupRepo interfaces.UserGroupRepository, userRepo interfaces.UserRepository, yamlCicd config.CicdConfig, emailSender mailer.Sender, appName string, nsEnsurer K8sNamespaceEnsurer) *Service {
	if yamlCicd.RunSyncIntervalSeconds <= 0 {
		yamlCicd.RunSyncIntervalSeconds = 15
	}
	if yamlCicd.ApprovalSlaHours <= 0 {
		yamlCicd.ApprovalSlaHours = 24
	}
	if yamlCicd.ApprovalReminderIntervalHours <= 0 {
		yamlCicd.ApprovalReminderIntervalHours = 4
	}
	return &Service{
		db:            db,
		serverRepo:    serverRepo,
		projectRepo:   projectRepo,
		userGroupRepo: userGroupRepo,
		userRepo:      userRepo,
		nsEnsurer:     nsEnsurer,
		mailer:        emailSender,
		appName:       strings.TrimSpace(appName),
		yamlCicd:      yamlCicd,
	}
}

func (s *Service) resolvedConfig(ctx context.Context) config.CicdConfig {
	base := s.yamlCicd
	if base.RunSyncIntervalSeconds <= 0 {
		base = config.DefaultCicdConfig()
		base.Jenkins = s.yamlCicd.Jenkins
	}
	return dictconfig.ResolveCicdConfig(ctx, s.db, base, dictconfig.DefaultCicdDictTypes())
}

func (s *Service) jenkinsClient(ctx context.Context) (*jenkins.Client, config.CicdConfig, error) {
	cfg := s.resolvedConfig(ctx)
	if !cfg.Enabled {
		return nil, cfg, constants.ErrBadRequestWithMsg("CI/CD 未启用，请在数据字典配置 cicd_enabled=true")
	}
	if strings.TrimSpace(cfg.Jenkins.BaseURL) == "" {
		return nil, cfg, constants.ErrBadRequestWithMsg("Jenkins 地址未配置，请在数据字典设置 cicd_jenkins_base_url")
	}
	if strings.TrimSpace(cfg.Jenkins.Username) == "" {
		return nil, cfg, constants.ErrBadRequestWithMsg("Jenkins 用户名未配置，请在数据字典设置 cicd_jenkins_username")
	}
	if strings.TrimSpace(cfg.Jenkins.APIToken) == "" {
		return nil, cfg, constants.ErrBadRequestWithMsg("Jenkins API Token 未配置，请在数据字典设置 cicd_jenkins_api_token")
	}
	return jenkins.NewClient(cfg.Jenkins.BaseURL, cfg.Jenkins.Username, cfg.Jenkins.APIToken, cfg.Jenkins.JobFolder), cfg, nil
}

func (s *Service) ensureProject(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		return constants.ErrBadRequestWithMsg("project id required")
	}
	if s.projectRepo == nil {
		return nil
	}
	_, err := s.projectRepo.GetByID(ctx, projectID)
	return err
}

// --- Service CRUD ---

type ServiceListQuery struct {
	ProjectID   uint   `form:"project_id"`
	Keyword     string `form:"keyword"`
	ServiceType string `form:"service_type"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type ServiceItem struct {
	model.CicdService
	HasCiConfig     bool `json:"has_ci_config"`
	DeployConfigCnt int  `json:"deploy_config_count"`
	LastBuildResult string `json:"last_build_result,omitempty"`
	LastBuildAt     *time.Time `json:"last_build_at,omitempty"`
}

type ServiceUpsertRequest struct {
	ProjectID   uint   `json:"project_id"`
	Identifier  string `json:"identifier" binding:"required,max=128"`
	Name        string `json:"name" binding:"required,max=128"`
	ServiceType string `json:"service_type" binding:"required,max=32"`
	Owner       string `json:"owner" binding:"omitempty,max=64"`
	ProductLine string `json:"product_line" binding:"omitempty,max=128"`
	Remark      string `json:"remark" binding:"omitempty,max=512"`
	Status      *int   `json:"status" binding:"omitempty,oneof=0 1"`
	JenkinsJob  string `json:"jenkins_job" binding:"omitempty,max=256"`
}

func (s *Service) ListServices(ctx context.Context, q ServiceListQuery) (*pagination.Result[ServiceItem], error) {
	if err := s.ensureProject(ctx, q.ProjectID); err != nil {
		return nil, err
	}
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	dbq := s.db.WithContext(ctx).Model(&model.CicdService{}).Where("project_id = ?", q.ProjectID)
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		dbq = dbq.Where("name LIKE ? OR identifier LIKE ?", like, like)
	}
	if st := strings.TrimSpace(q.ServiceType); st != "" {
		dbq = dbq.Where("service_type = ?", st)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.CicdService
	if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ServiceItem, 0, len(rows))
	for _, row := range rows {
		item := ServiceItem{CicdService: row}
		var ciCnt int64
		_ = s.db.WithContext(ctx).Model(&model.CicdCiConfig{}).Where("service_id = ?", row.ID).Count(&ciCnt).Error
		item.HasCiConfig = ciCnt > 0
		var deployCnt int64
		_ = s.db.WithContext(ctx).Model(&model.CicdDeployConfig{}).Where("service_id = ? AND status = 1", row.ID).Count(&deployCnt).Error
		item.DeployConfigCnt = int(deployCnt)
		var lastRun model.CicdBuildRun
		if err := s.db.WithContext(ctx).Where("service_id = ?", row.ID).Order("id DESC").First(&lastRun).Error; err == nil {
			item.LastBuildResult = lastRun.BuildResult
			item.LastBuildAt = lastRun.StartedAt
		}
		items = append(items, item)
	}
	return &pagination.Result[ServiceItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetService(ctx context.Context, projectID, serviceID uint) (*ServiceItem, error) {
	svc, err := s.loadService(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	res, err := s.ListServices(ctx, ServiceListQuery{ProjectID: projectID, Page: 1, PageSize: 1})
	if err != nil {
		return nil, err
	}
	item := ServiceItem{CicdService: *svc}
	for _, it := range res.List {
		if it.ID == serviceID {
			item.HasCiConfig = it.HasCiConfig
			item.DeployConfigCnt = it.DeployConfigCnt
			item.LastBuildResult = it.LastBuildResult
			item.LastBuildAt = it.LastBuildAt
			break
		}
	}
	var ciCnt int64
	_ = s.db.WithContext(ctx).Model(&model.CicdCiConfig{}).Where("service_id = ?", serviceID).Count(&ciCnt).Error
	item.HasCiConfig = ciCnt > 0
	return &item, nil
}

func (s *Service) UpsertService(ctx context.Context, serviceID uint, req ServiceUpsertRequest) (*model.CicdService, error) {
	if err := s.ensureProject(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		return nil, constants.ErrBadRequestWithMsg("identifier required")
	}
	serviceType := strings.TrimSpace(req.ServiceType)
	if serviceType == model.CicdServiceTypeFrontend || serviceType == model.CicdServiceTypeBackend || serviceType == model.CicdServiceTypeMicro {
		// ok
	} else if serviceType == "micro" {
		serviceType = model.CicdServiceTypeMicro
	} else {
		return nil, constants.ErrBadRequestWithMsg("service_type must be frontend|backend|microservice")
	}
	status := 1
	if req.Status != nil {
		status = *req.Status
	}
	jenkinsJob := strings.TrimSpace(req.JenkinsJob)
	if jenkinsJob == "" {
		jenkinsJob = fmt.Sprintf("cicd-p%d-%s", req.ProjectID, identifier)
	}
	var row model.CicdService
	if serviceID > 0 {
		if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", serviceID, req.ProjectID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
		}
	} else {
		var exists int64
		if err := s.db.WithContext(ctx).Model(&model.CicdService{}).
			Where("project_id = ? AND identifier = ?", req.ProjectID, identifier).Count(&exists).Error; err != nil {
			return nil, err
		}
		if exists > 0 {
			return nil, constants.ErrBadRequestWithMsg("identifier already exists in project")
		}
	}
	row.ProjectID = req.ProjectID
	row.Identifier = identifier
	row.Name = strings.TrimSpace(req.Name)
	row.ServiceType = serviceType
	row.Owner = strings.TrimSpace(req.Owner)
	row.ProductLine = strings.TrimSpace(req.ProductLine)
	row.Remark = strings.TrimSpace(req.Remark)
	row.Status = status
	row.JenkinsJob = jenkinsJob
	if serviceID > 0 {
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	}
	return &row, nil
}

func (s *Service) DeleteService(ctx context.Context, projectID, serviceID uint) error {
	if _, err := s.loadService(ctx, projectID, serviceID); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("service_id = ?", serviceID).Delete(&model.CicdCiConfig{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_id = ?", serviceID).Delete(&model.CicdDeployConfig{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_id = ?", serviceID).Delete(&model.CicdBuildRun{}).Error; err != nil {
			return err
		}
		if err := tx.Where("service_id = ?", serviceID).Delete(&model.CicdReleaseRun{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND project_id = ?", serviceID, projectID).Delete(&model.CicdService{}).Error
	})
}

func (s *Service) loadService(ctx context.Context, projectID, serviceID uint) (*model.CicdService, error) {
	var row model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", serviceID, projectID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

// --- CI Config ---

type CiConfigUpsertRequest struct {
	GitURL           string `json:"git_url" binding:"required,max=512"`
	RefType          string `json:"ref_type" binding:"omitempty,oneof=branch tag"`
	RefName          string `json:"ref_name" binding:"required,max=128"`
	BuildType        string `json:"build_type" binding:"required,max=32"`
	BuildShell       string `json:"build_shell" binding:"omitempty,max=512"`
	BuildPath        string `json:"build_path" binding:"omitempty,max=256"`
	ProjectName      string `json:"project_name" binding:"omitempty,max=128"`
	Version          string `json:"version" binding:"omitempty,max=64"`
	NodeVersion      string `json:"node_version" binding:"omitempty,max=32"`
	NpmInstallMode   string `json:"npm_install_mode" binding:"omitempty,max=16"`
	CleanNpmCache    bool   `json:"clean_npm_cache"`
	CleanNodeModules bool   `json:"clean_node_modules"`
	JavaToolName     string `json:"java_tool_name" binding:"omitempty,max=64"`
	ServerPort       string `json:"server_port" binding:"omitempty,max=16"`
	PackConfigPaths  string `json:"pack_config_paths" binding:"omitempty,max=512"`
	Description      string `json:"description" binding:"omitempty,max=512"`
}

func (s *Service) FindCiConfig(ctx context.Context, projectID, serviceID uint) (*model.CicdCiConfig, bool, error) {
	if _, err := s.loadService(ctx, projectID, serviceID); err != nil {
		return nil, false, err
	}
	var row model.CicdCiConfig
	if err := s.db.WithContext(ctx).Where("service_id = ?", serviceID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &row, true, nil
}

// CiConfigView GET ci-config 响应（未配置时 configured=false，不返回 404）。
type CiConfigView struct {
	Configured bool               `json:"configured"`
	Config     *model.CicdCiConfig `json:"config,omitempty"`
}

func (s *Service) GetCiConfigView(ctx context.Context, projectID, serviceID uint) (*CiConfigView, error) {
	row, ok, err := s.FindCiConfig(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	return &CiConfigView{Configured: ok, Config: row}, nil
}

func (s *Service) requireCiConfig(ctx context.Context, projectID, serviceID uint) (*model.CicdCiConfig, error) {
	row, ok, err := s.FindCiConfig(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	if !ok || row == nil {
		return nil, constants.ErrBadRequestWithMsg("请先配置 CI 信息")
	}
	return row, nil
}

// CiConfigUpsertResult 保存 CI 配置结果（DB 保存与 Jenkins 同步解耦）。
type CiConfigUpsertResult struct {
	Config           *model.CicdCiConfig `json:"config"`
	JenkinsSync      *JenkinsSyncResult  `json:"jenkins_sync,omitempty"`
	JenkinsSyncError string              `json:"jenkins_sync_error,omitempty"`
}

func (s *Service) UpsertCiConfig(ctx context.Context, projectID, serviceID uint, req CiConfigUpsertRequest) (*CiConfigUpsertResult, error) {
	svc, err := s.loadService(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	refType := strings.TrimSpace(req.RefType)
	if refType == "" {
		refType = model.CicdRefTypeBranch
	}
	var row model.CicdCiConfig
	err = s.db.WithContext(ctx).Where("service_id = ?", serviceID).First(&row).Error
	isNew := err != nil
	row.ServiceID = serviceID
	row.GitURL = strings.TrimSpace(req.GitURL)
	row.RefType = refType
	row.RefName = strings.TrimSpace(req.RefName)
	row.BuildType = strings.TrimSpace(req.BuildType)
	row.BuildShell = strings.TrimSpace(req.BuildShell)
	row.BuildPath = strings.TrimSpace(req.BuildPath)
	row.ProjectName = strings.TrimSpace(req.ProjectName)
	if row.ProjectName == "" {
		row.ProjectName = svc.Identifier
	}
	row.Version = strings.TrimSpace(req.Version)
	row.NodeVersion = strings.TrimSpace(req.NodeVersion)
	if row.NodeVersion == "" {
		row.NodeVersion = model.DefaultNodeToolName
	}
	row.NpmInstallMode = strings.TrimSpace(req.NpmInstallMode)
	row.CleanNpmCache = req.CleanNpmCache
	row.CleanNodeModules = req.CleanNodeModules
	row.JavaToolName = strings.TrimSpace(req.JavaToolName)
	if row.JavaToolName == "" {
		row.JavaToolName = "jdk8"
	}
	row.ServerPort = strings.TrimSpace(req.ServerPort)
	row.PackConfigPaths = strings.TrimSpace(req.PackConfigPaths)
	row.Description = strings.TrimSpace(req.Description)
	if isNew {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return nil, err
		}
	}
	syncResult, err := s.syncJenkinsJob(ctx, svc, &row)
	if err != nil {
		return &CiConfigUpsertResult{
			Config:           &row,
			JenkinsSyncError: jenkins.HumanizeAPIError(err),
		}, nil
	}
	return &CiConfigUpsertResult{Config: &row, JenkinsSync: syncResult}, nil
}

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

// --- Trigger Build / Release ---

type TriggerBuildRequest struct {
	BranchName     string `json:"branch_name" binding:"omitempty,max=128"`
	PublishMode    string `json:"publish_mode" binding:"omitempty,max=32"`
	Tenv           string `json:"tenv" binding:"omitempty,max=16"`
	EmailUser      string `json:"email_user" binding:"omitempty,max=128"`
}

type TriggerReleaseRequest struct {
	DeployConfigID   uint   `json:"deploy_config_id" binding:"required"`
	Title            string `json:"title" binding:"required,max=256"`
	ReleaseOperation string `json:"release_operation" binding:"omitempty,max=32"`
	PublishMode      string `json:"publish_mode" binding:"omitempty,max=32"`
	ArtifactName     string `json:"artifact_name" binding:"omitempty,max=256"`
	ImageAddress     string `json:"image_address" binding:"omitempty,max=512"`
	BuildRunID       uint   `json:"build_run_id" binding:"omitempty"`
	ReleaseType      string `json:"release_type" binding:"omitempty,max=32"`
	EmailUser        string `json:"email_user" binding:"omitempty,max=128"`
}

func (s *Service) TriggerBuild(ctx context.Context, projectID, serviceID uint, req TriggerBuildRequest, builderUserID *uint, builderName string) (*model.CicdBuildRun, error) {
	svc, err := s.loadService(ctx, projectID, serviceID)
	if err != nil {
		return nil, err
	}
	ci, err := s.requireCiConfig(ctx, projectID, serviceID)
	if err != nil {
		return nil, constants.ErrBadRequestWithMsg("请先配置 CI 信息")
	}
	client, cfg, err := s.jenkinsClient(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.syncJenkinsJob(ctx, svc, ci); err != nil {
		return nil, err
	}
	tenv := strings.TrimSpace(req.Tenv)
	if tenv == "" {
		tenv = s.defaultBuildTenv(ctx, serviceID)
	}
	dc := s.primaryContainerDeployConfig(ctx, serviceID)
	if dc == nil {
		dc = s.firstDeployConfig(ctx, serviceID)
	}
	params := BuildJenkinsParams(BuildParamsInput{
		Service:         svc,
		CiConfig:        ci,
		DeployConfig:    dc,
		Cfg:             cfg,
		PublishMode:     model.CicdPublishModeBuildOnly,
		Tenv:            tenv,
		EmailUser:       s.resolveNotifyEmail(ctx, req.EmailUser, svc, builderUserID),
		UsesK8sPipeline: s.serviceUsesK8sPipeline(ctx, svc),
	})
	lastNum, _ := client.GetLastBuildNumber(ctx, svc.JenkinsJob)
	queuePath, err := client.BuildWithParameters(ctx, svc.JenkinsJob, params)
	if err != nil {
		return nil, fmt.Errorf("trigger jenkins build: %w", err)
	}
	now := time.Now()
	run := model.CicdBuildRun{
		ProjectID:      projectID,
		ServiceID:      serviceID,
		BuildNumber:    lastNum + 1,
		BranchName:     params["branchName"],
		PublishMode:    params["publishMode"],
		Tenv:           params["Tenv"],
		BuildResult:    model.CicdRunStatusRunning,
		BuilderUserID:  builderUserID,
		BuilderName:    builderName,
		Version:        ci.Version,
		ParamsJSON:     ParamsJSON(params),
		StartedAt:      &now,
	}
	if buildNum, err := client.ResolveQueueBuildNumber(ctx, queuePath, lastNum, 90*time.Second); err == nil && buildNum > 0 {
		run.BuildNumber = buildNum
		run.JenkinsBuildURL = client.BuildURL(svc.JenkinsJob, buildNum)
	}
	if err := s.db.WithContext(ctx).Create(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *Service) TriggerRelease(ctx context.Context, projectID, serviceID uint, req TriggerReleaseRequest, submitterUserID *uint, submitterName string) (*model.CicdReleaseRun, error) {
	p, err := s.prepareRelease(ctx, projectID, serviceID, req, submitterUserID)
	if err != nil {
		return nil, err
	}
	if p.dc.AuditEnabled {
		return s.createPendingRelease(ctx, projectID, serviceID, p, submitterUserID, submitterName)
	}

	now := time.Now()
	dcID := p.dc.ID
	release := model.CicdReleaseRun{
		ProjectID:       projectID,
		ServiceID:       serviceID,
		DeployConfigID:  &dcID,
		Title:           strings.TrimSpace(p.req.Title),
		ReleaseKind:     p.dc.DeployKind,
		ReleaseType:     p.releaseType,
		Tenv:            p.dc.Tenv,
		Status:          model.CicdRunStatusRunning,
		SubmitterUserID: submitterUserID,
		SubmitterName:   submitterName,
		ImageAddress:    p.imageAddress,
		ArtifactName:    p.artifactName,
		AuditEnabled:    false,
		RequestJSON:     snapshotJSON(p),
		StartedAt:       &now,
	}
	if err := s.db.WithContext(ctx).Create(&release).Error; err != nil {
		return nil, err
	}
	if err := s.executeReleaseRun(ctx, &release, submitterUserID); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", release.ID).First(&release).Error; err != nil {
		return nil, err
	}
	return &release, nil
}

func (s *Service) resolveDestIPs(ctx context.Context, projectID uint, dc *model.CicdDeployConfig) (string, error) {
	if dc == nil || strings.EqualFold(dc.DeployKind, model.CicdDeployKindContainer) {
		return "", nil
	}
	ids, err := ParseServerIDs(dc.ServerIDsJSON)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", nil
	}
	if s.serverRepo == nil {
		return "", nil
	}
	var hosts []string
	for _, id := range ids {
		srv, err := s.serverRepo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		if srv.ProjectID != projectID {
			continue
		}
		host := strings.TrimSpace(srv.Host)
		if host == "" {
			continue
		}
		if srv.Port > 0 && srv.Port != 22 {
			host = host + ":" + strconv.Itoa(srv.Port)
		}
		hosts = append(hosts, host)
	}
	return strings.Join(hosts, ","), nil
}

// --- Build / Release Records ---

type BuildRunListQuery struct {
	ProjectID uint   `form:"project_id"`
	ServiceID uint   `form:"service_id"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type BuildRunItem struct {
	model.CicdBuildRun
	ServiceName     string `json:"service_name"`
	ServiceIdentifier string `json:"service_identifier"`
}

func (s *Service) ListBuildRuns(ctx context.Context, q BuildRunListQuery) (*pagination.Result[BuildRunItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	dbq := s.db.WithContext(ctx).Model(&model.CicdBuildRun{})
	if q.ProjectID > 0 {
		dbq = dbq.Where("project_id = ?", q.ProjectID)
	}
	if q.ServiceID > 0 {
		dbq = dbq.Where("service_id = ?", q.ServiceID)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		dbq = dbq.Where("builder_name LIKE ? OR branch_name LIKE ?", like, like)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.CicdBuildRun
	if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	s.enrichBuildRunPackagePaths(ctx, rows)
	svcNames := s.loadServiceNameMap(ctx, rows)
	items := make([]BuildRunItem, 0, len(rows))
	for _, row := range rows {
		item := BuildRunItem{CicdBuildRun: row}
		if meta, ok := svcNames[row.ServiceID]; ok {
			item.ServiceName = meta.Name
			item.ServiceIdentifier = meta.Identifier
		}
		items = append(items, item)
	}
	return &pagination.Result[BuildRunItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

type ReleaseRunListQuery struct {
	ProjectID      uint   `form:"project_id"`
	ServiceID      uint   `form:"service_id"`
	Status         string `form:"status"`
	ReleaseType    string `form:"release_type"`
	Tenv           string `form:"tenv"`
	Keyword        string `form:"keyword"`
	Mine           bool   `form:"mine"`
	MineScope           string `form:"mine_scope"` // pending | done | all（与 mine 联用）
	Page                int    `form:"page"`
	PageSize            int    `form:"page_size"`
	ApproverUserID      *uint  // 内部：待审核
	ExecutorUserID      *uint  // 内部：待执行（提交人）
	ApprovalDoneUserID  *uint  // 内部：我已审批
	ExecutionDoneUserID *uint  // 内部：我已执行
	ApprovalMineUserID  *uint  // 内部：审批待办全部
	ExecutionMineUserID *uint  // 内部：执行待办全部
	MineTab             string `form:"-"` // approval | execution（mine 待办列表）
	MineViewerUserID    *uint  `form:"-"`
}

type ReleaseRunItem struct {
	model.CicdReleaseRun
	ServiceName       string `json:"service_name"`
	ServiceIdentifier string `json:"service_identifier"`
	ProjectName       string `json:"project_name"`
	CurrentStageName  string `json:"current_stage_name,omitempty"`
	MineStatus        string `json:"mine_status,omitempty"` // mine_pending | mine_done（待办列表按当前用户视角）
}

func (s *Service) ListReleaseRuns(ctx context.Context, q ReleaseRunListQuery) (*pagination.Result[ReleaseRunItem], error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	if q.ProjectID > 0 && strings.TrimSpace(q.Status) == model.CicdRunStatusPendingApproval {
		_ = s.backfillPendingReleaseSteps(ctx, q.ProjectID)
	}
	dbq := s.db.WithContext(ctx).Model(&model.CicdReleaseRun{})
	if q.ProjectID > 0 {
		dbq = dbq.Where("project_id = ?", q.ProjectID)
	}
	if q.ServiceID > 0 {
		dbq = dbq.Where("service_id = ?", q.ServiceID)
	}
	if st := strings.TrimSpace(q.Status); st != "" {
		dbq = dbq.Where("status = ?", st)
	}
	if rt := strings.TrimSpace(q.ReleaseType); rt != "" {
		dbq = dbq.Where("release_type = ?", rt)
	}
	if env := strings.TrimSpace(q.Tenv); env != "" {
		dbq = dbq.Where("tenv = ?", env)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		dbq = dbq.Where("title LIKE ? OR submitter_name LIKE ?", like, like)
	}
	if q.ApproverUserID != nil && *q.ApproverUserID > 0 {
		dbq = s.filterReleaseRunsForApprover(dbq, *q.ApproverUserID)
	}
	if q.ApprovalDoneUserID != nil && *q.ApprovalDoneUserID > 0 {
		dbq = s.filterReleaseRunsApprovalDone(dbq, *q.ApprovalDoneUserID)
	}
	if q.ApprovalMineUserID != nil && *q.ApprovalMineUserID > 0 {
		dbq = s.filterReleaseRunsApprovalMine(dbq, *q.ApprovalMineUserID)
	}
	if q.ExecutorUserID != nil && *q.ExecutorUserID > 0 {
		dbq = dbq.Where("status = ?", model.CicdRunStatusPendingExecution).
			Where("submitter_user_id = ?", *q.ExecutorUserID)
	}
	if q.ExecutionDoneUserID != nil && *q.ExecutionDoneUserID > 0 {
		dbq = s.filterReleaseRunsExecutionDone(dbq, *q.ExecutionDoneUserID)
	}
	if q.ExecutionMineUserID != nil && *q.ExecutionMineUserID > 0 {
		dbq = s.filterReleaseRunsExecutionMine(dbq, *q.ExecutionMineUserID)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.CicdReleaseRun
	if err := dbq.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	svcMap := make(map[uint]model.CicdService)
	for _, row := range rows {
		svcMap[row.ServiceID] = model.CicdService{}
	}
	ids := make([]uint, 0, len(svcMap))
	for id := range svcMap {
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		var svcs []model.CicdService
		_ = s.db.WithContext(ctx).Where("id IN ?", ids).Find(&svcs).Error
		for _, svc := range svcs {
			svcMap[svc.ID] = svc
		}
	}
	items := make([]ReleaseRunItem, 0, len(rows))
	projectName := ""
	if q.ProjectID > 0 {
		var proj model.Project
		if err := s.db.WithContext(ctx).Select("name").Where("id = ?", q.ProjectID).First(&proj).Error; err == nil {
			projectName = proj.Name
		}
	}
	for _, row := range rows {
		item := ReleaseRunItem{CicdReleaseRun: row, ProjectName: projectName}
		if svc, ok := svcMap[row.ServiceID]; ok {
			item.ServiceName = svc.Name
			item.ServiceIdentifier = svc.Identifier
		}
		if row.CurrentStageKey != "" {
			item.CurrentStageName = stageNameByKey(row.CurrentStageKey)
		}
		items = append(items, item)
	}
	if q.MineViewerUserID != nil && *q.MineViewerUserID > 0 {
		s.enrichReleaseRunMineStatus(ctx, items, *q.MineViewerUserID, strings.TrimSpace(q.MineTab))
	}
	return &pagination.Result[ReleaseRunItem]{List: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetBuildRun(ctx context.Context, projectID, runID uint) (*BuildRunItem, error) {
	var row model.CicdBuildRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&row).Error; err != nil {
		return nil, constants.ErrNotFound
	}
	item := BuildRunItem{CicdBuildRun: row}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", row.ServiceID).First(&svc).Error; err == nil {
		item.ServiceName = svc.Name
		item.ServiceIdentifier = svc.Identifier
	}
	return &item, nil
}

func (s *Service) GetBuildRunLog(ctx context.Context, projectID, runID uint) (string, error) {
	row, err := s.GetBuildRun(ctx, projectID, runID)
	if err != nil {
		return "", err
	}
	if row.BuildNumber <= 0 {
		return "", constants.ErrBadRequestWithMsg("构建编号尚未就绪")
	}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", row.ServiceID).First(&svc).Error; err != nil {
		return "", err
	}
	client, _, err := s.jenkinsClient(ctx)
	if err != nil {
		return "", err
	}
	return client.GetConsoleLog(ctx, svc.JenkinsJob, row.BuildNumber)
}

func (s *Service) GetReleaseRunLog(ctx context.Context, projectID, runID uint) (string, error) {
	var row model.CicdReleaseRun
	if err := s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).First(&row).Error; err != nil {
		return "", constants.ErrNotFound
	}
	if row.JenkinsBuildNumber <= 0 {
		return "", nil
	}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", row.ServiceID).First(&svc).Error; err != nil {
		return "", err
	}
	client, _, err := s.jenkinsClient(ctx)
	if err != nil {
		return "", err
	}
	return client.GetConsoleLog(ctx, svc.JenkinsJob, row.JenkinsBuildNumber)
}

func (s *Service) DeleteBuildRun(ctx context.Context, projectID, runID uint) error {
	return s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).Delete(&model.CicdBuildRun{}).Error
}

func (s *Service) DeleteReleaseRun(ctx context.Context, projectID, runID uint) error {
	return s.db.WithContext(ctx).Where("id = ? AND project_id = ?", runID, projectID).Delete(&model.CicdReleaseRun{}).Error
}

type serviceMeta struct {
	Name       string
	Identifier string
}

func (s *Service) loadServiceNameMap(ctx context.Context, runs []model.CicdBuildRun) map[uint]serviceMeta {
	out := make(map[uint]serviceMeta)
	if len(runs) == 0 {
		return out
	}
	ids := make([]uint, 0, len(runs))
	seen := make(map[uint]struct{})
	for _, r := range runs {
		if _, ok := seen[r.ServiceID]; ok {
			continue
		}
		seen[r.ServiceID] = struct{}{}
		ids = append(ids, r.ServiceID)
	}
	var svcs []model.CicdService
	_ = s.db.WithContext(ctx).Where("id IN ?", ids).Find(&svcs).Error
	for _, svc := range svcs {
		out[svc.ID] = serviceMeta{Name: svc.Name, Identifier: svc.Identifier}
	}
	return out
}

// RunSyncWorker 后台同步 Jenkins 构建状态。
func (s *Service) RunSyncWorker(ctx context.Context) {
	interval := time.Duration(s.resolvedConfig(ctx).RunSyncIntervalSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncPendingRuns(context.Background())
		}
	}
}

func (s *Service) syncPendingRuns(ctx context.Context) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	client, _, err := s.jenkinsClient(ctx)
	if err != nil {
		return
	}
	var buildRuns []model.CicdBuildRun
	_ = s.db.WithContext(ctx).Where("build_result IN ?", []string{model.CicdRunStatusRunning, model.CicdRunStatusPending}).Limit(50).Find(&buildRuns).Error
	for _, run := range buildRuns {
		s.syncOneBuildRun(ctx, client, run)
	}
	var releaseRuns []model.CicdReleaseRun
	_ = s.db.WithContext(ctx).Where("status IN ?", []string{model.CicdRunStatusRunning, model.CicdRunStatusPending}).Limit(50).Find(&releaseRuns).Error
	for _, run := range releaseRuns {
		s.syncOneReleaseRun(ctx, client, run)
	}
	var backfillBuilds []model.CicdBuildRun
	_ = s.db.WithContext(ctx).
		Where("build_result = ? AND (package_path = '' OR package_path IS NULL)", model.CicdRunStatusSuccess).
		Order("id DESC").
		Limit(30).
		Find(&backfillBuilds).Error
	for _, run := range backfillBuilds {
		s.backfillBuildArtifacts(ctx, client, run)
	}
	s.syncApprovalReminders(ctx)
}

func (s *Service) enrichBuildRunPackagePaths(ctx context.Context, rows []model.CicdBuildRun) {
	client, _, err := s.jenkinsClient(ctx)
	if err != nil {
		return
	}
	for i := range rows {
		if rows[i].BuildResult != model.CicdRunStatusSuccess {
			continue
		}
		if strings.TrimSpace(rows[i].PackagePath) != "" && strings.TrimSpace(rows[i].ImageAddress) != "" {
			continue
		}
		s.backfillBuildArtifacts(ctx, client, rows[i])
		var updated model.CicdBuildRun
		if err := s.db.WithContext(ctx).Select("package_path", "image_address").Where("id = ?", rows[i].ID).First(&updated).Error; err == nil {
			rows[i].PackagePath = updated.PackagePath
			rows[i].ImageAddress = updated.ImageAddress
		}
	}
}

func (s *Service) backfillBuildArtifacts(ctx context.Context, client *jenkins.Client, run model.CicdBuildRun) {
	if run.BuildNumber <= 0 {
		return
	}
	needPackage := strings.TrimSpace(run.PackagePath) == ""
	needImage := strings.TrimSpace(run.ImageAddress) == ""
	if !needPackage && !needImage {
		return
	}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", run.ServiceID).First(&svc).Error; err != nil {
		return
	}
	var ci model.CicdCiConfig
	_ = s.db.WithContext(ctx).Where("service_id = ?", run.ServiceID).First(&ci).Error
	jobName := resolveJenkinsJobName(&svc)
	if jobName == "" {
		return
	}
	logText, err := client.GetConsoleLog(ctx, jobName, run.BuildNumber)
	if err != nil {
		return
	}
	artifacts := s.resolveBuildArtifactsFromLog(ctx, svc, ci, logText)
	updates := map[string]any{}
	if needPackage && strings.TrimSpace(artifacts.PackagePath) != "" {
		updates["package_path"] = artifacts.PackagePath
	}
	if needImage && strings.TrimSpace(artifacts.ImageAddress) != "" {
		updates["image_address"] = artifacts.ImageAddress
	}
	if len(updates) == 0 {
		return
	}
	_ = s.db.WithContext(ctx).Model(&model.CicdBuildRun{}).Where("id = ?", run.ID).Updates(updates).Error
}

func (s *Service) syncOneBuildRun(ctx context.Context, client *jenkins.Client, run model.CicdBuildRun) {
	if run.BuildNumber <= 0 {
		return
	}
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", run.ServiceID).First(&svc).Error; err != nil {
		return
	}
	var ci model.CicdCiConfig
	_ = s.db.WithContext(ctx).Where("service_id = ?", run.ServiceID).First(&ci).Error
	jobName := resolveJenkinsJobName(&svc)
	if jobName == "" {
		return
	}
	info, err := client.GetBuild(ctx, jobName, run.BuildNumber)
	if err != nil {
		return
	}
	status := jenkins.MapResultToStatus(info.Result, info.Building)
	updates := map[string]any{
		"jenkins_build_url": info.URL,
		"updated_at":        time.Now(),
	}
	if strings.TrimSpace(run.PackagePath) == "" || strings.TrimSpace(run.ImageAddress) == "" {
		if logText, err := client.GetConsoleLog(ctx, jobName, run.BuildNumber); err == nil {
			artifacts := s.resolveBuildArtifactsFromLog(ctx, svc, ci, logText)
			if strings.TrimSpace(run.PackagePath) == "" && strings.TrimSpace(artifacts.PackagePath) != "" {
				updates["package_path"] = artifacts.PackagePath
			}
			if strings.TrimSpace(run.ImageAddress) == "" && strings.TrimSpace(artifacts.ImageAddress) != "" {
				updates["image_address"] = artifacts.ImageAddress
			}
		}
	}
	if status == model.CicdRunStatusRunning {
		_ = s.db.WithContext(ctx).Model(&model.CicdBuildRun{}).Where("id = ?", run.ID).Updates(updates).Error
		return
	}
	now := time.Now()
	updates["build_result"] = status
	if run.FinishedAt == nil {
		updates["finished_at"] = now
	}
	_ = s.db.WithContext(ctx).Model(&model.CicdBuildRun{}).Where("id = ?", run.ID).Updates(updates).Error
}

// releaseStuckTimeout 发布工单在 running 且构建号仍未落库时，允许的最长补偿窗口。
// 超过后仍拿不到构建号，则判定为触发/回填异常并置为 failure，避免永久卡死。
const releaseStuckTimeout = 30 * time.Minute

func (s *Service) syncOneReleaseRun(ctx context.Context, client *jenkins.Client, run model.CicdReleaseRun) {
	var svc model.CicdService
	if err := s.db.WithContext(ctx).Where("id = ?", run.ServiceID).First(&svc).Error; err != nil {
		return
	}
	// 构建号未落库（如触发后请求上下文被取消）：先尝试用 queue_url 补偿解析。
	if run.JenkinsBuildNumber <= 0 {
		if !s.recoverReleaseBuildNumber(ctx, client, &svc, &run) {
			return
		}
	}
	info, err := client.GetBuild(ctx, svc.JenkinsJob, run.JenkinsBuildNumber)
	if err != nil {
		return
	}
	status := jenkins.MapResultToStatus(info.Result, info.Building)
	if status == model.CicdRunStatusRunning {
		return
	}
	now := time.Now()
	updates := map[string]any{
		"status":            status,
		"jenkins_build_url": info.URL,
		"updated_at":        now,
	}
	if run.FinishedAt == nil {
		updates["finished_at"] = now
	}
	_ = s.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).Where("id = ?", run.ID).Updates(updates).Error
}

// recoverReleaseBuildNumber 针对构建号未落库的 running 工单做补偿：
// 依据存下的 queue_url 解析 Jenkins 已分配的构建号并回填 run（含内存副本）。
// 解析成功返回 true；仍未分配则返回 false（下轮再试）；
// 若已超过 releaseStuckTimeout 仍无构建号，则置为 failure 以避免永久卡死。
func (s *Service) recoverReleaseBuildNumber(ctx context.Context, client *jenkins.Client, svc *model.CicdService, run *model.CicdReleaseRun) bool {
	queueURL := strings.TrimSpace(run.JenkinsQueueURL)
	if queueURL != "" {
		if buildNum, err := client.QueueBuildNumber(ctx, queueURL); err == nil && buildNum > 0 {
			updates := map[string]any{
				"jenkins_build_number": buildNum,
				"jenkins_build_url":    client.BuildURL(svc.JenkinsJob, buildNum),
			}
			if err := s.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).
				Where("id = ? AND jenkins_build_number = 0", run.ID).
				Updates(updates).Error; err == nil {
				run.JenkinsBuildNumber = buildNum
				return true
			}
		}
	}
	// 触发时机参考 started_at（无则回退 created_at）；超窗仍无构建号判定为失败。
	ref := run.StartedAt
	if ref == nil {
		ref = &run.CreatedAt
	}
	if ref != nil && time.Since(*ref) > releaseStuckTimeout {
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(&model.CicdReleaseRun{}).
			Where("id = ? AND jenkins_build_number = 0 AND status = ?", run.ID, model.CicdRunStatusRunning).
			Updates(map[string]any{
				"status":      model.CicdRunStatusFailure,
				"finished_at": now,
				"updated_at":  now,
			}).Error
	}
	return false
}
