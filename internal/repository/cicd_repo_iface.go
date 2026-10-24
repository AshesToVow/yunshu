package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// CicdServiceListParams filters cicd_services listing.
type CicdServiceListParams struct {
	ProjectID       uint
	Keyword         string
	ServiceType     string
	IDs             []uint // empty = no id filter; when RestrictIDs, empty means empty result
	RestrictIDs     bool
	Offset          int
	Limit           int
}

// CicdBuildRunListParams filters cicd_build_runs listing.
type CicdBuildRunListParams struct {
	ProjectID   uint
	ServiceID   uint
	ServiceIDs  []uint
	RestrictSvc bool
	Keyword     string
	Offset      int
	Limit       int
}

// CicdReleaseRunListParams filters cicd_release_runs listing (incl. todo scopes).
type CicdReleaseRunListParams struct {
	ProjectID           uint
	ServiceID           uint
	ServiceIDs          []uint
	RestrictSvc         bool
	Status              string
	ReleaseType         string
	Tenv                string
	Keyword             string
	ApproverUserID      *uint
	ApprovalDoneUserID  *uint
	ApprovalMineUserID  *uint
	ExecutionUserID     *uint // pending execution for submitter
	ExecutionDoneUserID *uint
	ExecutionMineUserID *uint
	Offset              int
	Limit               int
}

// CicdAccessGrantListParams filters access grants.
type CicdAccessGrantListParams struct {
	ProjectID uint
	ServiceID uint
	Offset    int
	Limit     int
}

// CicdImageRegistryListParams filters image registries.
type CicdImageRegistryListParams struct {
	Keyword string
	Type    string
	Offset  int
	Limit   int
}

// CicdCleanupPolicyListParams filters cleanup policies.
type CicdCleanupPolicyListParams struct {
	RegistryID *uint
	ProjectID  *uint
	Offset     int
	Limit      int
}

// CicdServiceDeployCount is a grouped deploy-config count.
type CicdServiceDeployCount struct {
	ServiceID uint
	Cnt       int64
}

// CicdNameID is a lightweight id+name row.
type CicdNameID struct {
	ID   uint
	Name string
}

// CicdUserBrief is id/username/nickname for enrichment.
type CicdUserBrief struct {
	ID       uint
	Username string
	Nickname string
}

// CicdServiceBrief is id/name/identifier.
type CicdServiceBrief struct {
	ID         uint
	Name       string
	Identifier string
}

