import { getData, http } from "./http";
import type { ChangeEventItem } from "./change-events";

export interface AlertQualityReport {
  window_hours: number;
  from: string;
  to: string;
  project_id?: number;
  total_events: number;
  cur_firing_count?: number;
  notify_fail_rate: number;
  notify_failed: number;
  quality_score: number;
  noise_top: { title: string; severity: string; count: number; fingerprint?: string; alertname?: string }[];
  repeat_fingerprints: { fingerprint: string; title: string; count: number; severity?: string; alertname?: string }[];
  recent_changes_hint?: ChangeEventItem[];
}

export async function getAlertQualityReport(params?: { window_hours?: number; project_id?: number }) {
  return getData<AlertQualityReport>(http.get("/alerts/quality-report", { params }));
}
