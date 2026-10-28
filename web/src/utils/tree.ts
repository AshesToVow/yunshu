import type { TreeProps } from "antd";
import type { PermissionItem, PermissionTreeNode, RoleItem } from "../types/api";

export type AppTreeData = NonNullable<TreeProps["treeData"]>;

export type PermissionTreeBuildOptions = {
  menuLinks?: Record<string, { path: string }[]>;
  isPluginAllowed?: (resource: string) => boolean;
};

export function buildRoleTreeData(roles: RoleItem[]): AppTreeData {
  const enabledRoles = roles.filter((role) => role.status === 1);
  const disabledRoles = roles.filter((role) => role.status !== 1);
  const treeData: AppTreeData = [];

  if (enabledRoles.length > 0) {
    treeData.push({
      key: "roles-enabled",
      title: `启用角色 (${enabledRoles.length})`,
      selectable: false,
      disableCheckbox: true,
      children: enabledRoles.map((role) => ({
        key: role.id,
        value: role.id,
        title: `${role.name} (${role.code})`,
      })),
    });
  }

  if (disabledRoles.length > 0) {
    treeData.push({
      key: "roles-disabled",
      title: `停用角色 (${disabledRoles.length})`,
      selectable: false,
      disableCheckbox: true,
      children: disabledRoles.map((role) => ({
        key: role.id,
        value: role.id,
        title: `${role.name} (${role.code})`,
      })),
    });
  }

  return treeData;
}

export function buildPermissionTreeData(permissions: PermissionItem[], opts?: PermissionTreeBuildOptions): AppTreeData {
  const moduleMap = new Map<string, Map<string, PermissionItem[]>>();
  // 同一 resource+action 只保留最小 id，避免历史重复 seed 导致树节点翻倍。
  const deduped = dedupePermissionsByResourceAction(permissions);

  for (const permission of deduped) {
    const moduleName = getModuleName(permission.resource);
    if (!moduleMap.has(moduleName)) {
      moduleMap.set(moduleName, new Map<string, PermissionItem[]>());
    }
    const resourceMap = moduleMap.get(moduleName)!;
    if (!resourceMap.has(permission.resource)) {
      resourceMap.set(permission.resource, []);
    }
    resourceMap.get(permission.resource)!.push(permission);
  }

  return Array.from(moduleMap.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([moduleName, resourceMap]) => ({
      key: `module:${moduleName}`,
      title: moduleName,
      selectable: false,
      disableCheckbox: true,
      children: Array.from(resourceMap.entries())
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([resource, items]) => ({
          key: `resource:${resource}`,
          title: resource,
          selectable: false,
          disableCheckbox: true,
          children: items
            .slice()
            .sort((left, right) => {
              const actionCompare = left.action.localeCompare(right.action);
              if (actionCompare !== 0) {
                return actionCompare;
              }
              return left.name.localeCompare(right.name);
            })
            .map((permission) => {
              const key = `${permission.resource}::${permission.action.toUpperCase()}`;
              const menus = opts?.menuLinks?.[key] ?? [];
              const menuHint = menus.length > 0 ? ` → ${menus.map((m) => m.path).join(", ")}` : "";
              const pluginOff = opts?.isPluginAllowed ? !opts.isPluginAllowed(permission.resource) : false;
              return {
                key: permission.id,
                value: permission.id,
                disableCheckbox: pluginOff,
                title: `${permission.action} · ${permission.name}${menuHint}${pluginOff ? "（插件未启用）" : ""}`,
              };
            }),
        })),
    }));
}

export type TreeKey = string | number;

// 与后端 PermissionTree 的 "perm:N"/"menu:N" 字符串 key 不同，
// API 节点直接使用权限 ID 作为 key，才能与 checkedKeys（数字 ID）对上。
export function buildUnifiedPermissionTreeData(nodes: PermissionTreeNode[]): AppTreeData {
  const seenPermissions = new Set<number>();
  const walk = (items: PermissionTreeNode[]): AppTreeData => {
    const out: AppTreeData = [];
    for (const node of items) {
      const suffix = node.plugin_disabled ? "（插件未启用）" : "";
      if (node.node_type === "api") {
        // 同一入口 API 可能绑定多个菜单，重复出现会导致 key 冲突与勾选联动错乱
        if (!node.permission_id || seenPermissions.has(node.permission_id)) {
          continue;
        }
        seenPermissions.add(node.permission_id);
        out.push({
          key: node.permission_id,
          title: `${node.title}${suffix}`,
          selectable: false,
          disableCheckbox: !!node.plugin_disabled,
        });
        continue;
      }
      const children = node.children?.length ? walk(node.children) : [];
      if (children.length === 0) {
        continue;
      }
      out.push({
        key: node.key,
        title: `${node.title}${suffix}`,
        selectable: false,
        disableCheckbox: true,
        children,
      });
    }
    return out;
  };
  return walk(nodes);
}

export function collectGrantedPermissionIds(nodes: PermissionTreeNode[]): number[] {
  const ids: number[] = [];
  const walk = (items: PermissionTreeNode[]) => {
    for (const node of items) {
      if (node.node_type === "api" && node.granted && node.permission_id) {
        ids.push(node.permission_id);
      }
      if (node.children?.length) walk(node.children);
    }
  };
  walk(nodes);
  return ids;
}

export function normalizeCheckedKeys(checkedKeys: Parameters<NonNullable<TreeProps["onCheck"]>>[0]): number[] {
  const rawKeys = Array.isArray(checkedKeys) ? checkedKeys : checkedKeys.checked;
  return rawKeys.map((item) => Number(item)).filter((item) => Number.isInteger(item));
}

export function collectTreeDataKeys(data: AppTreeData): TreeKey[] {
  const keys: TreeKey[] = [];
  const walk = (nodes: AppTreeData) => {
    for (const node of nodes) {
      keys.push(node.key as TreeKey);
      if (node.children?.length) {
        walk(node.children);
      }
    }
  };
  walk(data);
  return keys;
}

// antd Tree 的 onCheck 只返回当前树里存在的 key；两棵树共享同一份勾选状态时，
// 不在当前树里的已勾选 ID 必须显式保留，否则会被同步误回收。
export function mergeTreeCheckedKeys(current: number[], treeChecked: number[], treeIds: Set<number>): number[] {
  const outOfTree = current.filter((id) => !treeIds.has(id));
  return [...new Set([...outOfTree, ...treeChecked])];
}

function getModuleName(resource: string) {
  const segments = resource.split("/").filter(Boolean);
  return segments[2] ?? segments[segments.length - 1] ?? resource;
}

function dedupePermissionsByResourceAction(permissions: PermissionItem[]): PermissionItem[] {
  const best = new Map<string, PermissionItem>();
  for (const permission of permissions) {
    const key = `${permission.resource}::${String(permission.action ?? "").toUpperCase()}`;
    const prev = best.get(key);
    if (!prev || permission.id < prev.id) {
      best.set(key, permission);
    }
  }
  return Array.from(best.values());
}
