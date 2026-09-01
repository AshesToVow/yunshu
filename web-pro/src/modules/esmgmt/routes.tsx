// @ts-nocheck
import { lazy } from "react";
import type { RouteObject } from 'react-router';

const EsmgmtConnectionsPage = lazy(() =>
  import("../../pages/esmgmt-connections-page").then((m) => ({ default: m.EsmgmtConnectionsPage })),
);
const EsmgmtOverviewPage = lazy(() =>
  import("../../pages/esmgmt-overview-page").then((m) => ({ default: m.EsmgmtOverviewPage })),
);
const EsmgmtConsolePage = lazy(() =>
  import("../../pages/esmgmt-console-page").then((m) => ({ default: m.EsmgmtConsolePage })),
);

export const ESMGMT_PLUGIN = "esmgmt";

export const esmgmtRoutes: RouteObject[] = [
  { path: "esmgmt/connections", element: <EsmgmtConnectionsPage /> },
  { path: "esmgmt/overview", element: <EsmgmtOverviewPage /> },
  { path: "esmgmt/console", element: <EsmgmtConsolePage /> },
];
