import { lazy } from "react";
import type { RouteObject } from "react-router-dom";

const EsmgmtConnectionsPage = lazy(() =>
  import("../../pages/esmgmt-connections-page").then((m) => ({ default: m.EsmgmtConnectionsPage })),
);
const EsmgmtOverviewPage = lazy(() =>
  import("../../pages/esmgmt-overview-page").then((m) => ({ default: m.EsmgmtOverviewPage })),
);
const EsmgmtConsolePage = lazy(() =>
  import("../../pages/esmgmt-console-page").then((m) => ({ default: m.EsmgmtConsolePage })),
);

const EsmgmtStoragePage = lazy(() =>
  import("../../pages/esmgmt-storage-page").then((m) => ({ default: m.EsmgmtStoragePage })),
);
const EsmgmtBackupsPage = lazy(() =>
  import("../../pages/esmgmt-backups-page").then((m) => ({ default: m.EsmgmtBackupsPage })),
);
const EsmgmtDocsPage = lazy(() =>
  import("../../pages/esmgmt-docs-page").then((m) => ({ default: m.EsmgmtDocsPage })),
);
const EsmgmtTemplatesPage = lazy(() =>
  import("../../pages/esmgmt-templates-page").then((m) => ({ default: m.EsmgmtTemplatesPage })),
);
const EsmgmtReindexPage = lazy(() =>
  import("../../pages/esmgmt-reindex-page").then((m) => ({ default: m.EsmgmtReindexPage })),
);

export const ESMGMT_PLUGIN = "esmgmt";

export const esmgmtRoutes: RouteObject[] = [
  { path: "esmgmt/connections", element: <EsmgmtConnectionsPage /> },
  { path: "esmgmt/storage", element: <EsmgmtStoragePage /> },
  { path: "esmgmt/overview", element: <EsmgmtOverviewPage /> },
  { path: "esmgmt/backups", element: <EsmgmtBackupsPage /> },
  { path: "esmgmt/console", element: <EsmgmtConsolePage /> },
  { path: "esmgmt/docs", element: <EsmgmtDocsPage /> },
  { path: "esmgmt/templates", element: <EsmgmtTemplatesPage /> },
  { path: "esmgmt/reindex", element: <EsmgmtReindexPage /> },
];
