import { getData, http } from "./http";

export interface EventItem {
  namespace: string;
  type: string;
  reason: string;
  message: string;
  count: number;
  first_time?: string;
  last_time?: string;
  creation_time?: string;
  involved_kind?: string;
  involved_name?: string;
}

export interface EventGroupItem {
  namespace: string;
  involved_kind: string;
  involved_name: string;
  reason: string;
  type: string;
  total_count: number;
  event_count: number;
  last_time: string;
  first_time?: string;
  message: string;
  events?: EventItem[];
}

export function listEvents(params: {
  cluster_id: number;
  namespace?: string;
  kind?: string;
  name?: string;
  keyword?: string;
  limit?: number;
}) {
  return getData<EventItem[]>(http.get("/events", { params }));
}

export function listEventsGrouped(params: {
  cluster_id: number;
  namespace?: string;
  kind?: string;
  name?: string;
  keyword?: string;
  limit?: number;
}) {
  return getData<EventGroupItem[]>(http.get("/events/grouped", { params }));
}
