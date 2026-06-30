package dictconfig

import (
	"context"
	"strconv"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"

	"gorm.io/gorm"
)

// CicdDictTypes 数据字典中覆盖 cicd.* 的 dict_type。
type CicdDictTypes struct {
	Enabled                string
	JenkinsBaseURL         string
	JenkinsUsername        string
	JenkinsAPIToken        string
	JenkinsJobFolder       string
	JenkinsfileRepo        string
	JenkinsfileBranch      string
	JenkinsfileFront       string
	JenkinsfileBackend     string
	JenkinsfileK8s         string
	SharedLibraryName      string
	GitCredentialID        string
	SSHDeployCredentialID  string
	MinIOCredentialID      string
	HarborCredentialID     string
	HarborURL              string
	HarborProjectGroup     string
	MinIOEndpoint          string
	MinIOBucketFrontend    string
	MinIOBucketBackend     string
	MCBin                  string
	MCAlias                string
	RunSyncIntervalSeconds string
	DefaultWaitMins        string
	DefaultArtifactRetain  string
}

func DefaultCicdDictTypes() CicdDictTypes {
	return CicdDictTypes{
		Enabled:                "cicd_enabled",
		JenkinsBaseURL:         "cicd_jenkins_base_url",
		JenkinsUsername:        "cicd_jenkins_username",
		JenkinsAPIToken:        "cicd_jenkins_api_token",
		JenkinsJobFolder:       "cicd_jenkins_job_folder",
		JenkinsfileRepo:        "cicd_jenkinsfile_repo",
		JenkinsfileBranch:      "cicd_jenkinsfile_branch",
		JenkinsfileFront:       "cicd_jenkinsfile_front",
		JenkinsfileBackend:     "cicd_jenkinsfile_backend",
		JenkinsfileK8s:         "cicd_jenkinsfile_k8s",
		SharedLibraryName:      "cicd_shared_library_name",
		GitCredentialID:        "cicd_git_credential_id",
		SSHDeployCredentialID:  "cicd_ssh_deploy_credential_id",
		MinIOCredentialID:      "cicd_minio_credential_id",
		HarborCredentialID:     "cicd_harbor_credential_id",
		HarborURL:              "cicd_harbor_url",
		HarborProjectGroup:     "cicd_harbor_project_group",
		MinIOEndpoint:          "cicd_minio_endpoint",
		MinIOBucketFrontend:    "cicd_minio_bucket_frontend",
		MinIOBucketBackend:     "cicd_minio_bucket_backend",
		MCBin:                  "cicd_mc_bin",
		MCAlias:                "cicd_mc_alias",
		RunSyncIntervalSeconds: "cicd_run_sync_interval_seconds",
		DefaultWaitMins:        "cicd_default_wait_mins",
		DefaultArtifactRetain:  "cicd_default_artifact_retain_count",
	}
}

// ResolveCicdConfig 以 yamlBase 为底，用已启用的数据字典项覆盖。
func ResolveCicdConfig(ctx context.Context, db *gorm.DB, yamlBase config.CicdConfig, types CicdDictTypes) config.CicdConfig {
	cfg := yamlBase
	if db == nil {
		return cfg
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.Enabled); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.Enabled = bv
		}
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.JenkinsBaseURL); ok {
		cfg.Jenkins.BaseURL = strings.TrimRight(v, "/")
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.JenkinsUsername); ok {
		cfg.Jenkins.Username = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.JenkinsAPIToken); ok {
		cfg.Jenkins.APIToken = v
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.JenkinsJobFolder); ok {
		cfg.Jenkins.JobFolder = strings.Trim(v, "/")
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.JenkinsfileRepo); ok {
		cfg.Jenkinsfile.Repo = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.JenkinsfileBranch); ok {
		cfg.Jenkinsfile.Branch = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.JenkinsfileFront); ok {
		cfg.Jenkinsfile.Front = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.JenkinsfileBackend); ok {
		cfg.Jenkinsfile.Backend = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.JenkinsfileK8s); ok {
		cfg.Jenkinsfile.K8s = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.SharedLibraryName); ok {
		cfg.Jenkinsfile.SharedLib = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.GitCredentialID); ok {
		cfg.Credentials.Git = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.SSHDeployCredentialID); ok {
		cfg.Credentials.SSHDeploy = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.MinIOCredentialID); ok {
		cfg.Credentials.MinIO = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.HarborCredentialID); ok {
		cfg.Credentials.Harbor = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.HarborURL); ok {
		cfg.Harbor.URL = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.HarborProjectGroup); ok {
		cfg.Harbor.ProjectGroup = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.MinIOEndpoint); ok {
		cfg.MinIO.Endpoint = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.MinIOBucketFrontend); ok {
		cfg.MinIO.BucketFrontend = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.MinIOBucketBackend); ok {
		cfg.MinIO.BucketBackend = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.MCBin); ok {
		cfg.MinIO.MCBin = v
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, types.MCAlias); ok {
		cfg.MinIO.MCAlias = v
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.RunSyncIntervalSeconds); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.RunSyncIntervalSeconds = n
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.DefaultWaitMins); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.DefaultWaitMins = n
		}
	}
	if v, ok := fetchEnabledDictValue(ctx, db, types.DefaultArtifactRetain); ok {
		if n, ok2 := parseInt(v); ok2 && n >= 0 {
			cfg.DefaultArtifactRetain = n
		}
	}
	cfg.MinIO.Endpoint = resolveCicdMinIOEndpoint(ctx, db, cfg.MinIO.Endpoint)
	return cfg
}

