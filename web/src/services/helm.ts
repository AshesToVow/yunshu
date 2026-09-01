import { getData, http } from "./http";

export type HarborConfigInfo = {
  url: string;
  project: string;
  oci_prefix: string;
  chart_repo_url: string;
  auth_configured: boolean;
};

export type HarborChartSummary = {
  name: string;
  latest_version: string;
  total_versions: number;
  deprecated: boolean;
};

export type HarborChartVersion = {
  version: string;
  app_version?: string;
  created?: string;
  deprecated: boolean;
};

export type HelmReleaseItem = {
  name: string;
  namespace: string;
  chart: string;
  app_version?: string;
  version: number;
  status: string;
  updated?: string;
  notes?: string;
};

export type HelmReleaseHistoryItem = {
  revision: number;
  status: string;
  chart: string;
  app_version?: string;
  updated?: string;
  description?: string;
};

export async function getHarborInfo() {
  return getData<HarborConfigInfo>(http.get("/helm/harbor/info"));
}

export async function listHarborCharts(keyword?: string) {
  return getData<HarborChartSummary[]>(http.get("/helm/harbor/charts", { params: { keyword: keyword || undefined } }));
}

export async function listHarborChartVersions(chartName: string) {
  return getData<HarborChartVersion[]>(http.get("/helm/harbor/charts/versions", { params: { chart_name: chartName } }));
}

export async function listHelmReleases(clusterId: number, namespace?: string, keyword?: string) {
  return getData<HelmReleaseItem[]>(
    http.get("/helm/releases", {
      params: { cluster_id: clusterId, namespace: namespace || undefined, keyword: keyword || undefined },
    }),
  );
}

export async function getHelmRelease(clusterId: number, namespace: string, releaseName: string) {
  return getData<HelmReleaseItem>(
    http.get("/helm/releases/detail", { params: { cluster_id: clusterId, namespace, release_name: releaseName } }),
  );
}

export async function getHelmReleaseHistory(clusterId: number, namespace: string, releaseName: string) {
  return getData<HelmReleaseHistoryItem[]>(
    http.get("/helm/releases/history", { params: { cluster_id: clusterId, namespace, release_name: releaseName } }),
  );
}

export async function getHelmReleaseValues(clusterId: number, namespace: string, releaseName: string) {
  return getData<{ values: Record<string, unknown> }>(
    http.get("/helm/releases/values", { params: { cluster_id: clusterId, namespace, release_name: releaseName } }),
  );
}

export async function installHelmRelease(body: {
  cluster_id: number;
  namespace: string;
  release_name: string;
  chart_name: string;
  chart_version?: string;
  values?: Record<string, unknown>;
  create_namespace?: boolean;
}) {
  return getData<HelmReleaseItem>(http.post("/helm/releases/install", body));
}

export async function upgradeHelmRelease(body: {
  cluster_id: number;
  namespace: string;
  release_name: string;
  chart_name?: string;
  chart_version?: string;
  values?: Record<string, unknown>;
}) {
  return getData<HelmReleaseItem>(http.post("/helm/releases/upgrade", body));
}

export async function rollbackHelmRelease(body: {
  cluster_id: number;
  namespace: string;
  release_name: string;
  revision: number;
}) {
  return getData<HelmReleaseItem>(http.post("/helm/releases/rollback", body));
}

export async function uninstallHelmRelease(clusterId: number, namespace: string, releaseName: string) {
  return getData<boolean>(
    http.delete("/helm/releases", {
      params: { cluster_id: clusterId, namespace, release_name: releaseName, confirm: true },
    }),
  );
}
