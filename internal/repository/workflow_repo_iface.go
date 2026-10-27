package repository

import (
	"context"
	"time"

	"yunshu/internal/model"
)

// WorkflowTicketListParams filters workflow ticket listing.
type WorkflowTicketListParams struct {
	Domain     string
	TicketType string
	ProjectID  *uint
	Status     string
	// VisibleUserID 非空时按可见性过滤：提交人 / 成员项目 / 步骤参与人。
	VisibleUserID    *uint
	MemberProjectIDs []uint
	Offset           int
	Limit            int
}

// WorkflowPendingFilter filters cross-domain pending steps.
type WorkflowPendingFilter struct {
	Domains           []string
	ProjectID         *uint
	MineScope         string
	UserID            uint
	IsSuper           bool
	CanPlatformReview bool
	Offset            int
	Limit             int
}

// WorkflowPendingRow is a joined pending/todo row.
type WorkflowPendingRow struct {
	WorkflowTicketID uint
	StepID           uint
	Domain           string
	TicketType       string
	ProjectID        uint
	Title            string
	Status           string
	CurrentStageName string
	SubmitterUserID  uint
	RefType          string
	RefID            uint
	ActivatedAt      *time.Time
	CreatedAt        time.Time
	StepStatus       string
	ReviewerUserID   *uint
}

// WorkflowRepo is implemented by *WorkflowRepository.
type WorkflowRepo interface {
	LoadDefinition(ctx context.Context, domain string, projectID uint, ticketType string) (*model.WorkflowDefinition, []model.WorkflowStage, error)
	GetDefinitionByID(ctx context.Context, id uint) (*model.WorkflowDefinition, error)
	CreateDefinition(ctx context.Context, def *model.WorkflowDefinition) error
	GetStageByKey(ctx context.Context, definitionID uint, stageKey string) (*model.WorkflowStage, error)
	GetStageByKeyUnscoped(ctx context.Context, definitionID uint, stageKey string) (*model.WorkflowStage, error)
	CreateStage(ctx context.Context, stage *model.WorkflowStage) error
	UpdateStageFields(ctx context.Context, id uint, fields map[string]any) error
	RestoreStage(ctx context.Context, id uint, fields map[string]any) error
	DeleteStagesNotIn(ctx context.Context, definitionID uint, keys []string) error

	CreateTicket(ctx context.Context, ticket *model.WorkflowTicket) error
	GetTicket(ctx context.Context, id uint) (*model.WorkflowTicket, error)
	ListTickets(ctx context.Context, p WorkflowTicketListParams) ([]model.WorkflowTicket, int64, error)
	UpdateTicketFields(ctx context.Context, id uint, fields map[string]any) error
	UpdateTicketFieldsIfStatus(ctx context.Context, id uint, expectedStatus string, fields map[string]any) (int64, error)
	UpdateTicketFieldsIfActive(ctx context.Context, id uint, fields map[string]any) (int64, error)
	GetTicketByRef(ctx context.Context, refType string, refID uint, ticketType string) (*model.WorkflowTicket, error)
	IsTicketParticipant(ctx context.Context, ticketID, userID uint) (bool, error)

	CreateStep(ctx context.Context, step *model.WorkflowTicketStep) error
	GetStep(ctx context.Context, ticketID, stepID uint) (*model.WorkflowTicketStep, error)
	ListStepsByTicket(ctx context.Context, ticketID uint) ([]model.WorkflowTicketStep, error)
	ListStepsByTicketIDs(ctx context.Context, ticketIDs []uint) ([]model.WorkflowTicketStep, error)
	GetActiveStep(ctx context.Context, ticketID uint) (*model.WorkflowTicketStep, error)
	ClaimStepReview(ctx context.Context, stepID, ticketID uint, fields map[string]any) (int64, error)
	ActivateStep(ctx context.Context, stepID uint, activatedAt time.Time) error
	ActivateStepIfTicketPending(ctx context.Context, stepID, ticketID uint, activatedAt time.Time) (int64, error)
	SkipPendingSteps(ctx context.Context, ticketID uint, at time.Time) error

	ListPending(ctx context.Context, f WorkflowPendingFilter) ([]WorkflowPendingRow, int64, error)

	Transaction(ctx context.Context, fn func(WorkflowRepo) error) error
}

var _ WorkflowRepo = (*WorkflowRepository)(nil)
