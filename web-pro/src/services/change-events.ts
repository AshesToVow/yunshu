// @ts-nocheck
/** 变更事件条目（服务画像 / 告警质量等聚合视图复用）。 */
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
