/** 与后端 plugins.enabled 默认全集一致 */
export const DEFAULT_ENABLED_PLUGINS = ["core", "k8s", "alert", "project", "cmdb", "backup"] as const;

export type PluginName = (typeof DEFAULT_ENABLED_PLUGINS)[number];

/** 菜单 path → 所属插件；未命中则不受插件开关约束 */
const PATH_PLUGIN_RULES: { plugin: PluginName; prefixes: string[] }[] = [
  {
    plugin: "core",
    prefixes: [
      "/users",
      "/departments",
      "/roles",
      "/permissions",
      "/policies",
      "/registrations",
      "/menus",
      "/login-logs",
      "/operation-logs",
      "/banned-ips",
      "/dict-entries",
      "/personal-settings",
      "/user-groups",
      "/plugins",
    ],
  },
  {
    plugin: "k8s",
    prefixes: [
      "/clusters",
      "/cluster",
      "/pods",
      "/namespaces",
      "/nodes",
      "/component-status",
      "/cluster-api-resources",
      "/horizontal-pod-autoscalers",
      "/k8s-resource-topology",
      "/deployments",
      "/statefulsets",
      "/daemonsets",
      "/cronjobs",
      "/jobs",
      "/configmaps",
      "/secrets",
      "/ingresses",
      "/ingress-classes",
      "/events",
      "/k8s-services",
      "/persistentvolumes",
      "/persistentvolumeclaims",
      "/storageclasses",
      "/crds",
      "/crs",
      "/rbac",
      "/serviceaccounts",
      "/k8s-scoped-policies",
      "/network-policies",
      "/k8s/",
    ],
  },
  {
    plugin: "alert",
    prefixes: ["/alert-"],
  },
  {
    plugin: "project",
    prefixes: [
      "/projects",
      "/application-topology",
      "/project-members",
      "/project-services",
      "/project-logs",
      "/project-log-sources",
      "/agent-list",
    ],
  },
  {
    plugin: "cmdb",
    prefixes: ["/project-servers", "/server-console"],
  },
  {
    plugin: "backup",
    prefixes: ["/mysql-backup"],
  },
];

function normalizePath(path: string): string {
  const raw = path.trim().toLowerCase();
  if (!raw || raw === "/") return "/";
  const withSlash = raw.startsWith("/") ? raw : `/${raw}`;
  return withSlash.replace(/\/+$/, "") || "/";
}

/** 解析路径归属的业务插件；首页总览依赖 k8s 插件 */
export function resolvePathPlugin(path: string): PluginName | null {
  const normalized = normalizePath(path);
  if (normalized === "/") return "k8s";
  for (const rule of PATH_PLUGIN_RULES) {
    if (rule.prefixes.some((p) => normalized === p || normalized.startsWith(`${p}/`) || normalized.startsWith(p))) {
      return rule.plugin;
    }
  }
  return null;
}

export function isPathAllowedByPlugins(path: string, isPluginEnabled: (name: string) => boolean): boolean {
  const normalized = normalizePath(path);
  const cmdbPaths = ["/project-servers", "/server-console"];
  if (cmdbPaths.some((p) => normalized === p || normalized.startsWith(`${p}/`))) {
    return isCmdbPageAllowed(isPluginEnabled);
  }
  const plugin = resolvePathPlugin(path);
  if (!plugin) return true;
  return isPluginEnabled(plugin);
}

/** CMDB 页面通常依赖 project 上下文；两者应同时启用 */
export function isCmdbPageAllowed(isPluginEnabled: (name: string) => boolean): boolean {
  return isPluginEnabled("cmdb") && isPluginEnabled("project");
}

const API_RESOURCE_PLUGIN_RULES: { plugin: PluginName; prefixes: string[] }[] = [
  {
    plugin: "core",
    prefixes: [
      "/api/v1/users",
      "/api/v1/departments",
      "/api/v1/roles",
      "/api/v1/permissions",
      "/api/v1/policies",
      "/api/v1/registrations",
      "/api/v1/menus",
      "/api/v1/login-logs",
      "/api/v1/operation-logs",
      "/api/v1/security",
      "/api/v1/dict-entries",
      "/api/v1/user-groups",
      "/api/v1/plugins",
      "/api/v1/overview",
    ],
  },
  {
    plugin: "k8s",
    prefixes: [
      "/api/v1/clusters",
      "/api/v1/pods",
      "/api/v1/namespaces",
      "/api/v1/nodes",
      "/api/v1/k8s-policies",
      "/api/v1/k8s-namespace-deny-rules",
      "/api/v1/k8s-namespace-allow-rules",
      "/api/v1/k8s/",
    ],
  },
  { plugin: "alert", prefixes: ["/api/v1/alerts"] },
  { plugin: "project", prefixes: ["/api/v1/projects"] },
  { plugin: "cmdb", prefixes: ["/api/v1/servers", "/api/v1/cloud-accounts", "/api/v1/server-groups"] },
  { plugin: "backup", prefixes: ["/api/v1/mysql-backup"] },
];

export function resolveAPIResourcePlugin(resource: string): PluginName | null {
  const r = resource.trim().toLowerCase();
  if (!r) return null;
  for (const rule of API_RESOURCE_PLUGIN_RULES) {
    if (rule.prefixes.some((p) => r === p || r.startsWith(`${p}/`) || r.startsWith(p))) {
      return rule.plugin;
    }
  }
  return null;
}

export function isAPIResourceAllowedByPlugins(
  resource: string,
  isPluginEnabled: (name: string) => boolean,
): boolean {
  const plugin = resolveAPIResourcePlugin(resource);
  if (!plugin) return true;
  if (plugin === "cmdb") return isCmdbPageAllowed(isPluginEnabled);
  return isPluginEnabled(plugin);
}

export function filterPermissionsByPlugins<T extends { resource: string }>(
  items: T[],
  isPluginEnabled: (name: string) => boolean,
): T[] {
  return items.filter((it) => isAPIResourceAllowedByPlugins(it.resource, isPluginEnabled));
}