func resolveCicdMinIOEndpoint(ctx context.Context, db *gorm.DB, configured string) string {
	useSSL := false
	if db != nil {
		if v, ok := fetchEnabledDictValue(ctx, db, "minio_use_ssl"); ok {
			if b, ok2 := parseBoolLoose(v); ok2 {
				useSSL = b
			}
		}
	}
	if ep := strings.TrimSpace(configured); ep != "" {
		return FormatMinioEndpointURL(ep, useSSL)
	}
	if db == nil {
		return ""
	}
	if v, ok := fetchEnabledDictValueNonEmpty(ctx, db, "minio_endpoint"); ok {
		return FormatMinioEndpointURL(v, useSSL)
	}
	return ""
}

// ResolveK8sCredentialID 按环境从字典 cicd_k8s_credential 解析 Jenkins 凭据 ID。
func ResolveK8sCredentialID(ctx context.Context, db *gorm.DB, tenv string) string {
	if db == nil {
		return ""
	}
	tenv = strings.TrimSpace(strings.ToLower(tenv))
	var rows []model.DictEntry
	_ = db.WithContext(ctx).
		Model(&model.DictEntry{}).
		Where("dict_type = ? AND status = 1", "cicd_k8s_credential").
		Order("sort ASC, id ASC").
		Find(&rows).Error
	for _, row := range rows {
		label := strings.TrimSpace(strings.ToLower(row.Label))
		if label == tenv || label == strings.ToUpper(tenv) {
			return strings.TrimSpace(row.Value)
		}
	}
	if len(rows) > 0 {
		return strings.TrimSpace(rows[0].Value)
	}
	switch tenv {
	case "prod":
		return "k8s-prod-config"
	case "test":
		return "k8s-test-config"
	default:
		return "k8s-dev-config"
	}
}

func JenkinsScriptPath(cfg config.CicdConfig, serviceType string, usesK8sPipeline bool) string {
	if usesK8sPipeline {
		if strings.TrimSpace(cfg.Jenkinsfile.K8s) != "" {
			return cfg.Jenkinsfile.K8s
		}
		return "cigroovy.jenkinsfile"
	}
	switch strings.ToLower(strings.TrimSpace(serviceType)) {
	case model.CicdServiceTypeFrontend:
		return cfg.Jenkinsfile.Front
	default:
		return cfg.Jenkinsfile.Backend
	}
}

func MinIOBucketForService(cfg config.CicdConfig, serviceType string) string {
	if strings.EqualFold(serviceType, model.CicdServiceTypeFrontend) {
		return cfg.MinIO.BucketFrontend
	}
	return cfg.MinIO.BucketBackend
}

func ParseArtifactRetain(raw string, fallback int) int {
	if n, ok := parseInt(strings.TrimSpace(raw)); ok && n >= 0 {
		return n
	}
	if fallback >= 0 {
		return fallback
	}
	return 10
}

func FormatWaitMins(cfg config.CicdConfig) string {
	if cfg.DefaultWaitMins > 0 {
		return strconv.Itoa(cfg.DefaultWaitMins)
	}
	return "60"
}
