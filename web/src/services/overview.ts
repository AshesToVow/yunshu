import { getData, http } from "./http";

export interface OverviewResponse {
  users_count: number;
  clusters_count: number;
  pending_registrations_count: number;
  servers_count: number;
  pod_normal_count: number;
  pod_abnormal_count: number;
  pod_cluster_errors: number;
  event_total_count: number;
  event_warning_count: number;
  event_cluster_errors: number;
  alert_firing_count: number;
  alert_events_today_count: number;
  loggie_agents_online_count: number;
  loggie_agents_offline_count: number;
}

export interface OverviewProjectLaunchSeries {
  project_id: number;
  project_name: string;
  data: number[];
  color?: string;
}

export interface OverviewProjectLaunchesResponse {
  days: string[];
  series: OverviewProjectLaunchSeries[];
}

export interface OverviewReleaseByPersonItem {
  person: string;
  count: number;
}

export interface OverviewReleaseByPersonResponse {
  items: OverviewReleaseByPersonItem[];
}

export interface OverviewLabelCount {
  label: string;
  count: number;
}

export interface OverviewScreenAlert {
  id: number;
  fingerprint: string;
  alertname: string;
  severity: string;
  cluster: string;
  project_id: number;
  summary: string;
  starts_at: string;
}

export interface OverviewScreenRelease {
  id: number;
  project_id: number;
  project_name: string;
  title: string;
  status: string;
  tenv: string;
  submitter_name: string;
  finished_at?: string;
  created_at: string;
}

export interface OverviewScreenChange {
  id: number;
  project_id: number;
  source: string;
  action: string;
  risk_level: string;
  status: string;
  summary: string;
  started_at: string;
}

export interface OverviewScreenLoggie {
  id: number;
  project_id: number;
  server_id: number;
  health_status: string;
  last_error: string;
  last_seen_at?: string;
}

export interface OverviewScreenResponse {
  generated_at: string;
  kpi: {
    users_count: number;
    clusters_count: number;
    servers_count: number;
    servers_enabled: number;
    servers_disabled: number;
    pending_registrations_count: number;
    pod_normal_count: number;
    pod_abnormal_count: number;
    event_warning_count: number;
    alert_firing_count: number;
    alert_events_today_count: number;
    loggie_agents_online_count: number;
    loggie_agents_offline_count: number;
    ai_investigations_today: number;
    ai_investigations_open: number;
  };
  health: {
    pod_health_pct: number;
    agent_online_pct: number;
    alert_by_severity: OverviewLabelCount[];
    loggie_by_health: OverviewLabelCount[];
    servers_by_status: OverviewLabelCount[];
  };
  alerts_top: OverviewScreenAlert[];
  recent_releases: OverviewScreenRelease[];
  recent_changes: OverviewScreenChange[];
  loggie_offline_sample: OverviewScreenLoggie[];
}

export function getOverview() {
  return getData<OverviewResponse>(http.get("/overview", { silentErrorToast: true }));
}

export function getOverviewScreen() {
  return getData<OverviewScreenResponse>(http.get("/overview/screen", { silentErrorToast: true }));
}

export function getOverviewProjectLaunches() {
  return getData<OverviewProjectLaunchesResponse>(http.get("/overview/project-launches", { silentErrorToast: true }));
}

export function getOverviewReleaseByPerson() {
  return getData<OverviewReleaseByPersonResponse>(http.get("/overview/release-by-person", { silentErrorToast: true }));
}
