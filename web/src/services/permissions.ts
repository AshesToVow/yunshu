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
    const pageSize = 200;
    let page = 1;
    let total = 0;
    const list: PermissionItem[] = [];

    while (true) {
      const data = await getData<PageData<PermissionItem>>(http.get("/permissions", { params: { page, page_size: pageSize } }));
      if (page === 1) total = data.total ?? 0;
      if (Array.isArray(data.list) && data.list.length > 0) {
        list.push(...data.list);
      }
      if (!data.list?.length || list.length >= total) break;
      page += 1;
    }

    const filtered = filterPermissionsByPlugins(list, isEnabled);
    return {
      list: filtered,
      total: filtered.length,
      page: 1,
      page_size: filtered.length || pageSize,
    } satisfies PageData<PermissionItem>;
  })();
}