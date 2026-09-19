package repository

import (
	"context"

	"yunshu/internal/model"
)

// ClusterLogRepo is implemented by *ClusterLogRepository (rules + agents).
type ClusterLogRepo interface {
	ListRules(ctx context.Context, projectID, clusterID uint) ([]model.ClusterLogRule, error)
	GetRuleByIDInProject(ctx context.Context, projectID, ruleID uint) (*model.ClusterLogRule, error)
	CreateRule(ctx context.Context, row *model.ClusterLogRule) error
	SaveRule(ctx context.Context, row *model.ClusterLogRule) error
	DeleteRuleByIDInProject(ctx context.Context, projectID, ruleID uint) (rowsAffected int64, err error)

	ListAgents(ctx context.Context, projectID uint) ([]model.ClusterLogAgent, error)
	GetAgent(ctx context.Context, projectID, clusterID uint) (*model.ClusterLogAgent, error)
	CreateAgent(ctx context.Context, row *model.ClusterLogAgent) error
	SaveAgent(ctx context.Context, row *model.ClusterLogAgent) error
}

var _ ClusterLogRepo = (*ClusterLogRepository)(nil)
