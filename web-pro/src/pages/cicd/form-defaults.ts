// @ts-nocheck
// CI 配置表单默认值。
// 由 web/src/pages/cicd-services-page.tsx 原样搬迁（RF-09），默认值逐字保留（node24 / jdk8 等与后端工具名对应）。
import type { CicdServiceItem } from "../../services/cicd";

export function defaultCiFormValues(svc: CicdServiceItem) {
  const isFront = svc.service_type === "frontend";
  return {
    ref_type: "branch",
    ref_name: "main",
    language_type: isFront ? "frontend" : "custom",
    build_type: isFront ? "npm" : "mvn",
    build_shell: isFront ? "run build" : "clean package -DskipTests",
    build_path: isFront ? "dist" : "target",
    npm_install_mode: "install",
    node_version: "node24",
    java_tool_name: "jdk8",
    project_name: svc.identifier,
    description: svc.name,
  };
}
