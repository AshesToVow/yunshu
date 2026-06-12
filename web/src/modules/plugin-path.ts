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
