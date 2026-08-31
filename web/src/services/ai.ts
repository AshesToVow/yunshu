import { getData, http } from "./http";
import type { PodDiagnoseResult } from "./pods";

export type AIProviderName = "openai_compat" | "deepseek" | "anthropic" | string;

export interface AIProviderStatus {
  name: string;
  configured: boolean;
  base_url: string;
  model: string;
  source?: string;
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
  session_id?: number;
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
  rag_hits?: Array<{ source: string; module?: string; content: string; score?: number }>;
}

export interface AIChatSession {
  id: number;
  user_id: number;
  title: string;
  project_id?: number;
  cluster_id?: number;
  provider?: string;
  enable_tools: boolean;
  enable_write: boolean;
  last_message_at?: string;
  created_at?: string;
  updated_at?: string;
  message_count?: number;
}

export interface AIChatMessageRow {
  id: number;
  session_id: number;
  role: "user" | "assistant" | string;
  content: string;
  meta_json?: string;
  created_at?: string;
}

export interface AISessionDetail {
  session: AIChatSession;
  messages: AIChatMessageRow[];
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
  session_id?: number;
  project_id?: number;
  cluster_id?: number;
  namespace?: string;
  enable_tools?: boolean;
  enable_write_tools?: boolean;
  disable_rag?: boolean;
}) {
  return getData<AIChatResult>(http.post("/ai/chat", payload, { timeout: 120000 }));
}

export type AIChatStreamEvent = {
  type: "progress" | "rag" | "tool" | "reply" | "error" | "done" | string;
  message?: string;
  tool_step?: NonNullable<AIChatResult["tool_steps"]>[number];
  rag_hits?: AIChatResult["rag_hits"];
  reply?: string;
  session_id?: number;
  error?: string;
};

/** SSE 进度流（工具/RAG/回复），Cookie 会话鉴权。 */
export async function chatAIStream(
  payload: {
    provider?: string;
    messages: AIChatMessage[];
    session_id?: number;
    project_id?: number;
    cluster_id?: number;
    namespace?: string;
    enable_tools?: boolean;
    enable_write_tools?: boolean;
    disable_rag?: boolean;
  },
  onEvent: (ev: AIChatStreamEvent) => void,
  signal?: AbortSignal,
): Promise<void> {
  const res = await fetch("/api/v1/ai/chat/stream", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json", Accept: "text/event-stream" },
    body: JSON.stringify(payload),
    signal,
  });
  if (!res.ok) {
    let msg = `流式对话失败 (${res.status})`;
    try {
      const body = (await res.json()) as { message?: string; msg?: string };
      msg = body.message || body.msg || msg;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  if (!res.body) {
    throw new Error("浏览器不支持流式响应");
  }
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    const chunks = buf.split("\n\n");
    buf = chunks.pop() || "";
    for (const chunk of chunks) {
      const line = chunk
        .split("\n")
        .map((l) => l.trim())
        .find((l) => l.startsWith("data:"));
      if (!line) continue;
      const raw = line.replace(/^data:\s*/, "");
      if (!raw || raw === "[DONE]") continue;
      try {
        onEvent(JSON.parse(raw) as AIChatStreamEvent);
      } catch {
        /* ignore malformed */
      }
    }
  }
}

export type AIInvestigation = {
  id: number;
  user_id: number;
  kind: string;
  title: string;
  status: string;
  project_id?: number;
  cluster_id?: number;
  namespace?: string;
  resource?: string;
  fingerprint?: string;
  input_json?: string;
  collect_json?: string;
  analysis_json?: string;
  report_json?: string;
  error_msg?: string;
  session_id?: number;
  approval_id?: number;
  created_at?: string;
  updated_at?: string;
};

export type AIInvestigationReport = {
  summary?: string;
  root_causes?: Array<Record<string, unknown>>;
  actions?: Array<Record<string, unknown>>;
  evidence?: Array<Record<string, unknown>>;
  provider?: string;
  model?: string;
  raw_reply?: string;
};

