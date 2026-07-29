import type { PageData } from "../types/api";
import { getData, http } from "./http";

export interface ChangeEventItem {
  id: number;
  project_id: number;
  service_id?: number | null;
  source: string;
  action: string;
  risk_level: string;
  status: string;
  actor_user_id?: number | null;
  summary: string;
  payload_json?: string;
  started_at: string;
  finished_at?: string | null;
  rollback_ref?: string;
}

export async function listChangeEvents(
  projectId: number,
  params?: {
    service_id?: number;
    source?: string;
    status?: string;
    keyword?: string;
    from?: string;
    to?: string;
    page?: number;
    page_size?: number;
  },
) {
  return getData<PageData<ChangeEventItem>>(
    http.get(`/projects/${projectId}/change-events`, { params }),
  );
}
