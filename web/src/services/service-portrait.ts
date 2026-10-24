import type { PageData } from "../types/api";
import { getData, http } from "./http";
import type { ChangeEventItem } from "./change-events";
import type { ServiceCatalogItem } from "./service-catalog";

export interface PortraitEntryPoint {
  kind: string;
  label: string;
  path: string;
  hint?: string;
}

export interface PortraitCicdSummary {
  cicd_service_id: number;
  identifier: string;
  name: string;
  last_release_id?: number | null;
  last_status?: string;
  last_title?: string;
  last_at?: string;
}

export interface PortraitLogAnomalyBrief {
  id: number;
  anomaly_type: string;
  title: string;
  severity: string;
  detected_at: string;
}

export interface PortraitLogSummary {
  open_anomaly_count: number;
  pattern_count: number;
  recent_anomalies: PortraitLogAnomalyBrief[];
}

export interface ServicePortrait {
  service: ServiceCatalogItem;
  recent_changes: ChangeEventItem[];
  entry_points: PortraitEntryPoint[];
  cicd_summary?: PortraitCicdSummary | null;
  log_summary?: PortraitLogSummary | null;
  health?: {
    score: number;
    grade: string;
    checked_at: string;
    factors: { key: string; label: string; score: number; max: number; detail: string; deduct: number }[];
  } | null;
}

export async function getServicePortrait(projectId: number, catalogId: number) {
  return getData<ServicePortrait>(http.get(`/projects/${projectId}/service-catalog/${catalogId}/portrait`));
}

export type { PageData };
