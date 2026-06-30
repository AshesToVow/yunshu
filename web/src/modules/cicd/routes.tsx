import { lazy } from "react";
import type { RouteObject } from "react-router-dom";

const CicdServicesPage = lazy(() =>
  import("../../pages/cicd-services-page").then((m) => ({ default: m.CicdServicesPage })),
);
const CicdBuildRecordsPage = lazy(() =>
  import("../../pages/cicd-build-records-page").then((m) => ({ default: m.CicdBuildRecordsPage })),
);
const CicdReleaseRecordsPage = lazy(() =>
  import("../../pages/cicd-release-records-page").then((m) => ({ default: m.CicdReleaseRecordsPage })),
);
const CicdTodoPage = lazy(() =>
  import("../../pages/cicd-todo-page").then((m) => ({ default: m.CicdTodoPage })),
);
const CicdApprovalFlowPage = lazy(() =>
  import("../../pages/cicd-approval-flow-page").then((m) => ({ default: m.CicdApprovalFlowPage })),
);

export const CICD_PLUGIN = "cicd";

export const cicdRoutes: RouteObject[] = [
  { path: "cicd/services", element: <CicdServicesPage /> },
  { path: "cicd/todo", element: <CicdTodoPage /> },
  { path: "cicd/approval-flow", element: <CicdApprovalFlowPage /> },
  { path: "cicd/build-records", element: <CicdBuildRecordsPage /> },
  { path: "cicd/release-records", element: <CicdReleaseRecordsPage /> },
];
