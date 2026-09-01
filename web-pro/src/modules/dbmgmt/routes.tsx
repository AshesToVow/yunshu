// @ts-nocheck
import { lazy } from 'react';
import { Navigate } from '@umijs/max';
import type { RouteObject } from 'react-router';

const DbmgmtInstancesPage = lazy(() => import('../../pages/dbmgmt/instances'));
const DbmgmtInstanceDetailPage = lazy(() => import('../../pages/dbmgmt/instance-detail'));
const DbmgmtSqlQueryPage = lazy(() => import('../../pages/dbmgmt/sql-query'));
const DbmgmtSqlAuditPage = lazy(() => import('../../pages/dbmgmt/sql-audit'));
const DbmgmtQueryApplyPage = lazy(() => import('../../pages/dbmgmt/apply-query'));
const DbmgmtDatabaseApplyPage = lazy(() => import('../../pages/dbmgmt/apply-database'));
const DbmgmtAppUserApplyPage = lazy(() => import('../../pages/dbmgmt/apply-app-user'));
const DbmgmtWorkflowHistoryPage = lazy(() => import('../../pages/dbmgmt/tickets'));
const DbmgmtTicketDetailPage = lazy(() => import('../../pages/dbmgmt/ticket-detail'));
const DbmgmtAuditPage = lazy(() => import('../../pages/dbmgmt/audit'));
const DbmgmtGrantsPage = lazy(() => import('../../pages/dbmgmt/grants'));

export const DBMGMT_PLUGIN = 'dbmgmt';

export const dbmgmtRoutes: RouteObject[] = [
  { path: 'dbmgmt/instances', element: <DbmgmtInstancesPage /> },
  { path: 'dbmgmt/instances/:instanceId', element: <DbmgmtInstanceDetailPage /> },
  { path: 'dbmgmt/apply/database', element: <DbmgmtDatabaseApplyPage /> },
  { path: 'dbmgmt/apply/query', element: <DbmgmtQueryApplyPage /> },
  { path: 'dbmgmt/apply/app-user', element: <DbmgmtAppUserApplyPage /> },
  { path: 'dbmgmt/apply/query-grants', element: <DbmgmtGrantsPage /> },
  { path: 'dbmgmt/sql/query', element: <DbmgmtSqlQueryPage /> },
  { path: 'dbmgmt/sql/audit', element: <DbmgmtSqlAuditPage /> },
  { path: 'dbmgmt/workflow/pending', element: <Navigate to="/workflow/inbox?domain=dbmgmt" replace /> },
  { path: 'dbmgmt/workflow/history', element: <DbmgmtWorkflowHistoryPage /> },
  { path: 'dbmgmt/workflow/tickets/:ticketId', element: <DbmgmtTicketDetailPage /> },
  { path: 'dbmgmt/approval-flow', element: <Navigate to="/workflow/definitions?domain=dbmgmt" replace /> },
  { path: 'dbmgmt/audit', element: <DbmgmtAuditPage /> },
  { path: 'dbmgmt/grants', element: <DbmgmtGrantsPage /> },
  // 旧路径兼容
  { path: 'dbmgmt/console', element: <Navigate to="/dbmgmt/sql/query" replace /> },
  { path: 'dbmgmt/access-requests', element: <Navigate to="/dbmgmt/apply/query" replace /> },
  { path: 'dbmgmt/access-request', element: <Navigate to="/dbmgmt/apply/query" replace /> },
  { path: 'dbmgmt/access-requests/all', element: <Navigate to="/dbmgmt/apply/query" replace /> },
  { path: 'dbmgmt/todo', element: <Navigate to="/workflow/inbox?domain=dbmgmt" replace /> },
  { path: 'dbmgmt/tickets', element: <Navigate to="/dbmgmt/workflow/history" replace /> },
];
