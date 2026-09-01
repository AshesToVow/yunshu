// @ts-nocheck
import { getData, http } from "./http";
import type { ApiResponse } from "../types/api";

export type PendingTicketItem = {
  workflow_ticket_id: number;
  step_id: number;
  domain: string;
  ticket_type: string;
  project_id: number;
  title: string;
  status: string;
  current_stage_name: string;
  submitter_user_id: number;
  submitter_name?: string;
  ref_type: string;
  ref_id: number;
  deep_link: string;
  activated_at?: string;
  created_at: string;
  mine_status: "mine_pending" | "mine_done";
  /** review=审批；execute=提交人执行发布 */
  action?: "review" | "execute" | string;
};

export type PendingListQuery = {
  domains?: string;
  project_id?: number;
  mine_scope?: "pending" | "done" | "all";
  page?: number;
  page_size?: number;
};

type PendingListResult = {
  list: PendingTicketItem[];
  total: number;
  page: number;
  page_size: number;
};

export async function listPendingWorkflowTickets(query: PendingListQuery = {}) {
  return getData<PendingListResult>(
    http.get("/workflow/tickets/pending", { params: query }) as unknown as Promise<ApiResponse<PendingListResult>>,
  );
}

export async function reviewWorkflowStep(ticketId: number, stepId: number, approve: boolean, comment?: string) {
  return getData(
    http.post(`/workflow/tickets/${ticketId}/steps/${stepId}/review`, {
      approve,
      comment: comment ?? "",
    }) as unknown as Promise<ApiResponse<unknown>>,
  );
}

export async function getWorkflowTicket(id: number) {
  return getData(http.get(`/workflow/tickets/${id}`) as unknown as Promise<ApiResponse<unknown>>);
}

export type WorkflowStageItem = {
  stage_key: string;
  stage_name: string;
  sort_order: number;
  enabled: boolean;
  assignee_rule_type?: string;
  user_group_id?: number;
  user_group_name?: string;
  duty_monitor_rule_id?: number;
};

export type WorkflowDefinition = {
  domain: string;
  project_id: number;
  ticket_type: string;
  configured: boolean;
  stages: WorkflowStageItem[];
};

export type WorkflowStageUpsert = {
  stage_key: string;
  stage_name: string;
  sort_order: number;
  enabled: boolean;
  assignee_rule_type?: string;
  user_group_id?: number;
  duty_monitor_rule_id?: number;
};

export async function getWorkflowDefinition(domain: string, projectId: number, ticketType = "default") {
  return getData<WorkflowDefinition>(
    http.get(`/workflow/definitions/${domain}/projects/${projectId}`, {
      params: { ticket_type: ticketType },
    }) as unknown as Promise<ApiResponse<WorkflowDefinition>>,
  );
}

export async function saveWorkflowDefinition(
  domain: string,
  projectId: number,
  stages: WorkflowStageUpsert[],
  ticketType = "default",
) {
  return getData<WorkflowDefinition>(
    http.put(
      `/workflow/definitions/${domain}/projects/${projectId}`,
      { stages },
      {
        params: { ticket_type: ticketType },
      },
    ) as unknown as Promise<ApiResponse<WorkflowDefinition>>,
  );
}

export type WorkflowTicketRow = {
  id: number;
  domain: string;
  ticket_type: string;
  project_id: number;
  title: string;
  status: string;
  ref_type: string;
  ref_id: number;
  created_at: string;
};

type TicketListResult = {
  list: WorkflowTicketRow[];
  total: number;
  page: number;
  page_size: number;
};

export async function listWorkflowTickets(
  query: {
    domain?: string;
    ticket_type?: string;
    project_id?: number;
    status?: string;
    page?: number;
    page_size?: number;
  } = {},
) {
  return getData<TicketListResult>(
    http.get("/workflow/tickets", { params: query }) as unknown as Promise<ApiResponse<TicketListResult>>,
  );
}
