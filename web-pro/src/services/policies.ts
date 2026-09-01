// @ts-nocheck
import type {
  MessageData,
  PermissionMenuLinksResponse,
  PermissionTreeResponse,
  PolicyConflictItem,
  PolicyItem,
  PolicyPayload,
  PolicySimulateRequest,
  PolicySimulateResponse,
} from "../types/api";
import { getData, http } from "./http";

export function getPolicies() {
  return getData<PolicyItem[]>(http.get("/policies"));
}

export function grantPolicy(payload: PolicyPayload) {
  return getData<MessageData>(http.post("/policies", payload));
}

export function revokePolicy(payload: PolicyPayload) {
  return getData<MessageData>(http.delete("/policies", { data: payload }));
}

export function getPolicyMenuLinks() {
  return getData<PermissionMenuLinksResponse>(http.get("/policies/menu-links"));
}

export function simulatePolicy(payload: PolicySimulateRequest) {
  return getData<PolicySimulateResponse>(http.post("/policies/simulate", payload));
}

export function getPolicyConflicts(roleId: number) {
  return getData<{ role_id: number; role_code: string; items: PolicyConflictItem[] }>(
    http.get("/policies/conflicts", { params: { role_id: roleId } }),
  );
}

export function fixMenuEntryAPIs(roleId: number) {
  return getData<{ granted: number; created: number; skipped: number; total: number }>(
    http.post("/policies/conflicts/fix-menu-entry", {}, { params: { role_id: roleId } }),
  );
}

export function fixDisabledPluginPolicies(roleId: number) {
  return getData<{ revoked: number; skipped: number; total: number }>(
    http.post("/policies/conflicts/fix-disabled-plugin", {}, { params: { role_id: roleId } }),
  );
}

export function getPermissionTree(roleId: number) {
  return getData<PermissionTreeResponse>(http.get("/policies/permission-tree", { params: { role_id: roleId } }));
}
