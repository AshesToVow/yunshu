/** 审批中心共享文案（待办 / 工单列表 / 流程配置） */

export const WORKFLOW_DOMAIN_OPTIONS = [
  { label: "全部域", value: "" },
  { label: "数据库", value: "dbmgmt" },
  { label: "发布", value: "cicd" },
  { label: "AI", value: "ai" },
  { label: "故障", value: "incident" },
  { label: "变更", value: "ops" },
] as const;

export function workflowDomainLabel(domain: string) {
  switch (domain) {
    case "dbmgmt":
      return "数据库";
    case "cicd":
      return "发布";
    case "ai":
      return "AI";
    case "incident":
      return "故障";
    case "ops":
      return "变更";
    default:
      return domain || "—";
  }
}

export function workflowDomainColor(domain: string) {
  switch (domain) {
    case "dbmgmt":
      return "blue";
    case "cicd":
      return "purple";
    case "ai":
      return "cyan";
    case "incident":
      return "red";
    case "ops":
      return "orange";
    default:
      return "default";
  }
}

export function workflowTicketTypeLabel(ticketType: string) {
  switch (ticketType) {
    case "sql_ticket":
      return "SQL 工单";
    case "access_request":
      return "权限申请";
    case "app_user_apply":
      return "应用用户";
    case "release":
      return "发布审批";
    case "tool_approval":
      return "高危操作";
    case "change":
      return "变更单";
    case "incident":
      return "故障单";
    case "default":
      return "默认流程";
    default:
      return ticketType || "工单";
  }
}

export function workflowStatusLabel(status: string) {
  switch (status) {
    case "pending":
      return "待审批";
    case "approved":
      return "已通过";
    case "rejected":
      return "已驳回";
    case "cancelled":
      return "已取消";
    case "closed":
      return "已关闭";
    default:
      return status || "—";
  }
}

export function workflowStatusColor(status: string) {
  switch (status) {
    case "pending":
      return "processing";
    case "approved":
      return "success";
    case "rejected":
      return "error";
    case "cancelled":
    case "closed":
      return "default";
    default:
      return "default";
  }
}

export function workflowStepStatusLabel(status: string) {
  switch (status) {
    case "pending":
      return "待审";
    case "approved":
      return "通过";
    case "rejected":
      return "驳回";
    case "skipped":
      return "跳过";
    default:
      return status || "—";
  }
}

/** 业务详情深链（成熟中心：待办只做审批，执行去详情） */
export function workflowBusinessDeepLink(row: {
  ref_type?: string;
  ref_id?: number;
  project_id?: number;
  deep_link?: string;
}) {
  if (row.deep_link) return row.deep_link;
  const projectId = row.project_id ?? 0;
  const refId = row.ref_id ?? 0;
  switch (row.ref_type) {
    case "db_sql_ticket":
      return `/dbmgmt/workflow/tickets/${refId}?project=${projectId}`;
    case "db_access_request":
      return `/dbmgmt/apply/query?project=${projectId}&highlight=${refId}`;
    case "db_app_user_request":
      return `/dbmgmt/apply/app-user?project=${projectId}&highlight=${refId}`;
    case "cicd_release_run":
      return `/cicd/release-records?project=${projectId}&release=${refId}`;
    case "ai_tool_approval":
      return `/ai/approvals?highlight=${refId}`;
    case "alert_event":
      return `/alert-events?highlight=${refId}`;
    default:
      return projectId ? `/workflow/inbox?project=${projectId}` : "/workflow/inbox";
  }
}
