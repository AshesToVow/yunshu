import { Spin } from "antd";
import { Suspense, lazy, useMemo } from "react";
import { Navigate, useLocation, useRoutes, type RouteObject } from "react-router-dom";
import { useAuth } from "../contexts/auth-context";
import { usePlugins } from "../contexts/plugin-context";
import { collectModuleRoutes } from "../modules";
import { DynamicMenuPage } from "../pages/dynamic-menu-page";

const AdminLayout = lazy(() => import("../layouts/admin-layout").then((module) => ({ default: module.AdminLayout })));
const LoginPage = lazy(() => import("../pages/login-page").then((module) => ({ default: module.LoginPage })));

function RouteFallback() {
  return (
    <div className="full-screen-loading">
      <Spin size="large" />
    </div>
  );
}

function ProtectedLayout() {
  const { isAuthenticated, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return <RouteFallback />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return (
    <Suspense fallback={<RouteFallback />}>
      <AdminLayout />
    </Suspense>
  );
}

function AuthLayout() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return <RouteFallback />;
  }

  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return (
    <Suspense fallback={<RouteFallback />}>
      <LoginPage />
    </Suspense>
  );
}

/** 使用 useRoutes 注册嵌套路由（不可在子组件内动态返回 &lt;Route&gt;，否则会触发 matched 未定义） */
export function AppRoutes() {
  const { isAuthenticated, loading: authLoading } = useAuth();
  const { loading: pluginsLoading, isPluginEnabled } = usePlugins();

  const routes = useMemo((): RouteObject[] => {
    const children: RouteObject[] = [
      ...collectModuleRoutes(isPluginEnabled),
      { path: "*", element: <DynamicMenuPage /> },
    ];
    return [
      { path: "/login", element: <AuthLayout /> },
      {
        path: "/",
        element: <ProtectedLayout />,
        children,
      },
    ];
  }, [isPluginEnabled]);

  const element = useRoutes(routes);

  if (authLoading || (isAuthenticated && pluginsLoading)) {
    return <RouteFallback />;
  }

  return element ?? <RouteFallback />;
}
