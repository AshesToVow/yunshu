import type { MenuItem } from "../services/menus";

export function normalizeMenuPath(path: string): string {
  if (!path?.trim()) return "/";
  const p = path.startsWith("/") ? path : `/${path}`;
  return p.replace(/\/$/, "") || "/";
}

/**
 * 辅助页 / 已下线旧 path：无独立侧栏菜单（或菜单已删），入口权限继承目标菜单。
 * MenuAccessGate 必须先解析到此，否则 Navigate/详情页会被 Gate 先拦。
 */
export const AUX_MENU_PARENT: Record<string, string> = {
  // 无菜单辅助页
  "/server-console": "/project-servers",

  // 已并入其它页面的旧入口
  "/alert-config-center": "/alert-monitor-platform",
  "/alert-events": "/alert-monitor-platform",
  "/log-kafka": "/log-retention",
  "/runtime-config": "/dict-entries",
  "/cluster": "/clusters",
  "/project-log-sources": "/project-services",

  // 数据库 / CI 旧待办与重定向 path
  "/dbmgmt/workflow/pending": "/workflow/inbox",
  "/dbmgmt/todo": "/workflow/inbox",
  "/dbmgmt/approval-flow": "/workflow/definitions",
  "/dbmgmt/console": "/dbmgmt/sql/query",
  "/dbmgmt/access-requests": "/dbmgmt/apply/query",
  "/dbmgmt/access-request": "/dbmgmt/apply/query",
  "/dbmgmt/access-requests/all": "/dbmgmt/apply/query",
  "/dbmgmt/tickets": "/dbmgmt/workflow/history",
  "/cicd/todo": "/workflow/inbox",
  "/cicd/approval-flow": "/workflow/definitions",
};

/**
 * 动态子路径（如 /dbmgmt/workflow/tickets/:id）按最长前缀继承父菜单入口。
 * findMenuByPath 的前缀匹配只认「带 component 的菜单 path」，无法覆盖兄弟子路径。
 */
export const AUX_MENU_PREFIX_PARENT: Record<string, string> = {
  "/dbmgmt/workflow/tickets": "/dbmgmt/workflow/history",
};

/**
 * 工单中心「我的待办」深链到各域详情页时，审批人往往只有 inbox 入口。
 * 除主映射外，额外允许这些菜单作为入口（任一命中即可）。
 */
export const AUX_MENU_ACCESS_ALTERNATES: Record<string, string[]> = {
  "/dbmgmt/workflow/tickets": ["/workflow/inbox"],
  "/dbmgmt/apply/query": ["/workflow/inbox"],
  "/dbmgmt/apply/app-user": ["/workflow/inbox"],
  "/cicd/release-records": ["/workflow/inbox"],
  "/alert-monitor-platform": ["/workflow/inbox"],
  "/alert-events": ["/workflow/inbox"],
};

/** 旧拼写兼容（下划线 → 连字符）。 */
const PATH_SPELLING_ALIASES: Record<string, string> = {
  "/server_console": "/server-console",
};

/**
 * 将当前 URL 解析为入口权限校验用的菜单 path。
 */
export function resolveMenuAccessPath(pathname: string): string {
  let p = normalizeMenuPath(pathname);
  p = PATH_SPELLING_ALIASES[p] ?? p;
  if (AUX_MENU_PARENT[p]) {
    return AUX_MENU_PARENT[p];
  }
  let bestParent = "";
  let bestLen = -1;
  for (const [prefix, parent] of Object.entries(AUX_MENU_PREFIX_PARENT)) {
    if (p === prefix || p.startsWith(`${prefix}/`)) {
      if (prefix.length > bestLen) {
        bestLen = prefix.length;
        bestParent = parent;
      }
    }
  }
  if (bestParent) {
    return bestParent;
  }
  return p;
}

/**
 * 入口权限候选菜单 path：主映射 + 工单深链备选（如 inbox）。
 */
export function resolveMenuAccessCandidates(pathname: string): string[] {
  let p = normalizeMenuPath(pathname);
  p = PATH_SPELLING_ALIASES[p] ?? p;
  const primary = resolveMenuAccessPath(p);
  const out: string[] = [primary];
  const seen = new Set(out);

  const pushAlts = (key: string) => {
    for (const alt of AUX_MENU_ACCESS_ALTERNATES[key] ?? []) {
      if (!seen.has(alt)) {
        seen.add(alt);
        out.push(alt);
      }
    }
  };

  pushAlts(p);
  for (const prefix of Object.keys(AUX_MENU_ACCESS_ALTERNATES)) {
    if (p === prefix || p.startsWith(`${prefix}/`)) {
      pushAlts(prefix);
    }
  }
  return out;
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
