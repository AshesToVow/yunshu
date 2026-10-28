import { getData, http } from "./http";
import type { AlertDutyBlockItem } from "./alert-platform";

export interface AlertDutyCalendarItem extends AlertDutyBlockItem {
  overlap?: boolean;
}

export interface AlertDutyRotationItem {
  id: number;
  monitor_rule_id: number;
  name: string;
  participants_json?: string;
  department_ids_json?: string;
  extra_emails_json?: string;
  anchor_at: string;
  period_minutes: number;
  enabled: boolean;
  remark?: string;
  created_at: string;
  updated_at: string;
}

export interface AlertDutyRotationSlot {
  rotation_id: number;
  name: string;
  period_minutes: number;
  starts_at: string;
  ends_at: string;
  participants: number[];
}

export interface AlertDutyOnCallUser {
  id: number;
  username: string;
  nickname: string;
  email: string;
  phone: string;
  source: "manual" | "rotation";
  rotation_id?: number;
}

export interface AlertDutyOnCallResult {
  at: string;
  users: AlertDutyOnCallUser[];
  extra_emails: string[];
  blocks: AlertDutyBlockItem[];
  rotations: AlertDutyRotationSlot[];
}

export interface AlertDutyRotationPayload {
  monitor_rule_id: number;
  name: string;
  participants_json: string;
  department_ids_json?: string;
  extra_emails_json?: string;
  anchor_at: string;
  period_minutes: number;
  enabled?: boolean;
  remark?: string;
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

export function listDutyRotations(params: {
  monitor_rule_id?: number;
  project_id?: number;
  enabled?: boolean;
  page?: number;
  page_size?: number;
}) {
  return getData<{ list?: AlertDutyRotationItem[]; items?: AlertDutyRotationItem[]; total: number }>(
    http.get("/alerts/duty-rotations", { params }),
  ).then((payload) => ({ list: payload.list ?? payload.items ?? [], total: payload.total ?? 0 }));
}

export function createDutyRotation(payload: AlertDutyRotationPayload) {
  return getData<AlertDutyRotationItem>(http.post("/alerts/duty-rotations", payload));
}

export function updateDutyRotation(id: number, payload: AlertDutyRotationPayload) {
  return getData<AlertDutyRotationItem>(http.put(`/alerts/duty-rotations/${id}`, payload));
}

export function deleteDutyRotation(id: number) {
  return getData<void>(http.delete(`/alerts/duty-rotations/${id}`));
}

export function getDutyOnCall(params: { monitor_rule_id: number; at?: string }) {
  return getData<AlertDutyOnCallResult>(http.get("/alerts/duty/on-call", { params }));
}
