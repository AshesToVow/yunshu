package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// InspectRunListParams filters inspect run listing.
type InspectRunListParams struct {
	ProjectID uint
	Offset    int
	Limit     int
}

// InspectRepo is implemented by *InspectRepository.
type InspectRepo interface {
	// Plan
	GetPlanByProject(ctx context.Context, projectID uint) (*model.InspectPlan, error)
	GetPlanByID(ctx context.Context, id uint) (*model.InspectPlan, error)
	CreatePlan(ctx context.Context, plan *model.InspectPlan) error
	SavePlan(ctx context.Context, plan *model.InspectPlan) error
	UpdatePlanLastRunAt(ctx context.Context, planID uint, at time.Time) error
	ListEnabledPlans(ctx context.Context) ([]model.InspectPlan, error)
	ListPlansWithRetain(ctx context.Context) ([]model.InspectPlan, error)

	// Datasource helpers (plan defaults / validation)
	FirstEnabledPrometheusDS(ctx context.Context, projectID uint) (*model.AlertDatasource, error)
	GetDatasource(ctx context.Context, id uint) (*model.AlertDatasource, error)

	// Items
	CountItemsByProject(ctx context.Context, projectID uint) (int64, error)
	ListItemsByProject(ctx context.Context, projectID uint) ([]model.InspectItem, error)
	ListGlobalItems(ctx context.Context) ([]model.InspectItem, error)
	ListEnabledItemsByProject(ctx context.Context, projectID uint) ([]model.InspectItem, error)
	ListEnabledGlobalItems(ctx context.Context) ([]model.InspectItem, error)
	GetItem(ctx context.Context, projectID, itemID uint) (*model.InspectItem, error)
	CreateItem(ctx context.Context, item *model.InspectItem) error
	SaveItem(ctx context.Context, item *model.InspectItem) error
	DeleteItem(ctx context.Context, projectID, itemID uint) (int64, error)
	DeleteItemsByProject(ctx context.Context, projectID uint) error
	GetGlobalItemByTypeName(ctx context.Context, typ, name string) (*model.InspectItem, error)
	UpdateItemLinkedRuleID(ctx context.Context, itemID, ruleID uint) error

	// Runs
	ListRuns(ctx context.Context, p InspectRunListParams) ([]model.InspectRun, int64, error)
	ListRunTrends(ctx context.Context, projectID uint, limit int) ([]model.InspectRun, error)
	GetRun(ctx context.Context, projectID, runID uint) (*model.InspectRun, error)
	GetRunByID(ctx context.Context, runID uint) (*model.InspectRun, error)
	CreateRun(ctx context.Context, run *model.InspectRun) error
	SaveRun(ctx context.Context, run *model.InspectRun) error
	UpdateRunFields(ctx context.Context, runID uint, fields map[string]any) error
	MarkRunRunning(ctx context.Context, runID uint, startedAt time.Time) (int64, error)
	ReclaimOrphanRunning(ctx context.Context, staleBefore, finishedAt time.Time, errMsg string) (int64, error)
	ReclaimStaleRunning(ctx context.Context, cutoff, finishedAt time.Time, errMsg string) error
	ListPendingRuns(ctx context.Context, limit int) ([]model.InspectRun, error)
	ListExpiredSuccessRuns(ctx context.Context, projectID uint, cutoff time.Time) ([]model.InspectRun, error)
	DeleteRun(ctx context.Context, run *model.InspectRun) error
	ListLocalStorageRuns(ctx context.Context, projectID uint) ([]model.InspectRun, error)

	// Findings
	PluckPreviousFindingRunID(ctx context.Context, projectID, excludeRunID uint) (uint, error)
	ListFindingsByRun(ctx context.Context, projectID, runID uint) ([]model.InspectFinding, error)
	CreateFindingsBatch(ctx context.Context, rows []model.InspectFinding) error

	// Report templates
	GetReportTemplate(ctx context.Context, projectID, templateID uint) (*model.InspectReportTemplate, error)
	GetGlobalDefaultReportTemplate(ctx context.Context) (*model.InspectReportTemplate, error)
	GetGlobalReportTemplateByCode(ctx context.Context, code string) (*model.InspectReportTemplate, error)
	ListReportTemplates(ctx context.Context, projectID uint) ([]model.InspectReportTemplate, error)
	GetProjectReportTemplate(ctx context.Context, projectID, tid uint) (*model.InspectReportTemplate, error)
	GetCopyableReportTemplate(ctx context.Context, projectID, sourceID uint) (*model.InspectReportTemplate, error)
	CreateReportTemplate(ctx context.Context, row *model.InspectReportTemplate) error
	SaveReportTemplate(ctx context.Context, row *model.InspectReportTemplate) error
	DeleteProjectReportTemplate(ctx context.Context, projectID, tid uint) (int64, error)

	// Alert promote
	GetMonitorRule(ctx context.Context, id uint) (*model.AlertMonitorRule, error)
	SaveMonitorRule(ctx context.Context, rule *model.AlertMonitorRule) error
	CreateMonitorRuleLinkItem(ctx context.Context, rule *model.AlertMonitorRule, itemID uint) error
	ClaimPlanLastRunAt(ctx context.Context, planID uint, expected *time.Time, claimed time.Time) (int64, error)
}

var _ InspectRepo = (*InspectRepository)(nil)
