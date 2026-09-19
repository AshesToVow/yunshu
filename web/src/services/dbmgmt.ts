import { getData, http } from "./http";

export type DbInstance = {
  id: number;
  project_id: number;
  name: string;
  env: string;
  driver: string;
  connect_mode: string;
  host: string;
  port: number;
  database?: string;
  server_id?: number;
  server_name?: string;
  username: string;
  ssl_mode?: string;
  read_only: boolean;
  role?: "primary" | "replica";
  primary_instance_id?: number;
  primary_instance_name?: string;
  require_ticket_for_dml: boolean;
  owner_user_id?: number;
  status: string;
  last_ping_at?: string;
  last_ping_ok: boolean;
  tags?: string;
  remark?: string;
  created_at?: string;
  updated_at?: string;
  backup_link?: string;
};

export type DbInstancePayload = {
  name: string;
  env?: string;
  driver?: string;
  connect_mode?: string;
  host?: string;
  port?: number;
  database?: string;
  server_id?: number;
  username: string;
  password?: string;
  ssl_mode?: string;
  read_only?: boolean;
  role?: "primary" | "replica";
  primary_instance_id?: number;
  require_ticket_for_dml?: boolean;
  owner_user_id?: number;
  tags?: string;
  remark?: string;
};

function base(projectId: number) {
  return `/projects/${projectId}/dbmgmt`;
}

export async function listDbInstances(projectId: number, params?: Record<string, string | number>) {
  return getData<{ list: DbInstance[]; total: number }>(http.get(`${base(projectId)}/instances`, { params }));
}

export async function getDbInstance(projectId: number, instanceId: number) {
  return getData<DbInstance>(http.get(`${base(projectId)}/instances/${instanceId}`));
}

export async function createDbInstance(projectId: number, payload: DbInstancePayload) {
  return getData<DbInstance>(http.post(`${base(projectId)}/instances`, payload));
}

export async function updateDbInstance(projectId: number, instanceId: number, payload: DbInstancePayload) {
  return getData<DbInstance>(http.put(`${base(projectId)}/instances/${instanceId}`, payload));
}

export async function deleteDbInstance(projectId: number, instanceId: number) {
  return getData<{ ok?: boolean }>(http.delete(`${base(projectId)}/instances/${instanceId}`));
}

export async function pingDbInstance(projectId: number, instanceId: number) {
  return getData<{ ok: boolean; message: string }>(http.post(`${base(projectId)}/instances/${instanceId}/ping`));
}

export async function listDbDatabases(projectId: number, instanceId: number) {
  return getData<{ name: string }[]>(http.get(`${base(projectId)}/instances/${instanceId}/metadata/databases`));
}

export async function listDbColumns(projectId: number, instanceId: number, database: string, table: string) {
  return getData<{ name: string; data_type: string; nullable: boolean; default?: string; comment?: string }[]>(
    http.get(`${base(projectId)}/instances/${instanceId}/metadata/columns`, { params: { database, table } }),
  );
}

export async function listDbTicketSteps(projectId: number, ticketId: number) {
  return getData<DbTicketStep[]>(http.get(`${base(projectId)}/tickets/${ticketId}/steps`));
}

export type DbTicketStep = {
  id: number;
  stage_key: string;
  stage_name: string;
  sort_order: number;
  status: string;
  reviewer_name?: string;
  review_comment?: string;
  reviewed_at?: string;
  activated_at?: string;
  created_at: string;
};

export async function listDbTables(projectId: number, instanceId: number, database: string) {
  return getData<{ name: string }[]>(
    http.get(`${base(projectId)}/instances/${instanceId}/metadata/tables`, { params: { database } }),
  );
}

export async function queryDb(projectId: number, instanceId: number, body: { database?: string; sql: string }) {
  return getData<{ columns: string[]; rows: unknown[][]; row_count: number; duration_ms: number; truncated: boolean }>(
    http.post(`${base(projectId)}/instances/${instanceId}/query`, body),
  );
}

export type DbSubmitOptions = {
  database?: string;
  sql: string;
  reason?: string;
  audit_mode?: "system" | "manual";
  is_backup?: boolean;
  sql_file_ref?: string;
};

export async function executeDb(projectId: number, instanceId: number, body: DbSubmitOptions) {
  return getData<{ status: string; ticket_id?: number; risk_level?: string; message?: string; rows_affected?: number }>(
    http.post(`${base(projectId)}/instances/${instanceId}/execute`, body),
  );
}

export type DbSqlCheckRow = {
  order_id: number;
  stage: string;
  error_level: number;
  stage_status: string;
  error_message: string;
  sql: string;
  affected_rows?: number;
};

