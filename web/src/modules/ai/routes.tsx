import { lazy } from "react";
import type { RouteObject } from "react-router-dom";

const AiAssistantPage = lazy(() =>
  import("../../pages/ai-assistant-page").then((m) => ({ default: m.AiAssistantPage })),
);

export const AI_PLUGIN = "ai";

export const aiRoutes: RouteObject[] = [
  { path: "ai/assistant", element: <AiAssistantPage /> },
];
