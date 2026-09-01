import { getData, http } from "./http";

export interface IngressDiagnoseCheck {
  level: string;
  code: string;
  message: string;
}

export interface IngressDiagnoseResult {
  checks: IngressDiagnoseCheck[];
}

export function diagnoseIngress(params: { cluster_id: number; namespace: string; name: string }) {
  return getData<IngressDiagnoseResult>(http.get("/ingresses/diagnose", { params }));
}
