// @ts-nocheck
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

export function getOverview() {
  return getData<OverviewResponse>(http.get("/overview", { silentErrorToast: true }));
}

export function getOverviewProjectLaunches() {
  return getData<OverviewProjectLaunchesResponse>(http.get("/overview/project-launches", { silentErrorToast: true }));
}

export function getOverviewReleaseByPerson() {
  return getData<OverviewReleaseByPersonResponse>(http.get("/overview/release-by-person", { silentErrorToast: true }));
}
