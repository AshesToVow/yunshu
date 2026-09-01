import { lazy } from "react";
import { Navigate, type RouteObject } from "react-router-dom";

const DashboardPage = lazy(() => import("../../pages/dashboard-page").then((m) => ({ default: m.DashboardPage })));
const UsersPage = lazy(() => import("../../pages/users-page").then((m) => ({ default: m.UsersPage })));
const DepartmentsPage = lazy(() => import("../../pages/departments-page").then((m) => ({ default: m.DepartmentsPage })));
const RolesPage = lazy(() => import("../../pages/roles-page").then((m) => ({ default: m.RolesPage })));
const PermissionsPage = lazy(() => import("../../pages/permissions-page").then((m) => ({ default: m.PermissionsPage })));
const PoliciesPage = lazy(() => import("../../pages/policies-page").then((m) => ({ default: m.PoliciesPage })));
const RegistrationsPage = lazy(() => import("../../pages/registrations-page").then((m) => ({ default: m.RegistrationsPage })));
const MenusPage = lazy(() => import("../../pages/menus-page").then((m) => ({ default: m.MenusPage })));
const LoginLogsPage = lazy(() => import("../../pages/login-logs-page").then((m) => ({ default: m.LoginLogsPage })));
const OperationLogsPage = lazy(() =>
  import("../../pages/operation-logs-page").then((m) => ({ default: m.OperationLogsPage })),
);
const BannedIPsPage = lazy(() => import("../../pages/banned-ips-page").then((m) => ({ default: m.BannedIPsPage })));
const PersonalSettingsPage = lazy(() =>
  import("../../pages/personal-settings-page").then((m) => ({ default: m.PersonalSettingsPage })),
);
const PluginsPage = lazy(() => import("../../pages/plugins-page").then((m) => ({ default: m.PluginsPage })));

export const CORE_PLUGIN = "core";

export const coreRoutes: RouteObject[] = [
  { index: true, element: <DashboardPage /> },
  { path: "users", element: <UsersPage /> },
  { path: "departments", element: <DepartmentsPage /> },
  { path: "roles", element: <RolesPage /> },
  { path: "permissions", element: <PermissionsPage /> },
  { path: "policies", element: <PoliciesPage /> },
  { path: "registrations", element: <RegistrationsPage /> },
  { path: "menus", element: <MenusPage /> },
  { path: "login-logs", element: <LoginLogsPage /> },
  { path: "operation-logs", element: <OperationLogsPage /> },
  { path: "banned-ips", element: <BannedIPsPage /> },
  { path: "personal-settings", element: <PersonalSettingsPage /> },
  { path: "plugins", element: <PluginsPage /> },
  { path: "runtime-config", element: <Navigate to="/dict-entries" replace /> },
];
