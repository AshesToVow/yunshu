// @ts-nocheck
import { lazy } from "react";
import type { RouteObject } from 'react-router';

const AiAssistantPage = lazy(() =>
  import("../../pages/ai-assistant-page").then((m) => ({ default: m.AiAssistantPage })),
);
const AiApprovalsPage = lazy(() =>
  import("../../pages/ai-approvals-page").then((m) => ({ default: m.AiApprovalsPage })),
);
const AiCenterPage = lazy(() =>
  import("../../pages/ai-center-page").then((m) => ({ default: m.AiCenterPage })),
);
const AiInvestigationsPage = lazy(() =>
  import("../../pages/ai-investigations-page").then((m) => ({ default: m.AiInvestigationsPage })),
);

export const AI_PLUGIN = "ai";

export const aiRoutes: RouteObject[] = [
  { path: "ai/assistant", element: <AiAssistantPage /> },
  { path: "ai/approvals", element: <AiApprovalsPage /> },
  { path: "ai/center", element: <AiCenterPage /> },
  { path: "ai/investigations", element: <AiInvestigationsPage /> },
];
