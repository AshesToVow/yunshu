import { Result, Spin } from "antd";
import { Link, useLocation } from "react-router-dom";
import type { ReactNode } from "react";
import { useAuth } from "../contexts/auth-context";
import { useMenuTree } from "../hooks/use-menu-tree";
import {
  findMenuByPath,
  normalizeMenuPath,
  resolveMenuAccessCandidates,
} from "../utils/menu-path";

/** 登录即可访问、不依赖菜单入口权限的路径。 */
const ALWAYS_ALLOW = new Set(["/", "/dashboard", "/personal-settings", "/plugins"]);

type MenuAccessGateProps = {
  children: ReactNode;
};

/**
 * 按侧栏菜单树拦截无入口权限的页面。
 * 后端 API 仍以 Casbin / K8s 档位为准；此处避免「藏菜单但仍可 URL 直达」。
 * 无菜单辅助页 / 旧重定向 / 工单深链备选见 resolveMenuAccessCandidates。
 */
export function MenuAccessGate({ children }: MenuAccessGateProps) {
  const location = useLocation();
  const { user } = useAuth();
  const { menus, loading, error } = useMenuTree();
  const path = normalizeMenuPath(location.pathname);
  const accessCandidates = resolveMenuAccessCandidates(path);

  if (ALWAYS_ALLOW.has(path)) {
    return <>{children}</>;
  }

  const isSuper = Boolean(user?.roles?.some((r) => r.code === "super-admin"));
  if (isSuper) {
    return <>{children}</>;
  }

  if (loading && !menus.length) {
    return (
      <div className="page-loading">
        <Spin size="large" tip="校验菜单权限..." />
      </div>
    );
  }

  if (error) {
    return (
      <Result
        status="error"
        title="菜单权限校验失败"
        subTitle={error instanceof Error ? error.message : "无法加载菜单树，请刷新后重试"}
      />
    );
  }

  const menuItem = menus?.length
    ? accessCandidates.map((c) => findMenuByPath(menus, c)).find(Boolean)
    : undefined;
  if (!menuItem) {
    return (
      <Result
        status="403"
        title="无访问权限"
        subTitle={`当前账号不具备菜单 ${accessCandidates[0] ?? path} 的入口权限。请联系管理员在「授权管理」中分配对应 API。`}
        extra={<Link to="/">返回总览</Link>}
      />
    );
  }

  return <>{children}</>;
}
