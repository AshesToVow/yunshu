import { getData, http } from "./http";
import type { PodDiagnoseResult } from "./pods";

export type AIProviderName = "openai_compat" | "deepseek" | "anthropic" | string;

export interface AIProviderStatus {
  name: string;
  configured: boolean;
  base_url: string;
  model: string;
}

export interface AIStatus {
  enabled: boolean;
  default_provider: string;
  timeout_sec: number;
  max_tokens: number;
  providers: AIProviderStatus[];
}

export interface AIPingResult {
  ok: boolean;
  provider: string;
  model: string;
  reply?: string;
  message?: string;
}

export interface AIChatMessage {
  role: "user" | "assistant";
  content: string;
}

export interface AIChatResult {
  reply: string;
  provider: string;
  model: string;
  usage?: {
    prompt_tokens?: number;
    completion_tokens?: number;
    total_tokens?: number;
  };
  tool_steps?: Array<{
    name: string;
    args?: string;
    result?: string;
    ok?: boolean;
    error?: string;
  }>;
  rag_hits?: Array<{ source: string; content: string; score?: number }>;
}

export interface AIPodDiagnoseResult {
  diagnose: PodDiagnoseResult;
  ai_summary: string;
  root_causes: Array<Record<string, unknown>>;
  actions: Array<Record<string, unknown>>;
  raw_reply?: string;
  provider: string;
  model: string;
}

export interface AIAnalysisCommon {
  ai_summary: string;
  root_causes: Array<Record<string, unknown>>;
  actions: Array<Record<string, unknown>>;
  raw_reply?: string;
  provider: string;
  model: string;
}

export type AICicdBuildFailResult = AIAnalysisCommon & {
  build?: Record<string, unknown>;
};

export type AIAlertExplainResult = AIAnalysisCommon & {
  explain?: Record<string, unknown>;
};

export function getAIStatus() {
  return getData<AIStatus>(http.get("/ai/status"));
}

export function pingAI(provider?: string) {
  return getData<AIPingResult>(http.post("/ai/ping", { provider: provider || "" }));
}

export function chatAI(payload: {
  provider?: string;
  messages: AIChatMessage[];
  project_id?: number;
  cluster_id?: number;
  namespace?: string;
  enable_tools?: boolean;
  enable_write_tools?: boolean;
  disable_rag?: boolean;
}) {
  return getData<AIChatResult>(
    http.post("/ai/chat", payload, { timeout: 120000 }),
  );
}

export function analyzePodDiagnoseAI(payload: {
  provider?: string;
  cluster_id: number;
  namespace: string;
  name: string;
}) {
  return getData<AIPodDiagnoseResult>(
    http.post("/ai/k8s/pod-diagnose", payload, { timeout: 120000 }),
  );
}

export function analyzeCicdBuildFailAI(payload: {
  provider?: string;
  project_id: number;
  run_id: number;
}) {
  return getData<AICicdBuildFailResult>(
    http.post("/ai/cicd/build-fail", payload, { timeout: 120000 }),
  );
}

export function analyzeAlertExplainAI(payload: {
  provider?: string;
  fingerprint: string;
  project_id?: number;
  window_hours?: number;
}) {
  return getData<AIAlertExplainResult>(
    http.post("/ai/alert/explain", payload, { timeout: 120000 }),
  );
}

export interface AIApprovalItem {
  id: number;
  user_id: number;
  tool_name: string;
  args_json?: string;
  cluster_id?: number;
  namespace?: string;
  resource?: string;
  reason?: string;
  status: string;
  review_note?: string;
  result_msg?: string;
  created_at?: string;
}

export function listAIApprovals(params?: { status?: string; page?: number; page_size?: number }) {
  return getData<{ list: AIApprovalItem[]; total: number }>(
    http.get("/ai/approvals", { params }),
  );
}

export function reviewAIApproval(
  id: number,
  payload: { approve: boolean; execute?: boolean; note?: string },
) {
  return getData<AIApprovalItem>(http.post(`/ai/approvals/${id}/review`, payload));
}

export function executeAIApproval(id: number) {
  return getData<AIApprovalItem>(http.post(`/ai/approvals/${id}/execute`, {}));
}
