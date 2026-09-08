package workflow

import (
	"context"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// MigrateLegacyDefinitions 将 dbmgmt/cicd 旧审批流表一次性导入 workflow_definitions。
func MigrateLegacyDefinitions(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := migrateDbmgmtStages(ctx, db); err != nil {
		return err
	}
	return migrateCicdStages(ctx, db)
}

func migrateDbmgmtStages(ctx context.Context, db *gorm.DB) error {
	var legacy []model.DbApprovalFlowStage
	if err := db.WithContext(ctx).Find(&legacy).Error; err != nil {
		return err
	}
	byProject := map[uint][]model.DbApprovalFlowStage{}
	for _, row := range legacy {
		byProject[row.ProjectID] = append(byProject[row.ProjectID], row)
	}
	svc := NewService(repository.NewWorkflowRepository(db), nil, nil, nil)
	for projectID, rows := range byProject {
		key := DefinitionKey{Domain: model.WorkflowDomainDbmgmt, ProjectID: projectID}
		def, _, err := svc.loadDefinition(ctx, key)
		if err != nil {
			return err
		}
		if def != nil {
			continue
		}
		items := make([]StageUpsertItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, StageUpsertItem{
				StageKey: row.StageKey, StageName: row.StageName, SortOrder: row.SortOrder,
				Enabled: row.Enabled, AssigneeRuleType: model.WorkflowAssigneeUserGroup,
				UserGroupID: row.UserGroupID,
			})
		}
		if _, err := svc.UpsertDefinition(ctx, key, DefinitionUpsertRequest{Stages: items}); err != nil {
			return err
		}
	}
	return nil
}

func migrateCicdStages(ctx context.Context, db *gorm.DB) error {
	var legacy []model.CicdApprovalFlowStage
	if err := db.WithContext(ctx).Find(&legacy).Error; err != nil {
		return err
	}
	byProject := map[uint][]model.CicdApprovalFlowStage{}
	for _, row := range legacy {
		byProject[row.ProjectID] = append(byProject[row.ProjectID], row)
	}
	svc := NewService(repository.NewWorkflowRepository(db), nil, nil, nil)
	for projectID, rows := range byProject {
		key := DefinitionKey{Domain: model.WorkflowDomainCicd, ProjectID: projectID}
		def, _, err := svc.loadDefinition(ctx, key)
		if err != nil {
			return err
		}
		if def != nil {
			continue
		}
		items := make([]StageUpsertItem, 0, len(rows))
		for _, row := range rows {
			items = append(items, StageUpsertItem{
				StageKey: row.StageKey, StageName: row.StageName, SortOrder: row.SortOrder,
				Enabled: row.Enabled, AssigneeRuleType: model.WorkflowAssigneeUserGroup,
				UserGroupID: row.UserGroupID,
			})
		}
		if _, err := svc.UpsertDefinition(ctx, key, DefinitionUpsertRequest{Stages: items}); err != nil {
			return err
		}
	}
	return nil
}

// EnabledLegacyDbmgmtStages 供 dbmgmt 初始化审批步骤时读取统一引擎配置。
func EnabledLegacyDbmgmtStages(ctx context.Context, db *gorm.DB, userGroupRepo interfaces.UserGroupRepository, projectID uint) ([]model.DbApprovalFlowStage, error) {
	return EnabledLegacyDbmgmtStagesFrom(ctx, NewService(repository.NewWorkflowRepository(db), userGroupRepo, nil, nil), projectID)
}

// EnabledLegacyDbmgmtStagesFrom 使用已有引擎实例读取 dbmgmt 启用节点。
func EnabledLegacyDbmgmtStagesFrom(ctx context.Context, svc *Service, projectID uint) ([]model.DbApprovalFlowStage, error) {
	if svc == nil {
		return nil, gorm.ErrInvalidDB
	}
	stages, err := svc.EnabledStages(ctx, DefinitionKey{Domain: model.WorkflowDomainDbmgmt, ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	out := make([]model.DbApprovalFlowStage, 0, len(stages))
	for _, st := range stages {
		row := workflowStageToLegacyDbmgmt(st)
		row.ProjectID = projectID
		out = append(out, row)
	}
	return out, nil
}

// EnabledLegacyCicdStages 供 cicd 初始化发布审批步骤。
func EnabledLegacyCicdStages(ctx context.Context, db *gorm.DB, userGroupRepo interfaces.UserGroupRepository, projectID uint) ([]model.CicdApprovalFlowStage, error) {
	return EnabledLegacyCicdStagesFrom(ctx, NewService(repository.NewWorkflowRepository(db), userGroupRepo, nil, nil), projectID)
}

// EnabledLegacyCicdStagesFrom 使用已有引擎实例读取 cicd 启用节点。
func EnabledLegacyCicdStagesFrom(ctx context.Context, svc *Service, projectID uint) ([]model.CicdApprovalFlowStage, error) {
	if svc == nil {
		return nil, gorm.ErrInvalidDB
	}
	stages, err := svc.EnabledStages(ctx, DefinitionKey{Domain: model.WorkflowDomainCicd, ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	out := make([]model.CicdApprovalFlowStage, 0, len(stages))
	for _, st := range stages {
		out = append(out, model.CicdApprovalFlowStage{
			ProjectID: projectID, StageKey: st.StageKey, StageName: st.StageName,
			SortOrder: st.SortOrder, Enabled: st.Enabled, UserGroupID: st.UserGroupID,
		})
	}
	return out, nil
}
