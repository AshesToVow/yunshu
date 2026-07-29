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

export interface ServicePortrait {
  service: ServiceCatalogItem;
  recent_changes: ChangeEventItem[];
  entry_points: PortraitEntryPoint[];
  cicd_summary?: PortraitCicdSummary | null;
  health?: {
    score: number;
    grade: string;
    checked_at: string;
    factors: { key: string; label: string; score: number; max: number; detail: string; deduct: number }[];
  } | null;
}

export interface IncidentRelease {
  id: number;
  service_id: number;
  title: string;
  status: string;
  tenv: string;
  started_at: string;
}

export interface IncidentContext {
  project_id: number;
  window_minutes: number;
  from: string;
  to: string;
  changes: ChangeEventItem[];
  releases: IncidentRelease[];
}

export async function getServicePortrait(projectId: number, catalogId: number) {
  return getData<ServicePortrait>(http.get(`/projects/${projectId}/service-catalog/${catalogId}/portrait`));
}

export async function getIncidentContext(projectId: number, windowMinutes = 30) {
  return getData<IncidentContext>(
    http.get(`/projects/${projectId}/incident-context`, { params: { window_minutes: windowMinutes } }),
  );
}

export type { PageData };