export type DbSqlCheckResult = {
  checked: boolean;
  goinception: boolean;
  syntax_type: number;
  error_count: number;
  warning_count: number;
  risk_level: string;
  rows?: DbSqlCheckRow[];
  error?: string;
};

export async function checkDbSql(
  projectId: number,
  instanceId: number,
  body: { database?: string; sql: string; audit_mode?: "system" | "manual" },
) {
  return getData<DbSqlCheckResult>(http.post(`${base(projectId)}/instances/${instanceId}/check`, body));
}

export async function importDbSql(projectId: number, instanceId: number, body: DbSubmitOptions) {
  return getData<{ status: string; ticket_id?: number; risk_level?: string; message?: string }>(
    http.post(`${base(projectId)}/instances/${instanceId}/import`, body),
  );
}

export async function listDbGrants(projectId: number, instanceId?: number) {
  return getData<DbGrant[]>(
    http.get(`${base(projectId)}/grants`, { params: instanceId ? { instance_id: instanceId } : undefined }),
  );
}

export async function createDbGrant(projectId: number, payload: DbGrantPayload) {
  return getData<DbGrant>(http.post(`${base(projectId)}/grants`, payload));
}

export async function deleteDbGrant(projectId: number, grantId: number) {
  return getData<{ ok?: boolean }>(http.delete(`${base(projectId)}/grants/${grantId}`));
}

export async function updateDbGrant(
  projectId: number,
  grantId: number,
  payload: { query_limit_num?: number; expires_at?: string; remark?: string },
) {
  return getData<DbGrant>(http.put(`${base(projectId)}/grants/${grantId}`, payload));
}

export async function getEffectiveDbPermission(projectId: number, instanceId: number) {
  return getData<DbEffectivePermission>(
    http.get(`${base(projectId)}/grants/effective`, { params: { instance_id: instanceId } }),
  );
}

export type DbGrant = {
  id: number;
  instance_id: number;
  principal_kind: string;
  principal_ref: string;
  database_name?: string;
  table_names?: string[];
  privileges?: string[];
  query_limit_num?: number;
  can_connect: boolean;
  can_query: boolean;
  can_dml: boolean;
  can_ddl: boolean;
  can_export: boolean;
  can_import: boolean;
  can_manage: boolean;
  expires_at?: string;
  remark?: string;
};

export type DbGrantPayload = {
  instance_id: number;
  principal_kind: string;
  principal_ref: string;
  database_name?: string;
  table_names?: string[];
  privileges?: string[];
  can_connect?: boolean;
  can_query?: boolean;
  can_dml?: boolean;
  can_ddl?: boolean;
  can_export?: boolean;
  can_import?: boolean;
  can_manage?: boolean;
  expires_at?: string;
  remark?: string;
};

export type DbEffectivePermission = {
  can_connect: boolean;
  can_query: boolean;
  can_dml: boolean;
  can_ddl: boolean;
  can_export: boolean;
  can_import: boolean;
  can_manage: boolean;
  databases?: string[];
};

export async function getDbApprovalFlow(projectId: number) {
  return getData<{ project_id: number; stages: DbApprovalStage[] }>(http.get(`${base(projectId)}/approval-flow`));
}

export async function saveDbApprovalFlow(
  projectId: number,
  stages: {
    stage_key: string;
    stage_name?: string;
    sort_order?: number;
    enabled: boolean;
    user_group_id?: number;
  }[],
) {
  return getData<{ project_id: number; stages: DbApprovalStage[] }>(
    http.put(`${base(projectId)}/approval-flow`, { stages }),
  );
}

export type DbApprovalStage = {
  stage_key: string;
  stage_name: string;
  sort_order: number;
  enabled: boolean;
  user_group_id?: number;
  user_group_name?: string;
};

export async function listDbAccessRequests(projectId: number, params?: Record<string, string | number>) {
  return getData<{ list: DbAccessRequest[]; total: number }>(
    http.get(`${base(projectId)}/access-requests`, { params }),
  );
}

export async function createDbAccessRequest(projectId: number, payload: DbAccessRequestPayload) {
  return getData<DbAccessRequest>(http.post(`${base(projectId)}/access-requests`, payload));
}

export async function approveDbAccessRequest(
  projectId: number,
  requestId: number,
  payload?: { comment?: string; expires_at?: string | null },
) {
  return getData<{ ok?: boolean }>(http.post(`${base(projectId)}/access-requests/${requestId}/approve`, payload ?? {}));
}

export async function rejectDbAccessRequest(projectId: number, requestId: number, comment?: string) {
  return getData<{ ok?: boolean }>(http.post(`${base(projectId)}/access-requests/${requestId}/reject`, { comment }));
}

