/** CD 工单展示共用：状态、类型标签与参数解析 */

export function cicdReleaseStatusTagColor(s: string) {
  const map: Record<string, string> = {
    success: "success",
    failure: "error",
    running: "processing",
    pending: "processing",
    pending_approval: "warning",
    pending_execution: "processing",
    cancelled: "default",
    aborted: "warning",
    rejected: "error",
    mine_pending: "warning",
    mine_done: "success",
  };
  return map[s] || "default";
}

export function cicdReleaseStatusLabel(s: string) {
  const label: Record<string, string> = {
    success: "执行成功",
    failure: "执行失败",
    running: "执行中",
    pending: "排队中",
    pending_approval: "待审批",
    pending_execution: "待执行",
    cancelled: "已取消",
    aborted: "已中止",
    rejected: "已驳回",
    mine_pending: "待审批",
    mine_done: "已审批",
  };
  return label[s] || s;
}

const CICD_RELEASE_TERMINAL_STATUSES = new Set([
  "success",
  "failure",
  "aborted",
  "cancelled",
  "rejected",
]);

/** 待办列表状态：终态用工单真实结果；进行中再按 mine_status 区分「待我/我已」 */
export function cicdTodoStatusLabel(
  row: { status: string; mine_status?: string },
  tab: "pending_approval" | "pending_execution" = "pending_approval",
) {
  if (CICD_RELEASE_TERMINAL_STATUSES.has(row.status)) {
    return cicdReleaseStatusLabel(row.status);
  }
  if (row.mine_status === "mine_done") {
    return tab === "pending_execution" ? "已执行" : "已审批";
  }
  if (row.mine_status === "mine_pending") {
    return tab === "pending_execution" ? "待执行" : "待审批";
  }
  return cicdReleaseStatusLabel(row.status);
}

export function cicdTodoStatusTagColor(row: { status: string; mine_status?: string }) {
  if (CICD_RELEASE_TERMINAL_STATUSES.has(row.status)) {
    return cicdReleaseStatusTagColor(row.status);
  }
  if (row.mine_status === "mine_done") return cicdReleaseStatusTagColor("mine_done");
  if (row.mine_status === "mine_pending") return cicdReleaseStatusTagColor("mine_pending");
  return cicdReleaseStatusTagColor(row.status);
}

export function cicdReleaseTypeLabel(releaseType: string) {
  const labels: Record<string, string> = {
    frontend_online: "服务上线",
    frontend_rollback: "服务回滚",
    backend_initial: "服务初次部署",
    backend_update: "服务更新",
    service_online: "服务上线",
    pod_update: "POD更新",
    container_rollback: "回滚",
    artifact_deploy: "制品发布",
  };
  return labels[releaseType] ?? releaseType;
}

export function cicdReleaseTypeTagColor(releaseType: string) {
  if (releaseType === "frontend_rollback") return "orange";
  if (releaseType === "backend_initial") return "purple";
  if (releaseType === "pod_update") return "green";
  if (releaseType === "container_rollback") return "orange";
  return "blue";
}

export function cicdReleaseKindLabel(kind: string) {
  if (kind === "container") return "容器化";
  return "常规 SSH";
}

export type ReleaseParams = {
  artifact?: string;
  publishMode?: string;
  operation?: string;
  branch?: string;
  tenv?: string;
  deployStrategy?: string;
};

export function parseReleaseParams(paramsJson?: string): ReleaseParams {
  if (!paramsJson) return {};
  try {
    const obj = JSON.parse(paramsJson) as Record<string, string>;
    return {
      artifact: obj.selectedVersion,
      publishMode: obj.publishMode,
      operation: obj.deployAction,
      branch: obj.branch,
      tenv: obj.Tenv || obj.tenv,
      deployStrategy: obj.deployStrategy,
    };
  } catch {
    return {};
  }
}

export function formatReleaseDuration(start?: string, end?: string) {
  if (!start || !end) return "—";
  const ms = new Date(end).getTime() - new Date(start).getTime();
  if (!Number.isFinite(ms) || ms < 0) return "—";
  const sec = Math.floor(ms / 1000);
  if (sec < 60) return `${sec} 秒`;
  const min = Math.floor(sec / 60);
  const rem = sec % 60;
  return rem ? `${min} 分 ${rem} 秒` : `${min} 分`;
}

export function handlerDisplayName(h: { username?: string; nickname?: string }) {
  const nick = h.nickname?.trim();
  const user = h.username?.trim();
  if (nick && user) return `${nick}（${user}）`;
  return nick || user || "—";
}
