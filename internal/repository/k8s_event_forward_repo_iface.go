package repository

import (
	"context"

	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
)

type K8sEventForwardRuleListFilter struct {
	Page     int
	PageSize int
}

type K8sForwardedEventListFilter struct {
	Page      int
	PageSize  int
	Status    string
	ClusterID string
}

type K8sEventForwardRepo interface {
	ListRules(ctx context.Context, f K8sEventForwardRuleListFilter) (*pagination.Result[model.K8sEventForwardRule], error)
	GetRule(ctx context.Context, id uint) (*model.K8sEventForwardRule, error)
	CreateRule(ctx context.Context, rule *model.K8sEventForwardRule) error
	SaveRule(ctx context.Context, rule *model.K8sEventForwardRule) error
	DeleteRule(ctx context.Context, id uint) (int64, error)
	GetSettings(ctx context.Context) (model.K8sEventForwardSetting, error)
	SaveSettings(ctx context.Context, st *model.K8sEventForwardSetting) error
	EnsureDefaultSettings(ctx context.Context, defaults model.K8sEventForwardSetting) error

	SaveForwardedEvent(ctx context.Context, ev *model.K8sForwardedEvent) error
	ListUnprocessedEvents(ctx context.Context, limit int) ([]model.K8sForwardedEvent, error)
	ClaimUnprocessedEvents(ctx context.Context, limit int) ([]model.K8sForwardedEvent, error)
	MarkEventProcessed(ctx context.Context, id int64, processed bool) error
	MarkEventDead(ctx context.Context, id int64, lastErr string) error
	MarkEventFailed(ctx context.Context, id int64, lastErr string) error
	IncrementEventAttempts(ctx context.Context, id int64) error
	ListForwardedEvents(ctx context.Context, f K8sForwardedEventListFilter) (*pagination.Result[model.K8sForwardedEvent], error)
	RequeueDeadEvent(ctx context.Context, id int64) error
	ListEnabledRules(ctx context.Context) ([]model.K8sEventForwardRule, error)
	HasEnabledRules(ctx context.Context) (bool, error)
	ListEnabledClusterIDs(ctx context.Context) ([]uint, error)
	GetClusterName(ctx context.Context, id uint) string
	GetClusterOwningProjectID(ctx context.Context, id uint) uint
}
