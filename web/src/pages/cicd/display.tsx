// CI/CD 列表展示映射。
// 由 web/src/pages/cicd-services-page.tsx 原样搬迁（RF-09），映射分支顺序与空值回落（"—" / ""）逐字保留。
import { Tag } from "antd";
import type { UserItem } from "../../types/api";

export function serviceTypeLabel(v: string) {
  if (v === "frontend") return "前端服务";
  if (v === "backend") return "后端服务";
  return "容器化服务";
}

export function buildResultColor(r?: string) {
  if (r === "success") return "success";
  if (r === "failure") return "error";
  if (r === "running") return "processing";
  return "default";
}

export function ownerLabel(username: string | undefined, users: UserItem[]) {
  if (!username) return "—";
  const u = users.find((it) => it.username === username);
  if (!u) return username;
  return u.nickname ? `${u.nickname} (${u.username})` : u.username;
}

export function ownerEmailPreview(username: string | undefined, users: UserItem[]) {
  if (!username) return "";
  const u = users.find((it) => it.username === username);
  return String(u?.email || "").trim();
}

/** 节点状态标签：状态文案由后端下发中文，前端只做配色 */
export function nodesStatusTag(status?: string) {
  const s = status || "—";
  const color =
    s === "正常" || s === "启用"
      ? "success"
      : s === "部分异常"
        ? "warning"
        : s === "异常" || s === "已停用"
          ? "error"
          : "default";
  return <Tag color={color}>{s}</Tag>;
}
