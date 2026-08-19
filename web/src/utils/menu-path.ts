import type { MenuItem } from "../services/menus";

export function normalizeMenuPath(path: string): string {
  if (!path?.trim()) return "/";
  const p = path.startsWith("/") ? path : `/${path}`;
  return p.replace(/\/$/, "") || "/";
}

export function flattenMenuItems(menus: MenuItem[]): MenuItem[] {
  const out: MenuItem[] = [];
  function walk(list: MenuItem[]) {
    for (const m of list) {
      out.push(m);
      if (m.children?.length) walk(m.children);
    }
  }
  walk(menus);
  return out;
}

/**
 * 按当前 URL 在已授权菜单树中查找节点。
 * 先精确匹配；再按「带 component 的叶子」做最长前缀匹配（支持 /page/:id 等子路由）。
 */
export function findMenuByPath(menus: MenuItem[], pathname: string): MenuItem | undefined {
  const p = normalizeMenuPath(pathname);
  const flat = flattenMenuItems(menus);
  const exact = flat.find((m) => normalizeMenuPath(m.path) === p);
  if (exact) return exact;

  let best: MenuItem | undefined;
  let bestLen = -1;
  for (const m of flat) {
    if (!m.component?.trim()) continue;
    const mp = normalizeMenuPath(m.path);
    if (mp === "/") continue;
    if (p === mp || p.startsWith(`${mp}/`)) {
      if (mp.length > bestLen) {
        best = m;
        bestLen = mp.length;
      }
    }
  }
  return best;
}
