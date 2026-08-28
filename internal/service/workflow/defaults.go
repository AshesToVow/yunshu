package workflow

import (
	"yunshu/internal/model"
)

// DefaultDbmgmtStages dbmgmt 域默认审批节点骨架。
func DefaultDbmgmtStages() []StageItem {
	return []StageItem{
		{StageKey: model.DbApprovalStageDBALead, StageName: "DBA 负责人", SortOrder: 1, AssigneeRuleType: model.WorkflowAssigneeUserGroup},
		{StageKey: model.DbApprovalStageSecurityLead, StageName: "安全负责人", SortOrder: 2, AssigneeRuleType: model.WorkflowAssigneeUserGroup},
		{StageKey: model.DbApprovalStageOpsLead, StageName: "运维负责人", SortOrder: 3, AssigneeRuleType: model.WorkflowAssigneeUserGroup},
	}
}

// DefaultCicdStages cicd 域默认审批节点骨架。
func DefaultCicdStages() []StageItem {
	return []StageItem{
		{StageKey: model.CicdApprovalStageTestLead, StageName: "测试负责人", SortOrder: 1, AssigneeRuleType: model.WorkflowAssigneeUserGroup},
		{StageKey: model.CicdApprovalStageRDLead, StageName: "研发负责人", SortOrder: 2, AssigneeRuleType: model.WorkflowAssigneeUserGroup},
		{StageKey: model.CicdApprovalStageProductLead, StageName: "项目/产品负责人", SortOrder: 3, AssigneeRuleType: model.WorkflowAssigneeUserGroup},
		{StageKey: model.CicdApprovalStageOpsLead, StageName: "运维负责人", SortOrder: 4, AssigneeRuleType: model.WorkflowAssigneeUserGroup},
	}
}

// DefaultIncidentStages 告警/故障单默认节点（排班派单 + 运维负责人组）。
func DefaultIncidentStages() []StageItem {
	return []StageItem{
		{StageKey: "oncall_duty", StageName: "当班处理人", SortOrder: 1, AssigneeRuleType: model.WorkflowAssigneeDuty},
		{StageKey: model.DbApprovalStageOpsLead, StageName: "运维负责人", SortOrder: 2, AssigneeRuleType: model.WorkflowAssigneeUserGroup},
	}
}

func stageItemsToUpsert(items []StageItem) []StageUpsertItem {
	out := make([]StageUpsertItem, 0, len(items))
	for _, it := range items {
		out = append(out, StageUpsertItem{
			StageKey: it.StageKey, StageName: it.StageName, SortOrder: it.SortOrder,
			Enabled: it.Enabled, AssigneeRuleType: it.AssigneeRuleType,
			UserGroupID: it.UserGroupID, DutyMonitorRuleID: it.DutyMonitorRuleID,
		})
	}
	return out
}

func workflowStageToLegacyDbmgmt(st model.WorkflowStage) model.DbApprovalFlowStage {
	return model.DbApprovalFlowStage{
		ProjectID: 0, StageKey: st.StageKey, StageName: st.StageName,
		SortOrder: st.SortOrder, Enabled: st.Enabled, UserGroupID: st.UserGroupID,
	}
}
