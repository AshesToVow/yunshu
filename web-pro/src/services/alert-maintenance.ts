// @ts-nocheck
import { getData, http } from "./http";
import { normalizePagedPayload } from "./alert-mappers";

export interface AlertMaintenanceWindowItem {
  id: number;
  name: string;
  matchers_json: string;
  matchers?: Array<{ name: string; value: string; is_regex: boolean }>;
  starts_at: string;
  ends_at: string;
  comment?: string;
  created_by: number;
  project_id: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

function parseMatchers(raw?: string) {
  const s = String(raw || "").trim();
  if (!s) return [];
  try {
    const arr = JSON.parse(s) as unknown[];
    if (!Array.isArray(arr)) return [];
    return arr.map((it) => {
      const o = it as { name?: string; value?: string; is_regex?: boolean };
      return { name: String(o.name ?? ""), value: String(o.value ?? ""), is_regex: Boolean(o.is_regex) };
    });
  } catch {
    return [];
  }
}

export function listMaintenanceWindows(params?: {
  projectId?: number;
  keyword?: string;
  page?: number;
  page_size?: number;
}) {
  return getData<{
    list?: AlertMaintenanceWindowItem[];
    items?: AlertMaintenanceWindowItem[];
    total: number;
    page: number;
    page_size: number;
  }>(http.get("/alerts/maintenance-windows", { params })).then((payload) =>
    normalizePagedPayload(payload, (item) => ({
      ...item,
      matchers: parseMatchers(item.matchers_json),
    })),
  );
}

export function createMaintenanceWindow(payload: {
  name: string;
  matchers_json: string;
  starts_at: string;
  ends_at: string;
  comment?: string;
  project_id?: number;
  enabled?: boolean;
}) {
  return getData<AlertMaintenanceWindowItem>(http.post("/alerts/maintenance-windows", payload));
}

export function updateMaintenanceWindow(
  id: number,
  payload: {
    name: string;
    matchers_json: string;
    starts_at: string;
    ends_at: string;
    comment?: string;
    project_id?: number;
    enabled?: boolean;
  },
) {
  return getData<AlertMaintenanceWindowItem>(http.put(`/alerts/maintenance-windows/${id}`, payload));
}

export function deleteMaintenanceWindow(id: number) {
  return getData<void>(http.delete(`/alerts/maintenance-windows/${id}`));
}
