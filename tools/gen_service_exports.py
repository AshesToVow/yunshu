#!/usr/bin/env python3
"""Append missing type/func aliases to internal/service/exports.go from subpackages."""
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
EXPORTS = ROOT / "internal/service/exports.go"

ALIASES = {
    "system": [
        "UserDetailResponse", "RoleItem", "PermissionItem", "UserSubject",
        "UserCreateRequest", "UserUpdateRequest", "UserAssignRolesRequest", "UserListQuery",
        "DepartmentCreateRequest", "DepartmentUpdateRequest", "DepartmentDetailResponse",
        "ApplyRegisterRequest", "ReviewRequest",
        "PermissionCreateRequest", "PermissionUpdateRequest", "PermissionListQuery",
        "RoleCreateRequest", "RoleUpdateRequest", "RoleListQuery",
        "UserGroupCreateRequest", "UserGroupUpdateRequest", "UserGroupListQuery",
        "UserGroupAssignUsersRequest", "UserGroupItem", "UserGroupDetailResponse",
        "PolicyGrantRequest", "PolicyItemResponse",
        "OperationLogListQuery", "LoginLogListQuery",
        "MenuCreatePayload", "MenuUpdatePayload", "MenuBatchStatusPayload",
        "DictEntryListQuery", "DictEntryCreateRequest", "DictEntryUpdateRequest", "DictEntryOption",
        "NewRoleItem", "NewPermissionItem", "NewUserDetailResponse", "NewUserGroupItem",
        "SyncUserRoles", "ReplaceRoleCode", "RemoveRolePolicies",
        "ReplacePermissionResource", "RemovePermissionPolicies",
    ],
    "k8s": [
        "K8sAccessRankReadonly", "K8sAccessRankReadonlyExec", "K8sAccessRankAdmin",
        "IsK8sReadAPIPath", "IsK8sNginxRestartRoute", "RequiredK8sAccessRank",
        "K8sScopeRouteKey", "BuildK8sScopeMappings",
        "RbacListQuery", "CronJobTriggerRequest", "JobRerunRequest",
        "ClusterKeywordQuery", "ClusterNameQuery", "ClusterNamespaceKeywordQuery",
        "ClusterManifestApplyRequest", "ClusterNamespaceNameQuery",
    ],
    "project": [
        "ProjectItem", "ProjectListQuery", "ProjectCreateRequest", "ProjectUpdateRequest",
        "ServerItem", "ServerListQuery", "ServerUpsertRequest", "ServerDetailItem",
        "ServerExecRequest", "ServerExecResult",
        "ServerGroupItem", "ServerGroupUpsertRequest", "ServerGroupTreeQuery",
        "CloudAccountItem", "CloudAccountListQuery", "CloudAccountUpsertRequest",
        "CloudSyncRequest", "CloudSyncResult", "CloudServerActionRequest", "CloudServerActionResult",
        "BatchServerTestRequest", "BatchServerTestResult",
        "ServerSyncRequest", "ServerSyncResult", "ServerTestRequest", "ServerTestResult",
        "LogSourceItem", "LogSourceListQuery", "LogSourceUpsertRequest",
        "LogStreamQuery", "LogExportQuery", "RemoteLogFileQuery", "RemoteLogUnitQuery",
        "LogAgentRegisterRequest", "LogAgentPublicRegisterRequest",
        "ProjectMemberItem", "ProjectMemberAddRequest", "ProjectMemberUpdateRequest",
        "CloudExpiryRuleListQuery", "CloudExpiryRuleUpsertRequest",
        "ValidateCloudExpiryCronSpec",
        "ServiceItem", "ServiceListQuery", "ServiceUpsertRequest",
    ],
    "logplatform": [
        "AgentBootstrapRequest", "AgentBootstrapResult",
        "AgentRuntimeSource", "AgentRuntimeConfigResult",
        "AgentBatchHeartbeatRefreshRequest", "AgentBatchHeartbeatRefreshResult",
        "AgentDiscoveryItem", "AgentDiscoveryReportRequest", "AgentDiscoveryReportResult",
        "AgentDiscoveryListQuery", "AgentDiscoveryListItem",
        "LogAgentRegisterRequest", "LogAgentPublicRegisterRequest",
        "LogAgentRegisterResult", "LogAgentHealthReportRequest",
        "LogAgentStatusResult", "LogAgentListQuery", "LogAgentListItem",
        "AgentLogEvent", "BuildLogStreamKey", "AgentLogBroker", "MaxLogHistoryPerStream",
        "DiscoveryRootLogType",
    ],
    "alert": [
        "AlertInhibitionRuleListQuery", "AlertInhibitionRuleUpsertRequest",
    ],
}

def main():
    text = EXPORTS.read_text(encoding="utf-8")
    additions = []
    for pkg, names in ALIASES.items():
        block = []
        for n in names:
            if f"{n} =" in text or f"{n}=" in text:
                continue
            # type vs var vs func - heuristic
            if n[0].isupper() and not n.startswith("New") and n not in (
                "ValidateCloudExpiryCronSpec", "BuildLogStreamKey", "UserSubject",
                "SyncUserRoles", "ReplaceRoleCode", "RemoveRolePolicies",
                "ReplacePermissionResource", "RemovePermissionPolicies",
                "IsK8sReadAPIPath", "IsK8sNginxRestartRoute", "RequiredK8sAccessRank",
                "BuildK8sScopeMappings",
            ):
                if n.startswith("K8sAccessRank") or n == "MaxLogHistoryPerStream":
                    block.append(f"\t{n} = {pkg}.{n}")
                else:
                    block.append(f"\t{n} = {pkg}.{n}")
            else:
                block.append(f"\t{n} = {pkg}.{n}")
        if block:
            additions.append(f"\n// --- {pkg} (DTO / helpers) ---\nconst (\n" if pkg == "k8s" and any(x.startswith("K8sAccessRank") for x in names) else f"\n// --- {pkg} (DTO / helpers) ---\ntype (\n")
            # fix k8s constants - they're const not type
    # Simpler: just append typed blocks manually via StrReplace in exports.go

if __name__ == "__main__":
    main()
