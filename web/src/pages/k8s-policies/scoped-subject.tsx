// K8s 集群档位授权的主体类型、档位能力包与展示映射（RF-12）。
// 由 web/src/pages/k8s-scoped-policies-page.tsx 原样搬迁，取值与分支顺序逐字保留。
// 注意：PRESET_CAPS 的能力 code 与后端 k8s capability 目录一一对应（listK8sCapabilities），
// 且 admin 档位的能力顺序即后端下发顺序，不要「顺手排序」。
import { Space, Tag } from "antd";
import type { RoleItem } from "../../types/api";
import type { UserGroupItem } from "../../services/user-groups";

/** 授权主体：角色模板 / 用户组 / 用户直授 */
export type SubjectKind = "role" | "group" | "user";

export const PRESET_CAPS: Record<"readonly" | "readonly_exec" | "admin", string[]> = {
  readonly: ["read"],
  readonly_exec: ["read", "exec"],
  admin: ["read", "exec", "restart", "scale", "apply", "delete", "secret_reveal", "destructive"],
};

/** 首屏 bootstrap 的主体偏好（来自 URL query，用于从「授权管理」跳转过来时定位主体） */
export type BootstrapPref = {
  kind?: SubjectKind;
  roleId?: number;
  groupId?: number;
  userId?: number;
};

/**
 * 主体在黑名单规则里的 principal_ref：
 * 角色/用户组用 code，用户用 id 字符串——与后端 k8s_namespace_deny_rules.principal_ref 约定一致。
 */
export function subjectPrincipalRef(
  kind: SubjectKind,
  role: RoleItem | null,
  group: UserGroupItem | null,
  userId?: number,
): string {
  if (kind === "role") return role?.code ?? "";
  if (kind === "group") return group?.code ?? "";
  return userId != null && userId > 0 ? String(userId) : "";
}

export function presetLabel(p: string) {
  switch (p) {
    case "readonly":
      return "只读";
    case "readonly_exec":
      return "只读+Exec";
    case "admin":
      return "集群管理";
    case "custom":
      return "自定义能力包";
    default:
      return p;
  }
}

/** 能力 code → 中文名（取不到名称时回落展示 code，便于排查后端目录缺项） */
export function renderCapabilityTags(codes: string[] | undefined, capNameByCode: Map<string, string>) {
  const list = Array.isArray(codes) ? codes : [];
  if (list.length === 0) return <span className="inline-muted">—</span>;
  return (
    <Space size={[4, 4]} wrap>
      {list.map((code) => (
        <Tag key={code}>{capNameByCode.get(code) || code}</Tag>
      ))}
    </Space>
  );
}