// CicdRepo is implemented by *CicdRepository.
type CicdRepo interface {
	Transaction(ctx context.Context, fn func(CicdRepo) error) error

	// --- Service ---
	ListServices(ctx context.Context, p CicdServiceListParams) ([]model.CicdService, int64, error)
	GetService(ctx context.Context, projectID, serviceID uint) (*model.CicdService, error)
	GetServiceByID(ctx context.Context, serviceID uint) (*model.CicdService, error)
	GetServiceByProjectIdentifier(ctx context.Context, projectID uint, identifier string) (*model.CicdService, error)
	CreateService(ctx context.Context, row *model.CicdService) error
	SaveService(ctx context.Context, row *model.CicdService) error
	DeleteServiceCascade(ctx context.Context, projectID, serviceID uint) error
	UpdateServiceJenkinsJob(ctx context.Context, serviceID uint, jobName string) error
	CountDuplicateServiceIdentifier(ctx context.Context, projectID, excludeID uint, identifier string) (int64, error)
	ListServicesByProject(ctx context.Context, projectID uint) ([]model.CicdService, error)
	ListServicesByIDs(ctx context.Context, ids []uint) ([]model.CicdService, error)
	PluckServiceIDsWithCI(ctx context.Context, serviceIDs []uint) ([]uint, error)
	CountDeployConfigsByServiceIDs(ctx context.Context, serviceIDs []uint) ([]CicdServiceDeployCount, error)
	ListLatestBuildsByServiceIDs(ctx context.Context, serviceIDs []uint) ([]model.CicdBuildRun, error)
	CountCIByService(ctx context.Context, serviceID uint) (int64, error)

	// --- CI / Deploy config ---
	GetCIConfig(ctx context.Context, serviceID uint) (*model.CicdCiConfig, error)
	CreateCIConfig(ctx context.Context, row *model.CicdCiConfig) error
	SaveCIConfig(ctx context.Context, row *model.CicdCiConfig) error
	ListDeployConfigs(ctx context.Context, serviceID uint) ([]model.CicdDeployConfig, error)
	GetDeployConfig(ctx context.Context, serviceID, configID uint) (*model.CicdDeployConfig, error)
	GetDeployConfigByID(ctx context.Context, configID uint) (*model.CicdDeployConfig, error)
	CreateDeployConfig(ctx context.Context, row *model.CicdDeployConfig) error
	SaveDeployConfig(ctx context.Context, row *model.CicdDeployConfig) error
	DeleteDeployConfig(ctx context.Context, serviceID, configID uint) error
	CountDeployConfigDupName(ctx context.Context, serviceID, excludeID uint, name string) (int64, error)
	CountDeployConfigDupKindTenv(ctx context.Context, serviceID, excludeID uint, deployKind, tenv string) (int64, error)
	CountContainerDeploys(ctx context.Context, serviceID uint) (int64, error)
	GetFirstContainerDeploy(ctx context.Context, serviceID uint) (*model.CicdDeployConfig, error)
	GetFirstDeployConfig(ctx context.Context, serviceID uint) (*model.CicdDeployConfig, error)

	// --- Build / Release runs ---
	ListBuildRuns(ctx context.Context, p CicdBuildRunListParams) ([]model.CicdBuildRun, int64, error)
	ListReleaseRuns(ctx context.Context, p CicdReleaseRunListParams) ([]model.CicdReleaseRun, int64, error)
	GetBuildRun(ctx context.Context, projectID, runID uint) (*model.CicdBuildRun, error)
	GetBuildRunByID(ctx context.Context, runID uint) (*model.CicdBuildRun, error)
	GetReleaseRun(ctx context.Context, projectID, runID uint) (*model.CicdReleaseRun, error)
	GetReleaseRunByID(ctx context.Context, runID uint) (*model.CicdReleaseRun, error)
	CreateBuildRun(ctx context.Context, row *model.CicdBuildRun) error
	CreateReleaseRun(ctx context.Context, row *model.CicdReleaseRun) error
	SaveBuildRun(ctx context.Context, row *model.CicdBuildRun) error
	SaveReleaseRun(ctx context.Context, row *model.CicdReleaseRun) error
	UpdateBuildRunFields(ctx context.Context, runID uint, fields map[string]any) error
	UpdateBuildRunFieldsIfStatus(ctx context.Context, runID uint, statuses []string, fields map[string]any) (int64, error)
	UpdateReleaseRunFields(ctx context.Context, runID uint, fields map[string]any) error
	UpdateReleaseRunFieldsIfStatus(ctx context.Context, runID uint, statuses []string, fields map[string]any) (int64, error)
	DeleteBuildRun(ctx context.Context, projectID, runID uint) error
	DeleteReleaseRun(ctx context.Context, projectID, runID uint) error
	ListActiveBuildRuns(ctx context.Context, limit int) ([]model.CicdBuildRun, error)
	ListSuccessfulBuildRunsMissingArtifacts(ctx context.Context, limit int) ([]model.CicdBuildRun, error)
	ListActiveReleaseRuns(ctx context.Context, limit int) ([]model.CicdReleaseRun, error)
	ListStuckPendingExecutionReleases(ctx context.Context, olderThan time.Time, limit int) ([]model.CicdReleaseRun, error)
	FindBuildRunByServiceJenkins(ctx context.Context, serviceID uint, buildNumber int, queueID int64) (*model.CicdBuildRun, error)
	FindReleaseRunByServiceJenkins(ctx context.Context, serviceID uint, buildNumber int, queueID int64) (*model.CicdReleaseRun, error)
	GetServiceByJenkinsJob(ctx context.Context, jobName string) (*model.CicdService, error)
	ListPendingApprovalReleases(ctx context.Context, projectID uint) ([]model.CicdReleaseRun, error)
	ClaimReleaseRunStatus(ctx context.Context, runID uint, fromStatuses []string, fields map[string]any) (int64, error)
	UpdateReleaseRunFieldsIfBuildNumberZero(ctx context.Context, runID uint, fields map[string]any) (int64, error)
	ClaimReleaseRunNoBuildNumber(ctx context.Context, runID uint, fromStatus string, fields map[string]any) (int64, error)
	ListBuildRunsByImageAddress(ctx context.Context, imageAddress string, limit int) ([]model.CicdBuildRun, error)
	LookupLinkedK8sWorkload(ctx context.Context, cicdServiceID uint) (clusterID uint, namespace, kind, name string, err error)

	// --- Stages / Artifacts ---
	GetRunStage(ctx context.Context, runKind string, runID uint, stageType string, stageOrder int) (*model.CicdRunStage, error)
	CreateRunStage(ctx context.Context, row *model.CicdRunStage) error
	UpdateRunStage(ctx context.Context, row *model.CicdRunStage, fields map[string]any) error
	FindArtifact(ctx context.Context, buildRunID uint, artifactType, name, version string) (*model.CicdArtifact, error)
	FindArtifactByStoragePath(ctx context.Context, buildRunID uint, artifactType, storagePath string) (*model.CicdArtifact, error)
	CreateArtifact(ctx context.Context, row *model.CicdArtifact) error
	UpdateArtifact(ctx context.Context, row *model.CicdArtifact, fields map[string]any) error
	ListArtifactsByBuildRun(ctx context.Context, buildRunID uint) ([]model.CicdArtifact, error)
	ListRunStages(ctx context.Context, runKind string, runID uint) ([]model.CicdRunStage, error)

	// --- Approval steps ---
	CreateApprovalSteps(ctx context.Context, steps []model.CicdReleaseApprovalStep) error
	ListApprovalStepsByRun(ctx context.Context, releaseRunID uint) ([]model.CicdReleaseApprovalStep, error)
	ListApprovalStepsByRuns(ctx context.Context, runIDs []uint) ([]model.CicdReleaseApprovalStep, error)
	GetPendingApprovalStep(ctx context.Context, releaseRunID uint) (*model.CicdReleaseApprovalStep, error)
	UpdateApprovalStepFields(ctx context.Context, stepID uint, fields map[string]any) error
	ActivateApprovalStep(ctx context.Context, releaseRunID uint, stageKey string) error
	CountApprovalSteps(ctx context.Context, releaseRunID uint) (int64, error)
	ClaimPendingApprovalStep(ctx context.Context, stepID uint, fields map[string]any) (int64, error)
	ListStalePendingApprovalSteps(ctx context.Context, olderThan time.Time) ([]model.CicdReleaseApprovalStep, error)
	MarkApprovalStepsReminded(ctx context.Context, ids []uint, at time.Time) error

	// --- Access grants ---
	ListAccessGrants(ctx context.Context, p CicdAccessGrantListParams) ([]model.CicdAccessGrant, int64, error)
	UpsertAccessGrant(ctx context.Context, row *model.CicdAccessGrant) error
	DeleteAccessGrant(ctx context.Context, projectID, grantID uint) (int64, error)
	ListAccessGrantsForUser(ctx context.Context, projectID, userID uint) ([]model.CicdAccessGrant, error)
	ListMemberUserIDs(ctx context.Context, projectID uint) ([]model.ProjectMember, error)

	// --- Pipeline templates ---
	ListEnabledPipelineTemplates(ctx context.Context) ([]model.CicdPipelineTemplate, error)
	GetPipelineTemplate(ctx context.Context, id uint) (*model.CicdPipelineTemplate, error)
	CreatePipelineTemplate(ctx context.Context, row *model.CicdPipelineTemplate) error
	SavePipelineTemplate(ctx context.Context, row *model.CicdPipelineTemplate) error
	CountPipelineTemplateByLang(ctx context.Context, languageType string) (int64, error)
	GetPipelineTemplateByLang(ctx context.Context, languageType string) (*model.CicdPipelineTemplate, error)
	GetEnabledPipelineTemplateByLang(ctx context.Context, languageType string) (*model.CicdPipelineTemplate, error)

	// --- Registry / Harbor / Cleanup ---
	GetImageRegistry(ctx context.Context, id uint) (*model.ImageRegistry, error)
	ListImageRegistries(ctx context.Context, p CicdImageRegistryListParams) ([]model.ImageRegistry, int64, error)
	CreateImageRegistry(ctx context.Context, row *model.ImageRegistry) error
	SaveImageRegistry(ctx context.Context, row *model.ImageRegistry) error
	DeleteImageRegistryCascade(ctx context.Context, id uint) error
	ClearDefaultImageRegistries(ctx context.Context) error
	GetDefaultEnabledHarbor(ctx context.Context) (*model.ImageRegistry, error)
	GetProjectRegistryBinding(ctx context.Context, projectID uint) (*model.ProjectRegistryBinding, error)
	CreateProjectRegistryBinding(ctx context.Context, row *model.ProjectRegistryBinding) error
	SaveProjectRegistryBinding(ctx context.Context, row *model.ProjectRegistryBinding) error
	DeleteProjectRegistryBinding(ctx context.Context, projectID uint) (int64, error)
	GetProjectHarborFields(ctx context.Context, projectID uint) (harborURL, harborProject string, err error)
	UpdateProjectHarborFields(ctx context.Context, projectID uint, harborURL, harborProject string) error
	ListCleanupPolicies(ctx context.Context, p CicdCleanupPolicyListParams) ([]model.ImageCleanupPolicy, int64, error)
	GetCleanupPolicy(ctx context.Context, id uint) (*model.ImageCleanupPolicy, error)
	CreateCleanupPolicy(ctx context.Context, row *model.ImageCleanupPolicy) error
	SaveCleanupPolicy(ctx context.Context, row *model.ImageCleanupPolicy) error
	DeleteCleanupPolicy(ctx context.Context, id uint) (int64, error)
	ListEnabledCleanupPolicies(ctx context.Context) ([]model.ImageCleanupPolicy, error)
	UpdateCleanupPolicyFields(ctx context.Context, id uint, fields map[string]any) error

	// --- Enrichment helpers ---
	ListUserGroupsByIDs(ctx context.Context, ids []uint) ([]CicdNameID, error)
	ListUsersByIDs(ctx context.Context, ids []uint) ([]CicdUserBrief, error)
	GetProjectName(ctx context.Context, projectID uint) (string, error)
	GetServiceBrief(ctx context.Context, serviceID uint) (*CicdServiceBrief, error)
	ListServersByProject(ctx context.Context, projectID uint) ([]model.Server, error)

	// --- Cross-domain (verify / change / workflow SLA) ---
	CountFiringAlertsSince(ctx context.Context, projectID uint, since time.Time) (int64, error)
	CountChangeEventsByRelease(ctx context.Context, releaseRunID uint) (int64, error)
	UpdateWorkflowTicketStepFields(ctx context.Context, stepID uint, fields map[string]any) error
	ListStaleWorkflowApprovalSteps(ctx context.Context, olderThan time.Time) ([]model.WorkflowTicketStep, error)
	ListWorkflowApprovalReminderRows(ctx context.Context) ([]CicdWorkflowReminderRow, error)
	ListLegacyApprovalReminderSeedIDs(ctx context.Context) ([]uint, error)
}

// CicdWorkflowReminderRow is a joined workflow step needing SLA reminder evaluation.
type CicdWorkflowReminderRow struct {
	StepID          uint
	TicketID        uint
	StageName       string
	ActivatedAt     time.Time
	LastRemindedAt  *time.Time
	UserGroupID     *uint
	AssigneeUserID  *uint
	RefID           uint
	Title           string
	ProjectID       uint
	SubmitterUserID uint
}

var _ CicdRepo = (*CicdRepository)(nil)
