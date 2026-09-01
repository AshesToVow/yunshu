// CI/CD 发布主机选项与已选主机回显。
// 由 web/src/pages/cicd-services-page.tsx 原样搬迁（RF-09），保持列表接口顺序与静默错误策略不变。
import { getProjectServerDetail, type ServerItem } from "../../services/projects";

export function parseServerIds(json?: string): number[] {
  if (!json) return [];
  try {
    const parsed = JSON.parse(json) as unknown;
    return Array.isArray(parsed) ? parsed.map((v) => Number(v)).filter((v) => Number.isFinite(v) && v > 0) : [];
  } catch {
    return [];
  }
}

export function serverOptionLabel(s: Pick<ServerItem, "name" | "host">) {
  return `${s.name} (${s.host})`;
}

export async function mergeServersWithSelected(
  projectId: number,
  list: ServerItem[],
  selectedIds: number[],
): Promise<{ servers: ServerItem[]; unresolvedIds: number[] }> {
  const byId = new Map<number, ServerItem>();
  for (const s of list) {
    byId.set(Number(s.id), s);
  }
  const missing = selectedIds.filter((id) => !byId.has(id));
  if (missing.length === 0) {
    return { servers: list, unresolvedIds: [] };
  }
  // 探测请求不弹全局 toast，由调用方汇总成「发布主机」场景提示，避免重复弹窗
  const details = await Promise.all(
    missing.map((id) =>
      getProjectServerDetail(projectId, id, { silentErrorToast: true }).catch(() => null),
    ),
  );
  for (const d of details) {
    if (d) {
      byId.set(Number(d.id), d);
    }
  }
  // 保持列表接口顺序，缺失的已选主机追加在末尾，便于回显名称
  const merged = [...list];
  const unresolvedIds: number[] = [];
  for (const id of missing) {
    const s = byId.get(id);
    if (s && !merged.some((it) => Number(it.id) === id)) {
      merged.push(s);
    } else if (!s) {
      unresolvedIds.push(id);
    }
  }
  return { servers: merged, unresolvedIds };
}
