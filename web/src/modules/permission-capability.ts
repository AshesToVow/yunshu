/**
 * 按钮级权限的纯函数部分：把后端 /auth/permissions 的能力集合归一化为可 O(1) 查询的 Set。
 * 页面通过 usePermission(action, resource) 判断是否展示/启用操作按钮。
 */

export type PermissionCapability = {
  resource: string;
  action: string;
};

export type PermissionCapabilities = {
  superAdmin: boolean;
  /** 后端能力集合是否成功加载；未加载时前端按「放行」处理，真正拦截仍由后端 Authorize 负责。 */
  ready: boolean;
  keys: Set<string>;
};

export const EMPTY_CAPABILITIES: PermissionCapabilities = {
  superAdmin: false,
  ready: false,
  keys: new Set<string>(),
};

export function capabilityKey(action: string, resource: string) {
  return `${action.trim().toUpperCase()} ${resource.trim()}`;
}

export function buildCapabilities(input: {
  superAdmin: boolean;
  permissions?: PermissionCapability[] | null;
}): PermissionCapabilities {
  const keys = new Set<string>();
  for (const item of input.permissions ?? []) {
    if (!item?.resource || !item?.action) continue;
    keys.add(capabilityKey(item.action, item.resource));
  }
  return { superAdmin: Boolean(input.superAdmin), ready: true, keys };
}

/** 超管与「能力未加载」一律放行；仅在后端明确返回不包含该能力时隐藏按钮。 */
export function hasCapability(
  caps: PermissionCapabilities,
  action: string,
  resource: string,
): boolean {
  if (caps.superAdmin) return true;
  if (!caps.ready) return true;
  return caps.keys.has(capabilityKey(action, resource));
}
