import { Alert, Card, Result, Spin, Typography } from "antd";
import { Suspense, useMemo } from "react";
import { Link, Navigate, useLocation } from "react-router-dom";
import { useMenuTree } from "../hooks/use-menu-tree";
import { createLazyMenuPage } from "../utils/menu-page-loader";
import { findMenuByPath, normalizeMenuPath } from "../utils/menu-path";
import { usePlugins } from "../contexts/plugin-context";
import { isPathAllowedByPlugins } from "../modules/plugin-path";

const PATH_COMPONENT_FALLBACK: Record<string, string> = {
  "/clusters": "cluster-page",
  "/pods": "pod-page",
  "/departments": "departments-page",
  "/namespaces": "namespaces-page",
  "/nodes": "nodes-page",
  "/component-status": "component-status-page",
  "/cluster-api-resources": "cluster-api-resources-page",
  "/horizontal-pod-autoscalers": "horizontal-pod-autoscalers-page",
  "/k8s-resource-topology": "k8s-resource-topology-page",
  "/deployments": "deployments-page",
  "/statefulsets": "statefulsets-page",
  "/daemonsets": "daemonsets-page",
  "/cronjobs": "cronjobs-page",
  "/jobs": "jobs-page",
  "/configmaps": "configmaps-page",
  "/secrets": "secrets-page",
  "/ingresses": "ingresses-page",
  "/ingress-classes": "ingress-classes-page",
  "/events": "events-page",
  "/k8s-services": "k8s-services-page",
  "/persistentvolumes": "persistentvolumes-page",
  "/persistentvolumeclaims": "persistentvolumeclaims-page",
  "/storageclasses": "storageclasses-page",
  "/crds": "crds-page",
  "/crs": "crs-page",
  "/rbac/roles": "rbac-roles-page",
  "/rbac/rolebindings": "rbac-rolebindings-page",
  "/rbac/clusterroles": "rbac-clusterroles-page",
  "/rbac/clusterrolebindings": "rbac-clusterrolebindings-page",
  "/serviceaccounts": "serviceaccounts-page",
  "/k8s-scoped-policies": "k8s-scoped-policies-page",
  "/user-groups": "user-groups-page",
  "/alert-channels": "alert-channels-page",
  "/alert-events": "alert-events-page",
  "/alert-config-center": "alert-config-center-page",
  "/alert-monitor-platform": "alert-monitor-platform-page",
  "/alert-duty": "alert-duty-page",
  "/alert-maintenance": "alert-maintenance-page",
  "/dict-entries": "dict-entries-page",
  "/k8s/event-forward": "k8s-event-forward-page",
  "/mysql-backup": "mysql-backup-page",
  "/dbmgmt/instances": "dbmgmt-instances-page",
  "/dbmgmt/instances/:instanceId": "dbmgmt-instance-detail-page",
  "/dbmgmt/apply/database": "dbmgmt-database-apply-page",
  "/dbmgmt/apply/query": "dbmgmt-query-apply-page",
  "/dbmgmt/apply/app-user": "dbmgmt-app-user-apply-page",
  "/dbmgmt/apply/query-grants": "dbmgmt-query-grants-page",
  "/dbmgmt/access-requests/all": "dbmgmt-access-requests-page",
  "/dbmgmt/access-request": "dbmgmt-access-requests-page",
  "/dbmgmt/sql/query": "dbmgmt-sql-query-page",
  "/dbmgmt/sql/audit": "dbmgmt-sql-audit-page",
  "/dbmgmt/workflow/pending": "dbmgmt-workflow-pending-page",
  "/dbmgmt/workflow/history": "dbmgmt-workflow-history-page",
  "/dbmgmt/console": "dbmgmt-sql-query-page",
  "/dbmgmt/access-requests": "dbmgmt-query-apply-page",
  "/dbmgmt/todo": "dbmgmt-workflow-pending-page",
  "/dbmgmt/approval-flow": "dbmgmt-approval-flow-page",
  "/dbmgmt/tickets": "dbmgmt-workflow-history-page",
  "/dbmgmt/audit": "dbmgmt-audit-page",
  "/dbmgmt/grants": "dbmgmt-grants-page",
  "/cicd/services": "cicd-services-page",
  "/cicd/todo": "cicd-todo-page",
  "/cicd/approval-flow": "cicd-approval-flow-page",
  "/cicd/build-records": "cicd-build-records-page",
  "/cicd/release-records": "cicd-release-records-page",
  "/dashboard": "dashboard-page",
  "/projects": "projects-page",
  "/project-servers": "project-servers-page",
  "/project-services": "project-collect-config-page",
  "/project-members": "project-members-page",
  "/project-logs": "project-logs-page",
  "/log-retention": "log-retention-page",
  "/loggie-status": "loggie-status-page",
  "/project-log-sources": "project-collect-config-page",
  "/network-policies": "network-policies-page",
  "/users": "users-page",
  "/roles": "roles-page",
  "/menus": "menus-page",
  "/policies": "policies-page",
  "/permissions": "permissions-page",
  "/login-logs": "login-logs-page",
  "/operation-logs": "operation-logs-page",
  "/banned-ips": "banned-ips-page",
  "/registrations": "registrations-page",
  "/personal-settings": "personal-settings-page",
  "/server-console": "server-console-page",
};

