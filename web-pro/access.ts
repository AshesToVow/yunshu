// @ts-nocheck
// CI/CD 权限判定。
// 由 web/src/pages/cicd-services-page.tsx 原样搬迁（RF-09），判定顺序与默认值逐字保留。
// 语义必须与后端 internal/pkg/projectacl 保持一致，改动前先对照后端实现。
import type { CicdServiceItem } from "../../services/cicd";
import type { UserItem } from "../../types/api";

/** 后端未下发 access 时一律按「无权限」兜底，避免前端越权渲染操作按钮 */
export function cicdAccess(row: CicdServiceItem) {
  return (
    row.access ?? {
      can_view: false,
      can_build: false,
      can_release: false,
      can_manage: false,
    }
  );
}

export function isSuperAdminUser(u: UserItem | null | undefined): boolean {
  return Boolean(u?.roles?.some((r) => r.code === "super-admin"));
}

/** 与后端 projectacl.FullAccess 一致：超管或项目 owner/admin 可新建应用 */
export function canCreateCicdService(isSuper: boolean, myProjectRole: string | null | undefined): boolean {
  if (isSuper) return true;
  const r = String(myProjectRole || "").toLowerCase();
  return r === "owner" || r === "admin";
}
