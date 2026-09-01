// @ts-nocheck
import { lazy } from "react";
import { Navigate, useLocation } from '@umijs/max';
import type { RouteObject } from 'react-router';

const ServerConsolePage = lazy(() =>
  import("../../pages/server-console-page").then((m) => ({ default: m.ServerConsolePage })),
);

export const CMDB_PLUGIN = "cmdb";

function ServerConsoleUnderscoreRedirect() {
  const { search } = useLocation();
  return <Navigate to={`/server-console${search}`} replace />;
}

export const cmdbRoutes: RouteObject[] = [
  { path: "server-console", element: <ServerConsolePage /> },
  { path: "server_console", element: <ServerConsoleUnderscoreRedirect /> },
];
