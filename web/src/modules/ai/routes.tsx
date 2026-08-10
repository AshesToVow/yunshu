import { lazy } from "react";
import type { RouteObject } from "react-router-dom";

const AiAssistantPage = lazy(() =>
  import("../../pages/ai-assistant-page").then((m) => ({ default: m.AiAssistantPage })),
);
const AiApprovalsPage = lazy(() =>
  import("../../pages/ai-approvals-page").then((m) => ({ default: m.AiApprovalsPage })),
);

export const AI_PLUGIN = "ai";

export const aiRoutes: RouteObject[] = [
  { path: "ai/assistant", element: <AiAssistantPage /> },
  { path: "ai/approvals", element: <AiApprovalsPage /> },
];
