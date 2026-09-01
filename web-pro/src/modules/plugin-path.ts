// @ts-nocheck
/** 离线兜底：与后端 plugin.DefaultEnabled 一致；正常应以后端 /plugins 返回为准 */
export const DEFAULT_ENABLED_PLUGINS = ["core", "k8s", "alert", "project", "cmdb", "backup", "cicd", "dbmgmt", "inspect", "ai", "esmgmt"] as const;

export type PluginName = string;

export type PluginManifest = {
  menu_path_prefixes?: string[];
  api_prefixes?: string[];
  depends_on?: string[];
  workers?: string[];
};

export type PluginInfoWithManifest = {
  name: string;
  description?: string;
  enabled?: boolean;
  manifest?: PluginManifest;
};

let runtimeManifests: PluginInfoWithManifest[] = [];

/** 由 PluginProvider 注入后端契约；未注入时回退本地默认规则 */
export function setPluginManifests(plugins: PluginInfoWithManifest[]) {
  runtimeManifests = Array.isArray(plugins) ? plugins : [];
}

export function getPluginManifests(): PluginInfoWithManifest[] {
  return runtimeManifests;
}

const FALLBACK_PATH_RULES: { plugin: string; prefixes: string[] }[] = [
  {
    plugin: "core",
    prefixes: [
      "/users", "/departments", "/roles", "/permissions", "/policies", "/registrations",
      "/menus", "/login-logs", "/operation-logs", "/banned-ips", "/dict-entries",
      "/personal-settings", "/user-groups", "/plugins",
    ],
  },
  {
    plugin: "k8s",
    prefixes: [
      "/clusters", "/cluster", "/pods", "/namespaces", "/nodes", "/component-status",
      "/cluster-api-resources", "/horizontal-pod-autoscalers", "/helm/releases", "/helm/charts",
      "/k8s-resource-topology", "/deployments", "/statefulsets", "/daemonsets", "/cronjobs", "/jobs",
      "/configmaps", "/secrets", "/ingresses", "/ingress-classes", "/events", "/k8s-services",
      "/persistentvolumes", "/persistentvolumeclaims", "/storageclasses", "/crds", "/crs", "/k8s-cr-templates", "/rbac",
      "/serviceaccounts", "/k8s-scoped-policies", "/network-policies", "/k8s/",
    ],
  },
  { plugin: "alert", prefixes: ["/alert-"] },
  {
    plugin: "project",
    prefixes: [
      "/projects", "/project-members", "/project-services", "/service-catalog", "/service-portrait",
      "/project-logs", "/log-retention", "/loggie-status", "/project-log-sources",
    ],
  },
  { plugin: "cmdb", prefixes: ["/project-servers", "/server-console"] },
  { plugin: "backup", prefixes: ["/mysql-backup"] },
  { plugin: "dbmgmt", prefixes: ["/dbmgmt"] },
  { plugin: "cicd", prefixes: ["/cicd"] },
  { plugin: "inspect", prefixes: ["/project-inspect"] },
  { plugin: "ai", prefixes: ["/ai"] },
  { plugin: "esmgmt", prefixes: ["/esmgmt"] },
];

function normalizePath(path: string): string {
  const raw = path.trim().toLowerCase();
  if (!raw || raw === "/") return "/";
  const withSlash = raw.startsWith("/") ? raw : `/${raw}`;
  return withSlash.replace(/\/+$/, "") || "/";
}

function pathRules(): { plugin: string; prefixes: string[] }[] {
  if (runtimeManifests.length === 0) return FALLBACK_PATH_RULES;
  return runtimeManifests
    .filter((p) => (p.manifest?.menu_path_prefixes?.length ?? 0) > 0)
    .map((p) => ({ plugin: p.name, prefixes: p.manifest!.menu_path_prefixes! }));
}

function dependsOf(pluginName: string): string[] {
  const hit = runtimeManifests.find((p) => p.name.toLowerCase() === pluginName.toLowerCase());
  return hit?.manifest?.depends_on ?? defaultDepends(pluginName);
}

function defaultDepends(pluginName: string): string[] {
  const n = pluginName.toLowerCase();
  if (["cmdb", "cicd", "dbmgmt", "backup", "inspect"].includes(n)) return ["project"];
  return [];
}

function pluginAndDepsEnabled(pluginName: string, isPluginEnabled: (name: string) => boolean): boolean {
  if (!isPluginEnabled(pluginName)) return false;
  return dependsOf(pluginName).every((d) => isPluginEnabled(d));
}