export type DbAccessRequest = {
  id: number;
  instance_id: number;
  instance_name?: string;
  requester_name: string;
  database_name?: string;
  table_names?: string[];
  scope_type?: string;
  privileges?: string[];
  query_limit_num?: number;
  reason: string;
  status: string;
  current_stage_name?: string;
  mine_status?: string;
  is_final_approval?: boolean;
  expires_at?: string;
  created_at: string;
  create_meta?: {
    charset?: string;
    collation?: string;
    dev_owner_name?: string;
    dba_name?: string;
    grant_hosts?: string;
  };
};

export type DbAccessRequestPayload = {
  instance_id: number;
  scope_type?: string;
  database_name?: string;
  table_names?: string[];
  privileges?: string[];
  query_limit_num?: number;
  can_connect?: boolean;
  can_query?: boolean;
  can_dml?: boolean;
  can_ddl?: boolean;
  can_export?: boolean;
  can_import?: boolean;
  reason: string;
  expires_at?: string;
  charset?: string;
  collation?: string;
  dev_owner_user_id?: number;
  dba_user_id?: number;
  grant_hosts?: string;
};

export async function listDbTickets(projectId: number, params?: Record<string, string | number>) {
  return getData<{ list: DbTicket[]; total: number }>(http.get(`${base(projectId)}/tickets`, { params }));
}

export async function getDbTicket(projectId: number, ticketId: number) {
  return getData<DbTicket>(http.get(`${base(projectId)}/tickets/${ticketId}`));
}

export type DbRollbackItem = {
  original_sql: string;
  rollback_sql: string;
  sequence?: string;
  backup_db?: string;
};

export async function getDbTicketRollback(projectId: number, ticketId: number) {
  return getData<DbRollbackItem[]>(http.get(`${base(projectId)}/tickets/${ticketId}/rollback`));
}

export type DbRollbackPreview = {
  source_ticket_id: number;
  item_count: number;
  sql: string;
  database_name?: string;
  instance_id: number;
};

export async function previewDbRollbackTicket(projectId: number, ticketId: number) {
  return getData<DbRollbackPreview>(http.get(`${base(projectId)}/tickets/${ticketId}/rollback/preview`));
}

export async function submitDbRollbackTicket(projectId: number, ticketId: number, body?: { comment?: string }) {
  return getData<{ status: string; ticket_id?: number; risk_level?: string; message?: string }>(
    http.post(`${base(projectId)}/tickets/${ticketId}/rollback/submit`, body ?? {}),
  );
}

export type DbOscJob = {
  order_id: number;
  sql: string;
  sqlsha1: string;
  stage: string;
  stage_status: string;
};

export type DbOscPercentResult = {
  rows?: DbSqlCheckRow[];
  error?: string;
};

export async function listDbTicketOscJobs(projectId: number, ticketId: number) {
  return getData<DbOscJob[]>(http.get(`${base(projectId)}/tickets/${ticketId}/osc`));
}

export async function getDbTicketOscPercent(projectId: number, ticketId: number, sqlsha1: string) {
  return getData<DbOscPercentResult>(
    http.get(`${base(projectId)}/tickets/${ticketId}/osc/${encodeURIComponent(sqlsha1)}`),
  );
}

export async function controlDbTicketOsc(
  projectId: number,
  ticketId: number,
  sqlsha1: string,
  command: "kill" | "pause" | "resume",
) {
  return getData<DbOscPercentResult>(
    http.post(`${base(projectId)}/tickets/${ticketId}/osc/${encodeURIComponent(sqlsha1)}/control`, { command }),
  );
}

export async function approveDbTicket(projectId: number, ticketId: number, comment?: string) {
  return getData<{ ok?: boolean }>(http.post(`${base(projectId)}/tickets/${ticketId}/approve`, { comment }));
}

export async function rejectDbTicket(projectId: number, ticketId: number, comment?: string) {
  return getData<{ ok?: boolean }>(http.post(`${base(projectId)}/tickets/${ticketId}/reject`, { comment }));
}

export async function executeDbTicket(projectId: number, ticketId: number) {
  return getData<{ ok?: boolean }>(http.post(`${base(projectId)}/tickets/${ticketId}/execute`));
}

export type DbTicket = {
  id: number;
  project_id?: number;
  instance_id: number;
  instance_name?: string;
  ticket_type: string;
  submitter_name: string;
  database_name?: string;
  risk_level: string;
  status: string;
  current_stage_name?: string;
  mine_status?: string;
  submitter_user_id?: number;
  sql_excerpt?: string;
  sql_text?: string;
  review_json?: string;
  execute_json?: string;
  syntax_type?: number;
  is_backup?: boolean;
  reason?: string;
  created_at: string;
  updated_at?: string;
  sql_file_ref?: string;
  audit_mode?: "system" | "manual";
};

