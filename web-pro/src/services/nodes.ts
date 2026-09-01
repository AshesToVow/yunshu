// @ts-nocheck
import { getData, http } from "./http";

export interface NodeItem {
  name: string;
  status: string;
  /** 为 true 时表示已 cordon，新 Pod 不会调度到该节点 */
  unschedulable?: boolean;
  roles?: string[];
  kernel: string;
  kubelet: string;
  os_image: string;
  container_runtime: string;
  architecture: string;
  internal_ip?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  taints?: Array<{
    key: string;
    value?: string;
    effect?: string;
    time_added?: string;
  }>;
  creation_time: string;
  age?: string;
  pod_count?: number;
  pod_capacity?: number;
  pod_usage?: string;
  pod_usage_percent?: number;
  cpu_usage?: string;
  cpu_requests?: string;
  cpu_limits?: string;
  mem_usage?: string;
  mem_requests?: string;
  mem_limits?: string;
  cpu_usage_percent?: number;
  mem_usage_percent?: number;
}

export interface NodeAddressItem {
  type: string;
  address: string;
}

export interface NodeDetail {
  item: NodeItem;
  addresses: NodeAddressItem[];
  conditions: Array<{
    type: string;
    status: string;
    reason?: string;
    message?: string;
    last_heartbeat_time?: string;
    last_transition_time?: string;
  }>;
  taints: Array<{
    key: string;
    value?: string;
    effect?: string;
    time_added?: string;
  }>;
  capacity: Record<string, string>;
  allocatable: Record<string, string>;
  yaml: string;
}

export function listNodes(clusterId: number, keyword?: string) {
  return getData<NodeItem[]>(http.get("/nodes", { params: { cluster_id: clusterId, keyword } }));
}

export function getNodeDetail(clusterId: number, name: string) {
  return getData<NodeDetail>(http.get("/nodes/detail", { params: { cluster_id: clusterId, name } }));
}

export type NodeTaintInput = { key: string; value?: string; effect?: string };

export function setNodeSchedulability(clusterId: number, name: string, unschedulable: boolean) {
  return getData<{ ok: boolean }>(http.post("/nodes/schedulability", { cluster_id: clusterId, name, unschedulable }));
}

export type NodeDrainPodItem = {
  namespace: string;
  name: string;
  phase: string;
  owner_kind?: string;
  action: string;
  reason?: string;
};

export type NodeDrainResult = {
  node_name: string;
  cordoned: boolean;
  dry_run: boolean;
  evicted: number;
  skipped: number;
  failed: number;
  pending: number;
  pods: NodeDrainPodItem[];
  message: string;
  completed_at?: string;
};

export type NodeDrainStatus = {
  node_name: string;
  unschedulable: boolean;
  remaining: number;
  daemonset_pods: number;
  pods: NodeDrainPodItem[];
  drained: boolean;
  message: string;
};

export function drainNode(
  clusterId: number,
  name: string,
  opts?: {
    dry_run?: boolean;
    force?: boolean;
    confirm?: boolean;
    ignore_daemon_sets?: boolean;
    delete_emptydir_data?: boolean;
    grace_period_seconds?: number;
  },
) {
  return getData<NodeDrainResult>(
    http.post(
      "/nodes/drain",
      {
        cluster_id: clusterId,
        name,
        dry_run: opts?.dry_run ?? false,
        force: opts?.force ?? false,
        confirm: opts?.confirm ?? !opts?.dry_run,
        ignore_daemon_sets: opts?.ignore_daemon_sets ?? true,
        delete_emptydir_data: opts?.delete_emptydir_data ?? true,
        grace_period_seconds: opts?.grace_period_seconds,
      },
      { timeout: 120000 },
    ),
  );
}

export function getNodeDrainStatus(clusterId: number, name: string) {
  return getData<NodeDrainStatus>(http.get("/nodes/drain-status", { params: { cluster_id: clusterId, name } }));
}

export function replaceNodeTaints(clusterId: number, name: string, taints: NodeTaintInput[]) {
  return getData<{ ok: boolean }>(http.put("/nodes/taints", { cluster_id: clusterId, name, taints }));
}
