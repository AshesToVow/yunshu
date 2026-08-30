import type { TreeSelectProps } from "antd";
import type { DepartmentItem } from "../../types/api";
import { stringifyPrettyJSON } from "../../services/alert-mappers";

export function parseUintArrayJSON(raw?: string): number[] {
  const s = raw?.trim();
  if (!s) return [];
  try {
    const v = JSON.parse(s) as unknown;
    if (!Array.isArray(v)) return [];
    return v
      .map((x) => (typeof x === "number" ? x : typeof x === "string" && /^\d+$/.test(x) ? Number(x) : NaN))
      .filter((n) => !Number.isNaN(n));
  } catch {
    return [];
  }
}

export function deptToTreeData(nodes: DepartmentItem[]): TreeSelectProps["treeData"] {
  return nodes.map((n) => ({
    title: n.name,
    value: n.id,
    children: n.children?.length ? deptToTreeData(n.children) : undefined,
  }));
}

export function normalizeCloudExpiryLabelsJSON(raw: string): string | null {
  const s = String(raw || "").trim();
  if (!s) return "{}";
  try {
    const parsed = JSON.parse(s) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return null;
    return stringifyPrettyJSON(parsed, "{}");
  } catch {
    return null;
  }
}
