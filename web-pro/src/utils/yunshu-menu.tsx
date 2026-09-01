import type { MenuDataItem } from '@ant-design/pro-components';
import * as IconMap from '@ant-design/icons';
import React from 'react';
import { filterMenuTreeByPlugins } from '@/modules/filter-menu';
import { DEFAULT_ENABLED_PLUGINS, setPluginManifests } from '@/modules/plugin-path';
import { getMenuTree } from '@/services/yunshu/menus';
import { listPlugins } from '@/services/yunshu/plugins';

export function menuIconByName(name?: string) {
  if (!name?.trim()) return undefined;
  const Cmp = (IconMap as unknown as Record<string, React.ComponentType<object>>)[name.trim()];
  return Cmp ? React.createElement(Cmp) : undefined;
}

function filterVisible(menus: YunshuAPI.MenuItem[]): YunshuAPI.MenuItem[] {
  return menus
    .filter((m) => m.status === 1 && !m.hidden)
    .sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0))
    .map((m) => ({
      ...m,
      children: m.children?.length ? filterVisible(m.children) : undefined,
    }));
}

/** 后端菜单 → ProLayout menu.request 格式（含插件过滤） */
export async function fetchProLayoutMenu(): Promise<MenuDataItem[]> {
  const [tree, pluginResult] = await Promise.all([
    getMenuTree().catch(() => [] as YunshuAPI.MenuItem[]),
    listPlugins({ skipErrorHandler: true }).catch(() => null),
  ]);
  const enabled = pluginResult?.enabled?.length ? pluginResult.enabled : [...DEFAULT_ENABLED_PLUGINS];
  if (pluginResult?.plugins?.length) {
    setPluginManifests(pluginResult.plugins);
  }
  const enabledSet = new Set(enabled.map((n) => n.toLowerCase()));
  const isPluginEnabled = (name: string) => enabledSet.has(name.trim().toLowerCase());
  return buildProLayoutMenu(filterMenuTreeByPlugins(tree as Parameters<typeof filterMenuTreeByPlugins>[0], isPluginEnabled));
}

/** 后端菜单 → ProLayout menu.request 格式 */
export function buildProLayoutMenu(menus: YunshuAPI.MenuItem[]): MenuDataItem[] {
  const nodes = filterVisible(menus);
  return nodes.map((m) => toProItem(m)).filter(Boolean) as MenuDataItem[];
}

function toProItem(m: YunshuAPI.MenuItem): MenuDataItem | null {
  const children = m.children?.length ? buildProLayoutMenu(m.children) : undefined;
  const path = m.path?.trim() || '/';
  if (children?.length) {
    return {
      key: path !== '/' ? path : `menu-${m.id}`,
      name: m.name,
      icon: menuIconByName(m.icon),
      children,
    };
  }
  return {
    key: path,
    path,
    name: m.name,
    icon: menuIconByName(m.icon),
  };
}

/** 按 path 查找菜单项（用于 dynamic-menu 解析 component） */
export function findMenuByPath(
  menus: YunshuAPI.MenuItem[],
  pathname: string,
): YunshuAPI.MenuItem | undefined {
  const norm = pathname.replace(/\/+$/, '') || '/';
  for (const m of menus) {
    const p = (m.path?.trim() || '/').replace(/\/+$/, '') || '/';
    if (p === norm) return m;
    if (m.children?.length) {
      const hit = findMenuByPath(m.children, pathname);
      if (hit) return hit;
    }
  }
  return undefined;
}

