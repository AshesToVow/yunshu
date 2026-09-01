// @ts-nocheck
import { getData, http } from "./http";
import type { AlertDutyBlockItem } from "./alert-platform";

export interface AlertDutyCalendarItem extends AlertDutyBlockItem {
  overlap?: boolean;
}

export function listDutyCalendar(params: { monitor_rule_id?: number; project_id?: number; from: string; to: string }) {
  return getData<{ list: AlertDutyCalendarItem[] }>(http.get("/alerts/duty-blocks/calendar", { params }));
}

export function validateDutyBlocks(payload: { monitor_rule_id: number; blocks: unknown[] }) {
  return getData<{ ok: boolean; conflicts?: string[] }>(http.post("/alerts/duty-blocks/validate", payload));
}

export function handoffDutyBlock(
  id: number,
  payload: { user_ids_json: string; department_ids_json?: string; extra_emails_json?: string; remark?: string },
) {
  return getData<AlertDutyBlockItem>(http.post(`/alerts/duty-blocks/${id}/handoff`, payload));
}
