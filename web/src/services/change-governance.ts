import type { PageData } from "../types/api";
import { getData, http } from "./http";
import type { ChangeEventItem } from "./change-events";

export interface FreezeWindowItem {
  id: number;
  project_id: number;
  name: string;
  scope: string;
  env?: string;
  starts_at: string;
  ends_at: string;
  reason?: string;
  enabled: boolean;
}

export interface ConflictCheckResult {
  allowed: boolean;
  blocked_by_freeze?: boolean;
  freeze_window_id?: number;
  freeze_name?: string;
  conflict_warning?: boolean;
  conflict_events?: ChangeEventItem[];
  message?: string;
  active_freezes?: FreezeWindowItem[];
}

export async function listFreezeWindows(projectId: number, params?: { page?: number; page_size?: number; enabled?: boolean }) {
  return getData<PageData<FreezeWindowItem>>(http.get(`/projects/${projectId}/change-freeze-windows`, { params }));
}

export async function upsertFreezeWindow(
  projectId: number,
  body: {
    id?: number;
    name: string;
    scope?: string;
    env?: string;
    starts_at: string;
    ends_at: string;
    reason?: string;
    enabled?: boolean;
  },
) {
  return getData<FreezeWindowItem>(http.post(`/projects/${projectId}/change-freeze-windows`, body));
}

export async function deleteFreezeWindow(projectId: number, freezeId: number) {
  return getData<{ message: string }>(http.delete(`/projects/${projectId}/change-freeze-windows/${freezeId}`));
}

export async function conflictCheck(
  projectId: number,
  params?: { source?: string; env?: string; service_id?: number; namespace?: string; action?: string },
) {
  return getData<ConflictCheckResult>(http.get(`/projects/${projectId}/change-events/conflict-check`, { params }));
}
