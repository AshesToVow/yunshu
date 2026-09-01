// @ts-nocheck
import type { RouteObject } from 'react-router';
import { alertRoutes, ALERT_PLUGIN } from "./alert/routes";
import { aiRoutes, AI_PLUGIN } from "./ai/routes";
import { cicdRoutes, CICD_PLUGIN } from "./cicd/routes";
import { dbmgmtRoutes, DBMGMT_PLUGIN } from "./dbmgmt/routes";
import { esmgmtRoutes, ESMGMT_PLUGIN } from "./esmgmt/routes";
import { cmdbRoutes, CMDB_PLUGIN } from "./cmdb/routes";
import { coreRoutes, CORE_PLUGIN } from "./core/routes";
import { k8sRoutes, K8S_PLUGIN } from "./k8s/routes";
import { PROJECT_PLUGIN } from "./project/routes";

/** alert / backup / project 页面主要通过菜单 component 动态加载 */
const MODULE_ROUTES: Record<string, RouteObject[]> = {
  [CORE_PLUGIN]: coreRoutes,
  [K8S_PLUGIN]: k8sRoutes,
  [CMDB_PLUGIN]: cmdbRoutes,
  [ALERT_PLUGIN]: alertRoutes,
  [CICD_PLUGIN]: cicdRoutes,
  [DBMGMT_PLUGIN]: dbmgmtRoutes,
  [AI_PLUGIN]: aiRoutes,
  [ESMGMT_PLUGIN]: esmgmtRoutes,
};

export function collectModuleRoutes(isPluginEnabled: (name: string) => boolean): RouteObject[] {
  const routes: RouteObject[] = [];
  for (const [plugin, pluginRoutes] of Object.entries(MODULE_ROUTES)) {
    if (!isPluginEnabled(plugin)) continue;
    routes.push(...pluginRoutes);
  }
  // project 插件：菜单动态页，无静态路由表项；但仍需 enabled 才展示对应菜单
  void PROJECT_PLUGIN;
  return routes;
}
