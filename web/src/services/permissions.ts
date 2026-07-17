import type {
  MessageData,
  PageData,
  PermissionItem,
  PermissionPayload,
  PermissionQuery,
} from "../types/api";
import { DEFAULT_ENABLED_PLUGINS, filterPermissionsByPlugins } from "../modules/plugin-path";
import { getData, http } from "./http";

function defaultPluginEnabled(name: string) {
  return DEFAULT_ENABLED_PLUGINS.includes(name.toLowerCase() as (typeof DEFAULT_ENABLED_PLUGINS)[number]);
}

export function getPermissions(params: PermissionQuery) {
  return getData<PageData<PermissionItem>>(http.get("/permissions", { params }));
}

export function getPermission(id: number) {
  return getData<PermissionItem>(http.get(`/permissions/${id}`));
}

export function createPermission(payload: PermissionPayload, opts?: { silentErrorToast?: boolean }) {
  return getData<PermissionItem>(
    http.post("/permissions", payload, opts?.silentErrorToast ? { silentErrorToast: true } : {}),
  );
}

export function updatePermission(id: number, payload: Partial<PermissionPayload>) {
  return getData<PermissionItem>(http.put(`/permissions/${id}`, payload));
}

export function deletePermission(id: number) {
  return getData<MessageData>(http.delete(`/permissions/${id}`));
}

export type PermissionBatchK8sScopePayload = {
  enabled: boolean;
  keyword?: string;
  k8s_related?: "on" | "";
};

export type PermissionBatchK8sScopeResult = {
  affected: number;
};

export function batchSetPermissionK8sScope(payload: PermissionBatchK8sScopePayload) {
  return getData<PermissionBatchK8sScopeResult>(http.post("/permissions/k8s-scope/batch", payload));
}

export function getPermissionOptions(opts?: { isPluginEnabled?: (name: string) => boolean }) {
  const isEnabled = opts?.isPluginEnabled ?? defaultPluginEnabled;
  return (async () => {
    const all = await listAllPermissions();
    const filtered = filterPermissionsByPlugins(all, isEnabled);
    return {
      list: filtered,
      total: filtered.length,
      page: 1,
      page_size: filtered.length || 200,
    } satisfies PageData<PermissionItem>;
  })();
}

export function listAllPermissions() {
  return (async () => {
    const pageSize = 100;
    let page = 1;
    const list: PermissionItem[] = [];

    while (true) {
      const data = await getData<PageData<PermissionItem>>(http.get("/permissions", { params: { page, page_size: pageSize } }));
      if (!Array.isArray(data.list) || data.list.length === 0) {
        break;
      }
      list.push(...data.list);
      if (data.list.length < pageSize) {
        break;
      }
      page += 1;
    }
    return list;
  })();
}