export function startAIInvestigation(payload: {
  kind: "alert" | "pod" | "cicd" | "chat" | string;
  title?: string;
  provider?: string;
  project_id?: number;
  cluster_id?: number;
  namespace?: string;
  resource?: string;
  fingerprint?: string;
  run_id?: number;
  session_id?: number;
  query?: string;
}) {
  return getData<AIInvestigation>(http.post("/ai/investigations", payload, { timeout: 180000 }));
}

export function listAIInvestigations(params?: { kind?: string; status?: string; page?: number; page_size?: number }) {
  return getData<{ list: AIInvestigation[]; total: number; page: number; page_size: number }>(
    http.get("/ai/investigations", { params }),
  );
}

export function getAIInvestigation(id: number) {
  return getData<AIInvestigation>(http.get(`/ai/investigations/${id}`));
}

export function embedAIKnowledge(payload?: { limit?: number }) {
  return getData<{ updated?: number; skipped?: number; message?: string }>(
    http.post("/ai/knowledge/embed", payload || {}, { timeout: 300000 }),
  );
}

export function listAISessions(params?: { page?: number; page_size?: number }) {
  return getData<{ list: AIChatSession[]; total: number; page: number; page_size: number }>(
    http.get("/ai/sessions", { params }),
  );
}

export function createAISession(payload?: {
  title?: string;
  project_id?: number;
  cluster_id?: number;
  provider?: string;
  enable_tools?: boolean;
  enable_write?: boolean;
}) {
  return getData<AIChatSession>(http.post("/ai/sessions", payload || {}));
}

export function getAISession(id: number) {
  return getData<AISessionDetail>(http.get(`/ai/sessions/${id}`));
}

export function updateAISession(
  id: number,
  payload: {
    title?: string;
    project_id?: number;
    cluster_id?: number;
    provider?: string;
    enable_tools?: boolean;
    enable_write?: boolean;
  },
) {
  return getData<AIChatSession>(http.patch(`/ai/sessions/${id}`, payload));
}

export function deleteAISession(id: number) {
  return getData<{ ok: boolean }>(http.delete(`/ai/sessions/${id}`));
}

export function clearAISession(id: number) {
  return getData<{ ok: boolean }>(http.post(`/ai/sessions/${id}/clear`, {}));
}

export function analyzePodDiagnoseAI(payload: {
  provider?: string;
  cluster_id: number;
  namespace: string;
  name: string;
}) {
  return getData<AIPodDiagnoseResult>(http.post("/ai/k8s/pod-diagnose", payload, { timeout: 120000 }));
}

export type AIGenerateK8sYAMLResult = {
  yaml: string;
  provider: string;
  model: string;
  raw_reply?: string;
};

export function generateK8sYAML(payload: {
  provider?: string;
  resource_kind: string;
  namespace?: string;
  description: string;
  hint_yaml?: string;
  cluster_id?: number;
}) {
  return getData<AIGenerateK8sYAMLResult>(http.post("/ai/k8s/generate-yaml", payload, { timeout: 120000 }));
}

export function analyzeCicdBuildFailAI(payload: { provider?: string; project_id: number; run_id: number }) {
  return getData<AICicdBuildFailResult>(http.post("/ai/cicd/build-fail", payload, { timeout: 120000 }));
}

export function analyzeAlertExplainAI(payload: {
  provider?: string;
  fingerprint: string;
  project_id?: number;
  window_hours?: number;
}) {
  return getData<AIAlertExplainResult>(http.post("/ai/alert/explain", payload, { timeout: 120000 }));
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
  return getData<{ list: AIApprovalItem[]; total: number }>(http.get("/ai/approvals", { params }));
}

export function reviewAIApproval(id: number, payload: { approve: boolean; execute?: boolean; note?: string }) {
  return getData<AIApprovalItem>(http.post(`/ai/approvals/${id}/review`, payload));
}

export function executeAIApproval(id: number) {
  return getData<AIApprovalItem>(http.post(`/ai/approvals/${id}/execute`, {}));
}
