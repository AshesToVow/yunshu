import { getData, http } from "./http";

export function getAICenterOverview() {
  return getData<Record<string, unknown>>(http.get("/ai/center/overview"));
}

export function reseedAICenter() {
  return getData<{
    ok: boolean;
    error?: string;
    report?: {
      data_root?: string;
      data_root_ok?: boolean;
      prompts?: number;
      knowledge_bases?: number;
      kb_documents?: number;
      cases?: number;
      sops?: number;
      script_tools?: number;
      builtin_tools?: number;
      eval_cases?: number;
      warnings?: string[];
    };
  }>(http.post("/ai/center/reseed", {}));
}

export function listAICenterPrompts() {
  return getData<{ list: Array<Record<string, unknown>> }>(http.get("/ai/center/prompts"));
}

export function listAICenterPromptVersions(id: number) {
  return getData<{ list: Array<Record<string, unknown>> }>(http.get(`/ai/center/prompts/${id}/versions`));
}

export function publishAICenterPrompt(id: number, payload: { content: string; changelog?: string }) {
  return getData<Record<string, unknown>>(http.post(`/ai/center/prompts/${id}/publish`, payload));
}

export function rollbackAICenterPrompt(id: number, versionId: number) {
  return getData<{ ok: boolean }>(http.post(`/ai/center/prompts/${id}/versions/${versionId}/rollback`, {}));
}

export function listAICenterModels() {
  return getData<{ list: AILLMModelItem[] }>(http.get("/ai/center/models"));
}

export interface AILLMModelItem {
  id: number;
  name: string;
  provider: string;
  base_url?: string;
  has_api_key?: boolean;
  model_name: string;
  model_type?: string;
  model_version?: string;
  temperature?: number;
  max_tokens?: number;
  context_length?: number;
  enabled?: boolean;
  is_default?: boolean;
  remark?: string;
}

export function createAICenterModel(payload: Record<string, unknown>) {
  return getData<AILLMModelItem>(http.post("/ai/center/models", payload));
}

export function updateAICenterModel(id: number, payload: Record<string, unknown>) {
  return getData<AILLMModelItem>(http.put(`/ai/center/models/${id}`, payload));
}

export function deleteAICenterModel(id: number) {
  return getData<{ ok: boolean }>(http.delete(`/ai/center/models/${id}`));
}

export function setDefaultAICenterModel(id: number) {
  return getData<{ ok: boolean }>(http.post(`/ai/center/models/${id}/default`, {}));
}

export function listAICenterTools() {
  return getData<{ list: Array<Record<string, unknown>> }>(http.get("/ai/center/tools"));
}

export function updateAICenterTool(id: number, enabled: boolean) {
  return getData<{ ok: boolean }>(http.patch(`/ai/center/tools/${id}`, { enabled }));
}

export function listAICenterCases() {
  return getData<{ list: Array<Record<string, unknown>> }>(http.get("/ai/center/cases"));
}

export function listAICenterSOPs() {
  return getData<{ list: Array<Record<string, unknown>> }>(http.get("/ai/center/sops"));
}

export function listAICenterKBs() {
  return getData<{ list: Array<Record<string, unknown>> }>(http.get("/ai/center/knowledge-bases"));
}

export function listAIEvalCases() {
  return getData<{ list: Array<Record<string, unknown>> }>(http.get("/ai/center/eval/cases"));
}

export function runAIEval(live = false) {
  return getData<Record<string, unknown>>(http.post("/ai/center/eval/run", { live }, { timeout: 300000 }));
}

export function syncAIKnowledge() {
  return getData<{ indexed: number; failed?: number; errors?: string[] }>(http.post("/ai/knowledge/sync", {}));
}
