/**
 * K8s 资源「表单创建」抽屉集合 —— 桶文件（re-export）。
 *
 * RF-11 已完成拆分，本文件不再包含实现，仅保留导入路径以保证调用页零改动：
 * - 抽屉外壳 → ./form-drawers/drawer-shell-form
 * - 下拉选项常量 → ./form-drawers/options
 * - Namespace / ConfigMap / Secret → ./form-drawers/core-drawers
 * - Ingress → ./form-drawers/ingress-drawer
 * - Role / ClusterRole / RoleBinding / ClusterRoleBinding → ./form-drawers/rbac-drawers
 * - ServiceAccount → ./form-drawers/service-account-drawer
 *
 * 新增抽屉请直接放到 ./form-drawers/ 下并在此补一行 re-export，不要在本文件写实现。
 */

export {
  ConfigMapFormCreateDrawer,
  NamespaceFormCreateDrawer,
  SecretFormCreateDrawer,
} from "./form-drawers/core-drawers";
export { IngressFormCreateDrawer } from "./form-drawers/ingress-drawer";
export {
  RbacClusterRoleBindingFormCreateDrawer,
  RbacClusterRoleFormCreateDrawer,
  RbacRoleBindingFormCreateDrawer,
  RbacRoleFormCreateDrawer,
} from "./form-drawers/rbac-drawers";
export { ServiceAccountFormCreateDrawer } from "./form-drawers/service-account-drawer";
