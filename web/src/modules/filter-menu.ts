import type { MenuItem } from "../services/menus";
import type { AntdMenuItem } from "../utils/admin-menu";
import { isPathAllowedByPlugins } from "./plugin-path";

/** 按后端启用的业务插件过滤菜单树 */
export function filterMenuTreeByPlugins(
  menus: MenuItem[],
  isPluginEnabled: (name: string) => boolean,
): MenuItem[] {
  const walk = (nodes: MenuItem[]): MenuItem[] =>
    nodes
      .map((m) => {
        const children = m.children?.length ? walk(m.children) : undefined;
        return { ...m, children };
      })
      .filter((m) => {
        const path = m.path?.trim();
        if (path && !isPathAllowedByPlugins(path, isPluginEnabled)) {
          return false;
        }
        if (m.children?.length) {
          return m.children.length > 0;
        }
        return true;
      });

  return walk(menus);
}

/** 过滤 Ant Design 侧栏菜单（兜底菜单等） */
export function filterAntdMenuItems(
  items: AntdMenuItem[],
  isPluginEnabled: (name: string) => boolean,
): AntdMenuItem[] {
  const walk = (nodes: AntdMenuItem[]): AntdMenuItem[] =>
    nodes
      .map((node) => {
        if (!node || typeof node !== "object") return node;
        if ("children" in node && Array.isArray(node.children) && node.children.length) {
          const children = walk(node.children as AntdMenuItem[]);
          if (!children.length) return null;
          return { ...node, children };
        }
        const key = String(("key" in node && node.key) || "");
        if (key && !isPathAllowedByPlugins(key, isPluginEnabled)) {
          return null;
        }
        return node;
      })
      .filter(Boolean) as AntdMenuItem[];

  return walk(items);
}
