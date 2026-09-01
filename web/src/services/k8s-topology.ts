import { getData, http } from "./http";

export interface TopologyNode {
  id: string;
  label: string;
  kind: string;
  state?: string;
  state_level?: "normal" | "progressing" | "abnormal" | string;
}

export interface TopologyEdge {
  from: string;
  to: string;
  kind?: string;
}

export interface TopologyGraph {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
}

export function getWorkloadTopology(params: { cluster_id: number; namespace: string; kind: string; name: string }) {
  return getData<TopologyGraph>(http.get("/k8s/topology", { params }));
}
