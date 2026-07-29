import type { PageData } from "../types/api";
import { getData, http } from "./http";
import type { ChangeEventItem } from "./change-events";
import type { AlertEventItem } from "./alerts";
import type { IncidentRelease } from "./service-portrait";

export interface IncidentItem {
  id: number;
  project_id: number;
  service_id?: number | null;
  title: string;
  severity: string;
  status: string;
  summary?: string;
  alert_fingerprint?: string;
  assignee_user_id?: number | null;
  opened_by?: number | null;
  acknowledged_at?: string | null;
  resolved_at?: string | null;
  mtta_seconds?: number | null;
  mttr_seconds?: number | null;
  created_at: string;
}

export interface IncidentTimeline {
  incident: IncidentItem;
  notes: { id: number; incident_id: number; author_id: number; body: string; created_at: string }[];
  changes: ChangeEventItem[];
  alerts: AlertEventItem[];
  releases: IncidentRelease[];
  mtta_seconds?: number | null;
  mttr_seconds?: number | null;
}

export async function listIncidents(projectId: number, params?: { status?: string; severity?: string; page?: number; page_size?: number }) {
  return getData<PageData<IncidentItem>>(http.get(`/projects/${projectId}/incidents`, { params }));
}

export async function openIncident(
  projectId: number,
  body: {
    title: string;
    severity?: string;
    summary?: string;
    service_id?: number;
    alert_fingerprint?: string;
    assignee_user_id?: number;
  },
) {
  return getData<IncidentItem>(http.post(`/projects/${projectId}/incidents`, body));
}

export async function updateIncident(
  projectId: number,
  incidentId: number,
  body: { status?: string; summary?: string; assignee_user_id?: number },
) {
  return getData<IncidentItem>(http.put(`/projects/${projectId}/incidents/${incidentId}`, body));
}

export async function getIncidentTimeline(projectId: number, incidentId: number, windowMinutes = 60) {
  return getData<IncidentTimeline>(
    http.get(`/projects/${projectId}/incidents/${incidentId}/timeline`, { params: { window_minutes: windowMinutes } }),
  );
}
