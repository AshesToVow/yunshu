/** dbmgmt 工单/申请状态中文映射（前后端统一展示） */

export function ticketStatusLabel(s?: string) {
  const map: Record<string, string> = {
    draft: "草稿",
    pending_approval: "待审批",
    pending_execution: "待执行",
    executing: "执行中",
    success: "执行成功",
    failed: "执行失败",
    rejected: "已驳回",
    approved: "已审批",
    closed: "已关闭",
  };
  return map[s ?? ""] ?? s ?? "—";
}

export function accessRequestStatusLabel(s?: string) {
  const map: Record<string, string> = {
    pending: "待审批",
    approved: "已通过",
    rejected: "已驳回",
    closed: "已关闭",
  };
  return map[s ?? ""] ?? s ?? "—";
}

export function envLabel(env?: string) {
  const map: Record<string, string> = {
    prod: "生产",
    staging: "预发",
    test: "测试",
    dev: "开发",
  };
  return map[env ?? ""] ?? env ?? "—";
}