export async function listDbExecutions(projectId: number, params?: Record<string, string | number>) {
  return getData<{ list: DbSqlExecution[]; total: number }>(http.get(`${base(projectId)}/executions`, { params }));
}

export type DbSqlExecution = {
  id: number;
  instance_id: number;
  executor_name?: string;
  database_name?: string;
  sql_excerpt?: string;
  rows_affected?: number;
  duration_ms?: number;
  risk_level?: string;
  error_message?: string;
  created_at?: string;
};

export async function listDbAppUserRequests(projectId: number, params?: Record<string, string | number>) {
  return getData<{ list: DbAppUserRequest[]; total: number }>(
    http.get(`${base(projectId)}/app-user-requests`, { params }),
  );
}

export async function createDbAppUserRequest(projectId: number, payload: DbAppUserRequestPayload) {
  return getData<DbAppUserRequest>(http.post(`${base(projectId)}/app-user-requests`, payload));
}

export async function approveDbAppUserRequest(projectId: number, requestId: number, comment?: string) {
  return getData<{ ok?: boolean }>(http.post(`${base(projectId)}/app-user-requests/${requestId}/approve`, { comment }));
}

export async function rejectDbAppUserRequest(projectId: number, requestId: number, comment?: string) {
  return getData<{ ok?: boolean }>(http.post(`${base(projectId)}/app-user-requests/${requestId}/reject`, { comment }));
}

export type DbAppUserRequest = {
  id: number;
  instance_id: number;
  instance_name?: string;
  requester_name: string;
  apply_type: string;
  mysql_user: string;
  mysql_host?: string;
  database_name?: string;
  priv_level?: string;
  privileges?: string[];
  grant_hosts?: string;
  reason: string;
  status: string;
  execute_error?: string;
  current_stage_name?: string;
  mine_status?: string;
  created_at: string;
};

export type DbAppUserRequestPayload = {
  instance_id: number;
  apply_type: string;
  mysql_user: string;
  mysql_host?: string;
  database_name?: string;
  priv_level?: string;
  privileges: string[];
  grant_hosts?: string;
  reason: string;
};

export type DbInstanceMySQLUser = {
  id?: number;
  username: string;
  host: string;
  grant_lines?: string[];
  has_password?: boolean;
  from_platform?: boolean;
  remark?: string;
};

export async function listInstanceMySQLUsers(projectId: number, instanceId: number) {
  return getData<DbInstanceMySQLUser[]>(http.get(`${base(projectId)}/instances/${instanceId}/mysql-users`));
}

export async function getInstanceMySQLUserPrivileges(
  projectId: number,
  instanceId: number,
  params: { mysql_user: string; mysql_host?: string; priv_level: string; database?: string },
) {
  return getData<{ privileges: string[] }>(
    http.get(`${base(projectId)}/instances/${instanceId}/mysql-user-privileges`, { params }),
  );
}

export async function getInstanceAccountPassword(projectId: number, instanceId: number, accountId: number) {
  return getData<{ password: string }>(
    http.get(`${base(projectId)}/instances/${instanceId}/accounts/${accountId}/password`),
  );
}

export async function listDbAuditLogs(projectId: number, params?: Record<string, string | number>) {
  return getData<{ list: DbAuditLogItem[]; total: number }>(http.get(`${base(projectId)}/audit-logs`, { params }));
}

export type DbAuditLogItem = {
  id: number;
  project_id: number;
  instance_id?: number;
  instance_name?: string;
  instance_label?: string;
  actor_user_id: number;
  actor_name: string;
  action: string;
  detail_json?: Record<string, unknown>;
  ip?: string;
  created_at: string;
};

export type DbColumnMaskRule = {
  id: number;
  instance_id: number;
  schema_name: string;
  table_name: string;
  column_name: string;
  mask_type: string;
  pattern?: string;
  created_at?: string;
};

export async function listDbColumnMaskRules(projectId: number, instanceId: number) {
  return getData<DbColumnMaskRule[]>(http.get(`${base(projectId)}/instances/${instanceId}/column-mask-rules`));
}

export async function upsertDbColumnMaskRule(
  projectId: number,
  instanceId: number,
  payload: {
    schema_name?: string;
    table_name: string;
    column_name: string;
    mask_type: string;
    pattern?: string;
  },
) {
  return getData<DbColumnMaskRule>(http.post(`${base(projectId)}/instances/${instanceId}/column-mask-rules`, payload));
}

export async function deleteDbColumnMaskRule(projectId: number, instanceId: number, ruleId: number) {
  return getData<{ ok?: boolean }>(
    http.delete(`${base(projectId)}/instances/${instanceId}/column-mask-rules/${ruleId}`),
  );
}
