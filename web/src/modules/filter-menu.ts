import type { MenuItem } from "../services/menus";
import type { AntdMenuItem } from "../utils/admin-menu";
import { isPathAllowedByPlugins } from "./plugin-path";

/** 按后端启用的业务插件过滤菜单树；父目录关闭时提升已放行的子菜单 */
export function filterMenuTreeByPlugins(
  menus: MenuItem[],
  isPluginEnabled: (name: string) => boolean,
): MenuItem[] {
  const walk = (nodes: MenuItem[]): MenuItem[] => {
    const out: MenuItem[] = [];
    for (const m of nodes) {
      const children = m.children?.length ? walk(m.children) : undefined;
      const path = m.path?.trim();
      if (path && !isPathAllowedByPlugins(path, isPluginEnabled)) {
        if (children?.length) out.push(...children);
        continue;
      }
      if (m.children?.length && !(children?.length)) {
        continue;
      }
      out.push({ ...m, children });
    }
    return out;
  };
  return walk(menus);
}

/** 过滤 Ant Design 侧栏菜单（兜底菜单等） */
export function filterAntdMenuItems(
  items: AntdMenuItem[],
  isPluginEnabled: (name: string) => boolean,
): AntdMenuItem[] {
  const walk = (nodes: AntdMenuItem[]): AntdMenuItem[] => {
    const out: AntdMenuItem[] = [];
    for (const node of nodes) {
      if (!node || typeof node !== "object") continue;
      if ("children" in node && Array.isArray(node.children) && node.children.length) {
        const children = walk(node.children as AntdMenuItem[]);
        const key = String(("key" in node && node.key) || "");
        if (key && !isPathAllowedByPlugins(key, isPluginEnabled)) {
          out.push(...children);
          continue;
        }
        if (!children.length) continue;
        out.push({ ...node, children });
        continue;
      }
      const key = String(("key" in node && node.key) || "");
      if (key && !isPathAllowedByPlugins(key, isPluginEnabled)) {
        continue;
      }
      out.push(node);
    }
    return out;
  };
  return walk(items);
}
