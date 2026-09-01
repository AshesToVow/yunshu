import { MIGRATED_COMPONENT_MAP } from '@/utils/yunshu-menu';
import { PATH_COMPONENT_FALLBACK } from '@/constants/path-component-fallback';
import { useMenuTree } from '@/hooks/use-menu-tree';
import { collectModuleRoutes } from '@/modules';
import { isPathAllowedByPlugins } from '@/modules/plugin-path';
import { createLazyMenuPage } from '@/utils/menu-page-loader';
import { findMenuByPath, normalizeMenuPath, resolveMenuAccessCandidates } from '@/utils/menu-path';
import { Link, Navigate, useLocation, useRoutes } from '@umijs/max';
import type { RouteObject } from 'react-router';
import { Card, Result, Spin, Typography } from 'antd';
import { Suspense, useMemo } from 'react';
import { LegacyShell } from '@/components/LegacyShell';
import { usePlugins } from '@/contexts/plugin-context';

function RouteFallback() {
  return (
    <div style={{ padding: 48, textAlign: 'center' }}>
      <Spin size="large" />
    </div>
  );
}

function absolutizeModuleRoutes(routes: RouteObject[]): RouteObject[] {
  return routes.map((route) => {
    const next: RouteObject = { ...route };
    if (route.index) {
      next.path = '/';
    } else if (route.path && route.path !== '*' && !route.path.startsWith('/')) {
      next.path = `/${route.path}`;
    }
    if (route.children?.length) {
      next.children = absolutizeModuleRoutes(route.children);
    }
    return next;
  });
}

function wrapSuspense(routes: RouteObject[]): RouteObject[] {
  return routes.map((route) => ({
    ...route,
    element: route.element ? <Suspense fallback={<RouteFallback />}>{route.element}</Suspense> : route.element,
    children: route.children ? wrapSuspense(route.children) : undefined,
  }));
}

function resolveMigratedPath(componentField?: string): string | null {
  if (!componentField?.trim()) return null;
  const base = componentField.trim().replace(/^\//, '').replace(/\.tsx$/i, '');
  const key = base.endsWith('-page') ? base : `${base}-page`;
  return MIGRATED_COMPONENT_MAP[key] ?? null;
}

/** Merge menu redirect target (may include ?query) with current location search/hash. */
function joinMigratedTarget(migratedTo: string, search: string, hash: string): string {
  const hashPart = hash || '';
  if (!migratedTo.includes('?')) {
    return `${migratedTo}${search || ''}${hashPart}`;
  }
  const qIndex = migratedTo.indexOf('?');
  const path = migratedTo.slice(0, qIndex);
  const merged = new URLSearchParams(migratedTo.slice(qIndex + 1));
  if (search?.startsWith('?')) {
    new URLSearchParams(search.slice(1)).forEach((v, k) => {
      if (!merged.has(k)) merged.set(k, v);
    });
  }
  const qs = merged.toString();
  return `${path}${qs ? `?${qs}` : ''}${hashPart}`;
}

function DynamicMenuInner() {
  const location = useLocation();
  const { isPluginEnabled, loading: pluginsLoading } = usePlugins();
  const { menus, loading: menuLoading, error: menuError } = useMenuTree();

  const menuItem = useMemo(() => {
    if (!menus?.length) return undefined;
    const found = findMenuByPath(menus, location.pathname);
    if (found) return found;
    for (const accessPath of resolveMenuAccessCandidates(location.pathname)) {
      const via = findMenuByPath(menus, accessPath);
      if (via) return via;
    }
    return undefined;
  }, [menus, location.pathname]);

  const loadError = menuError instanceof Error ? menuError.message : menuError ? '加载菜单失败' : null;

  const LazyComp = useMemo(() => {
    const normalizedPath = normalizeMenuPath(location.pathname);
    const fallbackComp = PATH_COMPONENT_FALLBACK[normalizedPath];
    const c = (fallbackComp ?? menuItem?.component)?.trim();
    if (!c) return null;
    return createLazyMenuPage(c);
  }, [location.pathname, menuItem?.component]);

  const normalizedPath = useMemo(() => normalizeMenuPath(location.pathname), [location.pathname]);
  const hasPathFallback = Boolean(PATH_COMPONENT_FALLBACK[normalizedPath]);
  const componentField = (PATH_COMPONENT_FALLBACK[normalizedPath] ?? menuItem?.component)?.trim();
  const migratedTo = resolveMigratedPath(componentField);

  if (migratedTo && migratedTo !== normalizedPath) {
    return <Navigate to={joinMigratedTarget(migratedTo, location.search, location.hash)} replace />;
  }

  if (normalizedPath === '/log-kafka') {
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
      <div style={{ padding: 48, textAlign: 'center' }}>
        <Spin tip="加载菜单..." />
      </div>
    );
  }

  if (!menuItem && !hasPathFallback) {
    return (
      <Result
        status="403"
        title="无访问权限"
        subTitle={`当前地址 ${location.pathname} 不在已授权菜单中，或已被隐藏/停用。`}
        extra={<Link to="/">返回总览</Link>}
      />
    );
  }

  if (menuItem?.children && menuItem.children.length > 0) {
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
        <Result status="info" title="未配置前端组件" subTitle="请在菜单管理中填写 component 字段。" />
      </Card>
    );
  }

  if (!LazyComp) {
    const compText = (menuItem?.component ?? PATH_COMPONENT_FALLBACK[normalizedPath] ?? '').trim();
    return (
      <Card className="table-card">
        <Result
          status="warning"
          title="未找到页面文件"
          subTitle={`component「${compText || '（空）'}」在 legacy 注册表中不存在，请运行 npm run sync:legacy`}
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

function LegacyRouteSwitch() {
  const { isPluginEnabled, loading: pluginsLoading } = usePlugins();
  const moduleRoutes = useMemo(
    () => wrapSuspense(absolutizeModuleRoutes(collectModuleRoutes(isPluginEnabled))),
    [isPluginEnabled],
  );
  const routes = useMemo(
    (): RouteObject[] => [
      ...moduleRoutes,
      {
        path: '*',
        element: <DynamicMenuInner />,
      },
    ],
    [moduleRoutes],
  );
  const element = useRoutes(routes);

  if (pluginsLoading) {
    return <RouteFallback />;
  }

  return element ?? <DynamicMenuInner />;
}

export default function DynamicMenuPage() {
  return (
    <LegacyShell>
      <LegacyRouteSwitch />
    </LegacyShell>
  );
}