function RouteFallback() {
  return (
    <div className="page-loading">
      <Spin size="large" />
    </div>
  );
}

export function DynamicMenuPage() {
  const location = useLocation();
  const { isPluginEnabled, loading: pluginsLoading } = usePlugins();
  const { menus, loading: menuLoading, error: menuError } = useMenuTree();

  const menuItem = useMemo(() => {
    if (!menus?.length) return undefined;
    return findMenuByPath(menus, location.pathname);
  }, [menus, location.pathname]);
  const loadError = menuError instanceof Error ? menuError.message : menuError ? "加载菜单失败" : null;

  const LazyComp = useMemo(() => {
    const normalizedPath = normalizeMenuPath(location.pathname);
    const fallbackComp = PATH_COMPONENT_FALLBACK[normalizedPath];
    // 对核心内置页面优先使用 path->component 映射，避免菜单 component 配置错误导致串页
    const c = (fallbackComp ?? menuItem?.component)?.trim();
    if (!c) return null;
    return createLazyMenuPage(c);
  }, [location.pathname, menuItem?.component]);

  const normalizedPath = useMemo(() => normalizeMenuPath(location.pathname), [location.pathname]);
  const hasPathFallback = Boolean(PATH_COMPONENT_FALLBACK[normalizedPath]);

  if (normalizedPath === "/log-kafka") {
    return <Navigate to="/log-retention?tab=kafka" replace />;
  }

  if (!pluginsLoading && !isPathAllowedByPlugins(location.pathname, isPluginEnabled)) {
    return (
      <Result
        status="403"
        title="业务模块未启用"
        subTitle="该页面所属插件未在服务端 config.yaml 的 plugins.enabled 中启用。"
      />
    );
  }

  if (loadError) {
    return <Result status="error" title="菜单加载失败" subTitle={loadError} />;
  }

  if (menuLoading && !menus.length) {
    return (
      <div style={{ padding: 48, textAlign: "center" }}>
        <Spin tip="加载菜单..." />
      </div>
    );
  }

  if (!menuItem && !hasPathFallback) {
    return (
      <Result
        status="404"
        title="未找到菜单"
        subTitle={`当前地址 ${location.pathname} 未在「菜单管理」中配置，或已被隐藏/停用。`}
        extra={
          <Link to="/menus">
            <Typography.Link>前往菜单管理</Typography.Link>
          </Link>
        }
      />
    );
  }

  if (menuItem && menuItem.children && menuItem.children.length > 0) {
    return (
      <Card className="table-card" title={menuItem.name}>
        <Typography.Paragraph type="secondary">这是目录菜单，请从左侧选择具体子菜单进入页面。</Typography.Paragraph>
        <ul style={{ margin: 0, paddingLeft: 20 }}>
          {menuItem.children
            .filter((c) => c.status === 1 && !c.hidden)
            .map((c) => (
              <li key={c.id}>
                <Link to={normalizeMenuPath(c.path)}>{c.name}</Link>
                <Typography.Text type="secondary"> {c.path}</Typography.Text>
              </li>
            ))}
        </ul>
      </Card>
    );
  }

  if (menuItem && !menuItem.component?.trim() && !hasPathFallback) {
    return (
      <Card className="table-card">
        <Result
          status="info"
          title="未配置前端组件"
          subTitle={
            <span>
              请在「菜单管理」中为该菜单填写 <Typography.Text code>组件路径</Typography.Text>（例如 <Typography.Text code>containerd-page</Typography.Text>
              ），并新增对应文件 <Typography.Text code>src/pages/containerd-page.tsx</Typography.Text>，导出{" "}
              <Typography.Text code>ContainerdPage</Typography.Text>。
            </span>
          }
        />
      </Card>
    );
  }

  if (!LazyComp) {
    const normalized = normalizeMenuPath(location.pathname);
    const fallbackComp = PATH_COMPONENT_FALLBACK[normalized];
    const compText = (menuItem?.component ?? fallbackComp ?? "").trim();
    return (
      <Card className="table-card">
        <Result
          status="warning"
          title="未找到页面文件"
          subTitle={
            <span>
              已填写 component 为「{compText || "（空）"}」，但未找到匹配的 <Typography.Text code>src/pages/**/*-page.tsx</Typography.Text>。
              请新建文件并导出与文件名对应的 PascalCase 组件名。
            </span>
          }
          extra={
            <Alert
              type="info"
              showIcon
              message="命名约定"
              description="例如 component 填 foo-bar-page，则需存在 src/pages/foo-bar-page.tsx，且 export function FooBarPage() { ... }"
            />
          }
        />
      </Card>
    );
  }

  return (
    <Suspense fallback={<RouteFallback />}>
      <LazyComp />
    </Suspense>
  );
}
