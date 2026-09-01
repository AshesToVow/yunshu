// @ts-nocheck
/**
 * K8s 资源表单创建抽屉共用的下拉选项常量。
 *
 * 从 k8s-resource-form-drawers.tsx 原地抽出（RF-11 第一步），仅搬迁不改语义。
 * 所有 value 均为 K8s API 的**字面量取值**，直接进入 apply 的 YAML 文档，
 * 因此禁止为了「界面好看」修改 value；只允许改 label。
 */

export type SelectOption = { label: string; value: string };

/** Secret.type 可选值；顺序与 kubectl 文档一致，Opaque 为默认项。 */
export const secretTypes: SelectOption[] = [
  { label: "Opaque", value: "Opaque" },
  { label: "kubernetes.io/dockerconfigjson", value: "kubernetes.io/dockerconfigjson" },
  { label: "kubernetes.io/tls", value: "kubernetes.io/tls" },
  { label: "kubernetes.io/basic-auth", value: "kubernetes.io/basic-auth" },
  { label: "kubernetes.io/ssh-auth", value: "kubernetes.io/ssh-auth" },
];

/**
 * Ingress path.pathType 可选值。
 * Prefix 放首位是刻意的：表单未选时会回落 "Prefix"（见 IngressFormCreateDrawer.submit）。
 */
export const pathTypes: SelectOption[] = [
  { label: "Prefix", value: "Prefix" },
  { label: "ImplementationSpecific", value: "ImplementationSpecific" },
  { label: "Exact", value: "Exact" },
];

/**
 * RBAC rules[].apiGroups 可选值。
 * 核心组的 value 必须是空字符串 ""（K8s 约定），label 里的 (\"\") 只是给用户的提示。
 */
export const apiGroupOptions: SelectOption[] = [
  { label: "核心组 (\"\")", value: "" },
  { label: "apps", value: "apps" },
  { label: "batch", value: "batch" },
  { label: "networking.k8s.io", value: "networking.k8s.io" },
  { label: "rbac.authorization.k8s.io", value: "rbac.authorization.k8s.io" },
];

/** RBAC rules[].resources 常用资源；均为复数小写形式，与 K8s 资源名一致。 */
export const resourceOptions: SelectOption[] = [
  "pods",
  "services",
  "configmaps",
  "secrets",
  "deployments",
  "statefulsets",
  "daemonsets",
  "ingresses",
  "nodes",
  "namespaces",
  "events",
  "endpoints",
  "persistentvolumeclaims",
  "persistentvolumes",
].map((r) => ({ label: r, value: r }));

/** RBAC rules[].verbs 可选值；小写，不含 deletecollection（如需请同步后端权限校验）。 */
export const verbOptions: SelectOption[] = ["get", "list", "watch", "create", "update", "patch", "delete"].map((v) => ({
  label: v,
  value: v,
}));

/** RoleBinding / ClusterRoleBinding subjects[].kind 可选值。 */
export const subjectKindOptions: SelectOption[] = [
  { label: "User", value: "User" },
  { label: "ServiceAccount", value: "ServiceAccount" },
  { label: "Group", value: "Group" },
];
