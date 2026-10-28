import { describe, expect, it } from "vitest";
import { findMenuByPath, isMenuEnabled, resolveMenuAccessCandidates } from "./menu-path";
import type { MenuItem } from "../services/menus";

function menu(partial: Partial<MenuItem> & Pick<MenuItem, "id" | "path">): MenuItem {
  return {
    name: partial.path,
    icon: "",
    sort: 0,
    hidden: false,
    component: "",
    redirect: "",
    status: 1,
    created_at: "",
    updated_at: "",
    ...partial,
  } as MenuItem;
}

const tree: MenuItem[] = [
  menu({
    id: 1,
    path: "/k8s",
    children: [
      menu({ id: 11, path: "/k8s/clusters", component: "cluster-page" }),
      menu({ id: 12, path: "/k8s/offline", component: "offline-page", status: 0 }),
    ],
  }),
  menu({ id: 2, path: "/legacy", component: "legacy-page", status: 0 }),
  menu({ id: 3, path: "/dict-entries", component: "dict-entries-page" }),
];

describe("isMenuEnabled", () => {
  it("仅显式 status=0 视为停用", () => {
    expect(isMenuEnabled(menu({ id: 1, path: "/a", status: 1 }))).toBe(true);
    expect(isMenuEnabled(menu({ id: 2, path: "/b", status: 0 }))).toBe(false);
    expect(isMenuEnabled({ ...menu({ id: 3, path: "/c" }), status: undefined as unknown as number })).toBe(true);
  });
});

describe("findMenuByPath", () => {
  it("命中启用菜单", () => {
    expect(findMenuByPath(tree, "/dict-entries")?.id).toBe(3);
  });

  it("停用菜单即使 URL 直达也不命中（前端闸门兜底）", () => {
    expect(findMenuByPath(tree, "/legacy")).toBeUndefined();
    expect(findMenuByPath(tree, "/k8s/offline")).toBeUndefined();
  });

  it("子路由按最长前缀匹配启用叶子", () => {
    expect(findMenuByPath(tree, "/k8s/clusters/detail")?.id).toBe(11);
  });
});

describe("resolveMenuAccessCandidates", () => {
  it("辅助页继承父菜单入口，并附带工单深链备选", () => {
    expect(resolveMenuAccessCandidates("/alert-events")[0]).toBe("/alert-monitor-platform");
    expect(resolveMenuAccessCandidates("/dbmgmt/apply/query")).toContain("/workflow/inbox");
  });
});
