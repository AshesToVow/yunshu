package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// DbmgmtRepo is implemented by *DbmgmtRepository.
type DbmgmtRepo interface {
	CreateInstance(ctx context.Context, inst *model.DbInstance) error
	UpdateInstance(ctx context.Context, inst *model.DbInstance) error
	DeleteInstance(ctx context.Context, id uint) error
	GetInstance(ctx context.Context, id uint) (*model.DbInstance, error)
	GetInstanceInProject(ctx context.Context, projectID, id uint) (*model.DbInstance, error)
	ListInstances(ctx context.Context, p DbInstanceListParams) ([]model.DbInstance, int64, error)
	ListAllInstances(ctx context.Context) ([]model.DbInstance, error)
	CountReplicasByPrimary(ctx context.Context, projectID, primaryID uint) (int64, error)

	CreateGrant(ctx context.Context, g *model.DbAccessGrant) error
	UpdateGrant(ctx context.Context, g *model.DbAccessGrant) error
	DeleteGrant(ctx context.Context, id uint) error
	GetGrantInProject(ctx context.Context, projectID, id uint) (*model.DbAccessGrant, error)
	ListGrants(ctx context.Context, projectID, instanceID uint) ([]model.DbAccessGrant, error)
	ListGrantsForPrincipal(ctx context.Context, projectID, instanceID uint, kind, ref string) ([]model.DbAccessGrant, error)

	CreateAccessRequest(ctx context.Context, req *model.DbAccessRequest) error
	UpdateAccessRequest(ctx context.Context, req *model.DbAccessRequest) error
	GetAccessRequestInProject(ctx context.Context, projectID, id uint) (*model.DbAccessRequest, error)
	ListAccessRequests(ctx context.Context, p DbAccessRequestListParams) ([]model.DbAccessRequest, int64, error)
	CreateAccessRequestStep(ctx context.Context, step *model.DbAccessRequestStep) error
	ListAccessRequestSteps(ctx context.Context, requestID uint) ([]model.DbAccessRequestStep, error)
	UpdateAccessRequestStep(ctx context.Context, step *model.DbAccessRequestStep) error

	CreateSqlTicket(ctx context.Context, t *model.DbSqlTicket) error
	UpdateSqlTicket(ctx context.Context, t *model.DbSqlTicket) error
	GetSqlTicketInProject(ctx context.Context, projectID, id uint) (*model.DbSqlTicket, error)
	ListSqlTickets(ctx context.Context, p DbSqlTicketListParams) ([]model.DbSqlTicket, int64, error)
	CreateSqlTicketStep(ctx context.Context, step *model.DbSqlTicketStep) error
	ListSqlTicketSteps(ctx context.Context, ticketID uint) ([]model.DbSqlTicketStep, error)
	UpdateSqlTicketStep(ctx context.Context, step *model.DbSqlTicketStep) error

	CreateSqlExecution(ctx context.Context, ex *model.DbSqlExecution) error
	ListSqlExecutions(ctx context.Context, p DbSqlExecutionListParams) ([]model.DbSqlExecution, int64, error)
	CreateAuditLog(ctx context.Context, log *model.DbAuditLog) error
	ListAuditLogs(ctx context.Context, p DbAuditLogListParams) ([]model.DbAuditLog, int64, error)

	ListApprovalFlowStages(ctx context.Context, projectID uint) ([]model.DbApprovalFlowStage, error)
	UpsertApprovalFlowStage(ctx context.Context, stage *model.DbApprovalFlowStage) error
	DeleteApprovalFlowStagesNotIn(ctx context.Context, projectID uint, keys []string) error
	ListPendingAccessStepsForReminder(ctx context.Context, sla time.Duration) ([]model.DbAccessRequestStep, error)
	ListPendingTicketStepsForReminder(ctx context.Context, sla time.Duration) ([]model.DbSqlTicketStep, error)

	CreateAppUserRequest(ctx context.Context, req *model.DbAppUserRequest) error
	UpdateAppUserRequest(ctx context.Context, req *model.DbAppUserRequest) error
	GetAppUserRequestInProject(ctx context.Context, projectID, id uint) (*model.DbAppUserRequest, error)
	ListAppUserRequests(ctx context.Context, p DbAppUserRequestListParams) ([]model.DbAppUserRequest, int64, error)
	CreateAppUserRequestStep(ctx context.Context, step *model.DbAppUserRequestStep) error
	ListAppUserRequestSteps(ctx context.Context, requestID uint) ([]model.DbAppUserRequestStep, error)
	UpdateAppUserRequestStep(ctx context.Context, step *model.DbAppUserRequestStep) error

	CreateInstanceAccount(ctx context.Context, acc *model.DbInstanceAccount) error
	ListInstanceAccounts(ctx context.Context, projectID, instanceID uint) ([]model.DbInstanceAccount, error)
	GetInstanceAccount(ctx context.Context, projectID, id uint) (*model.DbInstanceAccount, error)
}

var _ DbmgmtRepo = (*DbmgmtRepository)(nil)

// MysqlBackupRepo is implemented by *MysqlBackupRepository.
type MysqlBackupRepo interface {
	CreateInstance(ctx context.Context, inst *model.MysqlBackupInstance) (error)
	UpdateInstance(ctx context.Context, inst *model.MysqlBackupInstance, updatePassword bool) (error)
	DeleteInstance(ctx context.Context, id uint) (error)
	GetInstance(ctx context.Context, id uint) (*model.MysqlBackupInstance, error)
	GetInstanceInProject(ctx context.Context, projectID uint, id uint) (*model.MysqlBackupInstance, error)
	ListInstances(ctx context.Context, p MysqlBackupInstanceListParams) ([]model.MysqlBackupInstance, int64, error)
	CreateJob(ctx context.Context, job *model.MysqlBackupJob) (error)
	UpdateJob(ctx context.Context, job *model.MysqlBackupJob) (error)
	PatchJob(ctx context.Context, jobID uint, fields map[string]any) (error)
	GetJob(ctx context.Context, id uint) (*model.MysqlBackupJob, error)
	GetJobInProject(ctx context.Context, projectID, jobID uint) (*model.MysqlBackupJob, error)
	DeleteJob(ctx context.Context, projectID, jobID uint) error
	ListScheduleEnabledInstances(ctx context.Context) ([]model.MysqlBackupInstance, error)
	TouchLastScheduledAt(ctx context.Context, id uint, at time.Time) (error)
	HasRunningJob(ctx context.Context, instanceID uint) (bool, error)
	FailStaleRunningJobs(ctx context.Context, maxAge time.Duration) (int64, error)
	ListJobs(ctx context.Context, p MysqlBackupJobListParams) ([]model.MysqlBackupJob, int64, error)
}

var _ MysqlBackupRepo = (*MysqlBackupRepository)(nil)

