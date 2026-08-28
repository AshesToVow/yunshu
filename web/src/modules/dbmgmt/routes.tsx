import { lazy } from "react";
import { Navigate, type RouteObject } from "react-router-dom";

const DbmgmtInstancesPage = lazy(() =>
  import("../../pages/dbmgmt-instances-page").then((m) => ({ default: m.DbmgmtInstancesPage })),
);
const DbmgmtInstanceDetailPage = lazy(() =>
  import("../../pages/dbmgmt-instance-detail-page").then((m) => ({ default: m.DbmgmtInstanceDetailPage })),
);
const DbmgmtSqlQueryPage = lazy(() =>
  import("../../pages/dbmgmt-console-page").then((m) => ({ default: m.DbmgmtSqlQueryPage })),
);
const DbmgmtSqlAuditPage = lazy(() =>
  import("../../pages/dbmgmt-console-page").then((m) => ({ default: m.DbmgmtSqlAuditPage })),
);
const DbmgmtQueryApplyPage = lazy(() =>
  import("../../pages/dbmgmt-access-requests-page").then((m) => ({ default: m.DbmgmtQueryApplyPage })),
);
const DbmgmtDatabaseApplyPage = lazy(() =>
  import("../../pages/dbmgmt-database-apply-page").then((m) => ({ default: m.DbmgmtDatabaseApplyPage })),
);
const DbmgmtAppUserApplyPage = lazy(() =>
  import("../../pages/dbmgmt-app-user-apply-page").then((m) => ({ default: m.DbmgmtAppUserApplyPage })),
);
const DbmgmtWorkflowHistoryPage = lazy(() =>
  import("../../pages/dbmgmt-tickets-page").then((m) => ({ default: m.DbmgmtWorkflowHistoryPage })),
);
const DbmgmtTicketDetailPage = lazy(() =>
  import("../../pages/dbmgmt-ticket-detail-page").then((m) => ({ default: m.DbmgmtTicketDetailPage })),
);
const DbmgmtAuditPage = lazy(() =>
  import("../../pages/dbmgmt-audit-page").then((m) => ({ default: m.DbmgmtAuditPage })),
);
const DbmgmtGrantsPage = lazy(() =>
  import("../../pages/dbmgmt-grants-page").then((m) => ({ default: m.DbmgmtGrantsPage })),
);
const DbmgmtQueryGrantsPage = lazy(() =>
  import("../../pages/dbmgmt-grants-page").then((m) => ({ default: m.DbmgmtQueryGrantsPage })),
);

export const DBMGMT_PLUGIN = "dbmgmt";

export const dbmgmtRoutes: RouteObject[] = [
  { path: "dbmgmt/instances", element: <DbmgmtInstancesPage /> },
  { path: "dbmgmt/instances/:instanceId", element: <DbmgmtInstanceDetailPage /> },
  { path: "dbmgmt/apply/database", element: <DbmgmtDatabaseApplyPage /> },
  { path: "dbmgmt/apply/query", element: <DbmgmtQueryApplyPage /> },
  { path: "dbmgmt/apply/app-user", element: <DbmgmtAppUserApplyPage /> },
  { path: "dbmgmt/apply/query-grants", element: <DbmgmtQueryGrantsPage /> },
  { path: "dbmgmt/sql/query", element: <DbmgmtSqlQueryPage /> },
  { path: "dbmgmt/sql/audit", element: <DbmgmtSqlAuditPage /> },
  { path: "dbmgmt/workflow/pending", element: <Navigate to="/workflow/inbox?domain=dbmgmt" replace /> },
  { path: "dbmgmt/workflow/history", element: <DbmgmtWorkflowHistoryPage /> },
  { path: "dbmgmt/workflow/tickets/:ticketId", element: <DbmgmtTicketDetailPage /> },
  { path: "dbmgmt/approval-flow", element: <Navigate to="/workflow/definitions?domain=dbmgmt" replace /> },
  { path: "dbmgmt/audit", element: <DbmgmtAuditPage /> },
  { path: "dbmgmt/grants", element: <DbmgmtGrantsPage /> },
  // 旧路径兼容
  { path: "dbmgmt/console", element: <Navigate to="/dbmgmt/sql/query" replace /> },
  { path: "dbmgmt/access-requests", element: <Navigate to="/dbmgmt/apply/query" replace /> },
  { path: "dbmgmt/access-request", element: <Navigate to="/dbmgmt/apply/query" replace /> },
  { path: "dbmgmt/access-requests/all", element: <Navigate to="/dbmgmt/apply/query" replace /> },
  { path: "dbmgmt/todo", element: <Navigate to="/workflow/inbox?domain=dbmgmt" replace /> },
  { path: "dbmgmt/tickets", element: <Navigate to="/dbmgmt/workflow/history" replace /> },
];