/** 已迁移到 web-pro 原生 Pro 页的 component 字段 → Umi 路由 */
export const MIGRATED_COMPONENT_MAP: Record<string, string> = {
  'workflow-inbox-page': '/workflow/inbox',
  'login-logs-page': '/login-logs',
  'operation-logs-page': '/operation-logs',
  'users-page': '/users',
  'departments-page': '/departments',
  'roles-page': '/roles',
  'banned-ips-page': '/banned-ips',
  'menus-page': '/menus',
  'permissions-page': '/permissions',
  'policies-page': '/policies',
  'cicd-release-records-page': '/cicd/release-records',
  'cicd-build-records-page': '/cicd/build-records',
  'cicd-services-page': '/cicd/services',
  'cicd-registries-page': '/cicd/registries',
  'cicd-image-browser-page': '/cicd/image-browser',
  'dbmgmt-instances-page': '/dbmgmt/instances',
  'dict-entries-page': '/dict-entries',
  'projects-page': '/projects',
  'dbmgmt-tickets-page': '/dbmgmt/workflow/history',
  'dbmgmt-workflow-history-page': '/dbmgmt/workflow/history',
  'dbmgmt-grants-page': '/dbmgmt/grants',
  'dbmgmt-query-grants-page': '/dbmgmt/apply/query-grants',
  'esmgmt-connections-page': '/esmgmt/connections',
  'user-groups-page': '/user-groups',
  'registrations-page': '/registrations',
  'alert-channels-page': '/alert-channels',
  'alert-duty-page': '/alert-duty',
  'alert-maintenance-page': '/alert-maintenance',
  'ai-approvals-page': '/ai/approvals',
  'ai-investigations-page': '/ai/investigations',
  'personal-settings-page': '/personal-settings',
  'plugins-page': '/plugins',
  'dashboard-page': '/dashboard',
  'dbmgmt-audit-page': '/dbmgmt/audit',
  'alert-quality-page': '/alert-quality',
  'alert-events-page': '/alert-events',
  'workflow-definitions-page': '/workflow/definitions',
  'dbmgmt-access-requests-page': '/dbmgmt/access-requests/all',
  'dbmgmt-query-apply-page': '/dbmgmt/apply/query',
  'dbmgmt-database-apply-page': '/dbmgmt/apply/database',
  'dbmgmt-app-user-apply-page': '/dbmgmt/apply/app-user',
  'esmgmt-overview-page': '/esmgmt/overview',
  'ai-center-page': '/ai/center',
  'platform-templates-page': '/platform-templates',
  'workflow-tickets-page': '/workflow/tickets',
  'cluster-page': '/clusters',
  'pod-page': '/pods',
  'namespaces-page': '/namespaces',
  'nodes-page': '/nodes',
  'component-status-page': '/component-status',
  'cluster-api-resources-page': '/cluster-api-resources',
  'horizontal-pod-autoscalers-page': '/horizontal-pod-autoscalers',
  'k8s-resource-topology-page': '/k8s-resource-topology',
  'deployments-page': '/deployments',
  'statefulsets-page': '/statefulsets',
  'daemonsets-page': '/daemonsets',
  'cronjobs-page': '/cronjobs',
  'jobs-page': '/jobs',
  'configmaps-page': '/configmaps',
  'secrets-page': '/secrets',
  'ingresses-page': '/ingresses',
  'ingress-classes-page': '/ingress-classes',
  'events-page': '/events',
  'k8s-services-page': '/k8s-services',
  'persistentvolumes-page': '/persistentvolumes',
  'persistentvolumeclaims-page': '/persistentvolumeclaims',
  'storageclasses-page': '/storageclasses',
  'crds-page': '/crds',
  'crs-page': '/crs',
  'rbac-roles-page': '/rbac/roles',
  'rbac-rolebindings-page': '/rbac/rolebindings',
  'rbac-clusterroles-page': '/rbac/clusterroles',
  'rbac-clusterrolebindings-page': '/rbac/clusterrolebindings',
  'serviceaccounts-page': '/serviceaccounts',
  'network-policies-page': '/network-policies',
  'helm-charts-page': '/helm/charts',
  'helm-releases-page': '/helm/releases',
  'k8s-scoped-policies-page': '/k8s-scoped-policies',
  'k8s-cr-templates-page': '/k8s-cr-templates',
  'k8s-event-forward-page': '/k8s/event-forward',
  'project-servers-page': '/project-servers',
  'project-inspect-page': '/project-inspect',
  'project-members-page': '/project-members',
  'project-logs-page': '/project-logs',
  'project-collect-config-page': '/project-services',
  'project-services-page': '/project-services',
  'project-log-sources-page': '/project-log-sources',
  'service-catalog-page': '/service-catalog',
  'service-portrait-page': '/service-portrait',
  'log-retention-page': '/log-retention',
  'loggie-status-page': '/loggie-status',
  'server-console-page': '/server-console',
  'mysql-backup-page': '/mysql-backup',
  'dbmgmt-sql-query-page': '/dbmgmt/sql/query',
  'dbmgmt-console-page': '/dbmgmt/console',
  'dbmgmt-sql-audit-page': '/dbmgmt/sql/audit',
  'esmgmt-console-page': '/esmgmt/console',
  'ai-assistant-page': '/ai/assistant',
  'alert-config-center-page': '/alert-config-center',
  'alert-monitor-platform-page': '/alert-monitor-platform',
  'cicd-todo-page': '/workflow/inbox?domain=cicd',
  'dbmgmt-todo-page': '/workflow/inbox?domain=dbmgmt',
  'cicd-approval-flow-page': '/workflow/definitions?domain=cicd',
  'dbmgmt-approval-flow-page': '/workflow/definitions?domain=dbmgmt',
  'project-cluster-log-page': '/project-services?tab=cluster-log',
};

/** path 重定向（与旧 web menu-path 对齐的常用项） */
export const PATH_ALIASES: Record<string, string> = {
  '/cicd/todo': '/workflow/inbox?domain=cicd',
  '/dbmgmt/todo': '/workflow/inbox?domain=dbmgmt',
  '/dbmgmt/workflow/pending': '/workflow/inbox?domain=dbmgmt',
  '/dbmgmt/tickets': '/dbmgmt/workflow/history',
  '/dbmgmt/approval-flow': '/workflow/definitions?domain=dbmgmt',
  '/cicd/approval-flow': '/workflow/definitions?domain=cicd',
  '/dbmgmt/access-request': '/dbmgmt/access-requests/all',
  '/dbmgmt/access-requests': '/dbmgmt/apply/query',
  '/dbmgmt/console': '/dbmgmt/sql/query',
  '/project-log-sources': '/project-services?tab=log-sources',
};

export function resolveMenuPath(pathname: string): string {
  const norm = pathname.replace(/\/+$/, '') || '/';
  return PATH_ALIASES[norm] ?? norm;
}
