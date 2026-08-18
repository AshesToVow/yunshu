import { AlertQualityPage } from "../../alert-quality-page";
import { useAlertMonitor } from "../context";

/** 告警平台内嵌质量看板：跟随顶栏项目上下文。 */
export function QualityTab() {
  const ctx = useAlertMonitor();
  return <AlertQualityPage embedded projectContextId={ctx.projectContextId} />;
}
