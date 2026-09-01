// @ts-nocheck
import { lazy } from "react";
import { Navigate } from '@umijs/max';
import type { RouteObject } from 'react-router';

const CicdServicesPage = lazy(() =>
  import("../../pages/cicd-services-page").then((m) => ({ default: m.CicdServicesPage })),
);
const CicdBuildRecordsPage = lazy(() =>
  import("../../pages/cicd-build-records-page").then((m) => ({ default: m.CicdBuildRecordsPage })),
);
const CicdReleaseRecordsPage = lazy(() =>
  import("../../pages/cicd-release-records-page").then((m) => ({ default: m.CicdReleaseRecordsPage })),
);
const CicdRegistriesPage = lazy(() =>
  import("../../pages/cicd-registries-page").then((m) => ({ default: m.CicdRegistriesPage })),
);
const CicdImageBrowserPage = lazy(() =>
  import("../../pages/cicd-image-browser-page").then((m) => ({ default: m.CicdImageBrowserPage })),
);

export const CICD_PLUGIN = "cicd";

export const cicdRoutes: RouteObject[] = [
  { path: "cicd/services", element: <CicdServicesPage /> },
  { path: "cicd/todo", element: <Navigate to="/workflow/inbox?domain=cicd" replace /> },
  { path: "cicd/approval-flow", element: <Navigate to="/workflow/definitions?domain=cicd" replace /> },
  { path: "cicd/build-records", element: <CicdBuildRecordsPage /> },
  { path: "cicd/release-records", element: <CicdReleaseRecordsPage /> },
  { path: "cicd/registries", element: <CicdRegistriesPage /> },
  { path: "cicd/image-browser", element: <CicdImageBrowserPage /> },
];
