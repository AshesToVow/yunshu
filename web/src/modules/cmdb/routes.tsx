import { lazy } from "react";
import type { RouteObject } from "react-router-dom";

const ServerConsolePage = lazy(() =>
  import("../../pages/server-console-page").then((m) => ({ default: m.ServerConsolePage })),
);

export const CMDB_PLUGIN = "cmdb";

export const cmdbRoutes: RouteObject[] = [{ path: "server-console", element: <ServerConsolePage /> }];