/** 解析路径归属的业务插件；首页总览依赖 k8s 插件 */
export function resolvePathPlugin(path: string): PluginName | null {
  const normalized = normalizePath(path);
  if (normalized === "/") return "k8s";
  for (const rule of pathRules()) {
    if (rule.prefixes.some((p) => {
      const prefix = normalizePath(p);
      if (prefix.endsWith("-")) {
        return normalized === prefix || normalized.startsWith(prefix);
      }
      return normalized === prefix || normalized.startsWith(`${prefix}/`);
    })) {
      return rule.plugin;
    }
  }
  return null;
}

export function isPathAllowedByPlugins(path: string, isPluginEnabled: (name: string) => boolean): boolean {
  const plugin = resolvePathPlugin(path);
  if (!plugin) return true;
  return pluginAndDepsEnabled(plugin, isPluginEnabled);
}

export function isCmdbPageAllowed(isPluginEnabled: (name: string) => boolean): boolean {
  return pluginAndDepsEnabled("cmdb", isPluginEnabled);
}

export function isCicdAllowed(isPluginEnabled: (name: string) => boolean): boolean {
  return pluginAndDepsEnabled("cicd", isPluginEnabled);
}

export function isDbmgmtAllowed(isPluginEnabled: (name: string) => boolean): boolean {
  return pluginAndDepsEnabled("dbmgmt", isPluginEnabled);
}

export function isBackupAllowed(isPluginEnabled: (name: string) => boolean): boolean {
  return pluginAndDepsEnabled("backup", isPluginEnabled);
}

export function isInspectAllowed(isPluginEnabled: (name: string) => boolean): boolean {
  return pluginAndDepsEnabled("inspect", isPluginEnabled);
}

export function resolveAPIResourcePlugin(resource: string): PluginName | null {
  const r = resource.trim().toLowerCase();
  if (!r) return null;
  const cicdOverview = ["/api/v1/overview/project-launches", "/api/v1/overview/release-by-person"];
  if (cicdOverview.some((p) => r === p || r.startsWith(`${p}/`))) return "cicd";
  if (r.includes("/projects/") && r.includes("/dbmgmt")) return "dbmgmt";
  if (r.includes("/projects/") && r.includes("/mysql-backup")) return "backup";
  if (r.includes("/projects/") && r.includes("/cicd")) return "cicd";
  if (r.includes("/projects/") && r.includes("/inspect")) return "inspect";

  if (runtimeManifests.length > 0) {
    for (const p of runtimeManifests) {
      const prefixes = p.manifest?.api_prefixes ?? [];
      for (const prefix of prefixes) {
        const pref = prefix.trim().toLowerCase();
        if (!pref) continue;
        const base = pref.replace(/\/+$/, "");
        if (r === pref || r === base || r.startsWith(`${base}/`)) return p.name;
      }
    }
    return null;
  }

  const fallback: { plugin: string; prefixes: string[] }[] = [
    {
      plugin: "core",
      prefixes: [
        "/api/v1/users", "/api/v1/departments", "/api/v1/roles", "/api/v1/permissions",
        "/api/v1/policies", "/api/v1/registrations", "/api/v1/menus", "/api/v1/login-logs",
        "/api/v1/operation-logs", "/api/v1/security", "/api/v1/dict-entries", "/api/v1/user-groups",
        "/api/v1/plugins", "/api/v1/overview",
      ],
    },
    { plugin: "k8s", prefixes: ["/api/v1/clusters", "/api/v1/pods", "/api/v1/namespaces", "/api/v1/nodes", "/api/v1/k8s-policies", "/api/v1/k8s/", "/api/v1/helm/"] },
    { plugin: "alert", prefixes: ["/api/v1/alerts"] },
    { plugin: "project", prefixes: ["/api/v1/projects"] },
    { plugin: "cmdb", prefixes: ["/api/v1/servers", "/api/v1/cloud-accounts", "/api/v1/server-groups"] },
    { plugin: "backup", prefixes: ["/api/v1/mysql-backup"] },
  ];
  for (const rule of fallback) {
    if (rule.prefixes.some((p) => r === p || r.startsWith(`${p}/`))) return rule.plugin;
  }
  return null;
}

export function isAPIResourceAllowedByPlugins(
  resource: string,
  isPluginEnabled: (name: string) => boolean,
): boolean {
  const plugin = resolveAPIResourcePlugin(resource);
  if (!plugin) return true;
  return pluginAndDepsEnabled(plugin, isPluginEnabled);
}

export function filterPermissionsByPlugins<T extends { resource: string }>(
  items: T[],
  isPluginEnabled: (name: string) => boolean,
): T[] {
  return items.filter((it) => isAPIResourceAllowedByPlugins(it.resource, isPluginEnabled));
}
