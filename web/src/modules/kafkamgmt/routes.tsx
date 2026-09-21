import { lazy } from "react";
import type { RouteObject } from "react-router-dom";

const KafkamgmtConnectionsPage = lazy(() =>
  import("../../pages/kafkamgmt-connections-page").then((m) => ({ default: m.KafkamgmtConnectionsPage })),
);
const KafkamgmtTopicsPage = lazy(() =>
  import("../../pages/kafkamgmt-topics-page").then((m) => ({ default: m.KafkamgmtTopicsPage })),
);
const KafkamgmtGroupsPage = lazy(() =>
  import("../../pages/kafkamgmt-groups-page").then((m) => ({ default: m.KafkamgmtGroupsPage })),
);
const KafkamgmtBrokersPage = lazy(() =>
  import("../../pages/kafkamgmt-brokers-page").then((m) => ({ default: m.KafkamgmtBrokersPage })),
);

export const KAFKAMGMT_PLUGIN = "kafkamgmt";

export const kafkamgmtRoutes: RouteObject[] = [
  { path: "kafkamgmt/connections", element: <KafkamgmtConnectionsPage /> },
  { path: "kafkamgmt/topics", element: <KafkamgmtTopicsPage /> },
  { path: "kafkamgmt/groups", element: <KafkamgmtGroupsPage /> },
  { path: "kafkamgmt/brokers", element: <KafkamgmtBrokersPage /> },
];
