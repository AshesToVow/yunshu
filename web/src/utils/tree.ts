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

  for (const permission of permissions) {
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

export function buildUnifiedPermissionTreeData(nodes: PermissionTreeNode[]): AppTreeData {
  const walk = (items: PermissionTreeNode[]): AppTreeData =>
    items.map((node) => {
      const isAPI = node.node_type === "api";
      const suffix = node.plugin_disabled ? "（插件未启用）" : "";
      const grantedTag = node.granted ? " ✓" : "";
      return {
        key: node.key,
        title: `${node.title}${suffix}${grantedTag}`,
        selectable: false,
        disableCheckbox: !isAPI || !!node.plugin_disabled,
        value: isAPI ? node.permission_id : undefined,
        children: node.children?.length ? walk(node.children) : undefined,
      };
    });
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
  return rawKeys
    .map((item) => Number(item))
    .filter((item) => Number.isInteger(item));
}

function getModuleName(resource: string) {
  const segments = resource.split("/").filter(Boolean);
  return segments[2] ?? segments[segments.length - 1] ?? resource;
}
