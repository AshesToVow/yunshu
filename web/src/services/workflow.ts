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
