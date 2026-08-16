import { Card, Select, Space, Tabs, Typography } from "antd";
import { Suspense, lazy } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { PageTelemetryHeader } from "../../components/page-telemetry-header";
import { useAlertMonitor } from "./context";
import { ALERT_MONITOR_TABS, normalizeAlertMonitorTab, tabPathForKey, type AlertMonitorTabKey } from "./tab-config";

const tabLazy: Record<AlertMonitorTabKey, React.LazyExoticComponent<() => JSX.Element>> = {
  datasources: lazy(async () => ({ default: (await import("./tabs/datasources-tab")).DatasourcesTab })),
  objects: lazy(async () => ({ default: (await import("./tabs/objects-tab")).ObjectsTab })),
  quality: lazy(async () => ({ default: (await import("./tabs/quality-tab")).QualityTab })),
  policies: lazy(async () => ({ default: (await import("./tabs/policies-tab")).PoliciesTab })),
  history: lazy(async () => ({ default: (await import("./tabs/history-tab")).HistoryTab })),
  inhibition: lazy(async () => ({ default: (await import("./tabs/inhibition-tab")).InhibitionTab })),
  silences: lazy(async () => ({ default: (await import("./tabs/silences-tab")).SilencesTab })),
  rules: lazy(async () => ({ default: (await import("./tabs/rules-tab")).RulesTab })),
  "cloud-expiry": lazy(async () => ({ default: (await import("./tabs/cloud-expiry-tab")).CloudExpiryTab })),
  promql: lazy(async () => ({ default: (await import("./tabs/promql-tab")).PromqlTab })),
};

export function AlertMonitorLayout() {
  const navigate = useNavigate();
  const { tab: tabParam } = useParams();
  const [searchParams] = useSearchParams();
  const ctx = useAlertMonitor();
  const tab = normalizeAlertMonitorTab(tabParam);

  function goTab(key: AlertMonitorTabKey) {
    const qs = searchParams.toString();
    const path = tabPathForKey(key);
    navigate(qs ? `${path}?${qs}` : path, { replace: true });
  }

  const ActiveTab = tabLazy[tab];

  return (
    <div className="page-stack">
      <PageTelemetryHeader
        label="[ ALERT / ENGINE ]"
        title="告警平台"
        subtitle="数据源评测 · 规则中心 · 事件与降噪 · 通知（夜莺式引擎，无 Alertmanager）"
        meta={[
          ctx.projectContextId ? `项目 · ${ctx.activeProjectName}` : "项目 · 全部",
          `页签 · ${ALERT_MONITOR_TABS.find((x) => x.key === tab)?.label || tab}`,
          ctx.loading ? "同步中" : "已同步",
        ]}
      />
      <Card className="table-card" loading={ctx.loading}>
        <Space className="ops-filter-bar" style={{ marginBottom: 12 }} wrap>
          <Typography.Text type="secondary">全局项目上下文</Typography.Text>
          <Select
            style={{ minWidth: 280 }}
            allowClear
            value={ctx.projectContextId}
            onChange={(v) => ctx.setProjectContext(v)}
            options={ctx.projectOptions}
            placeholder="全部项目（可选）"
          />
        </Space>
        <Tabs
          activeKey={tab}
          onChange={(k) => goTab(k as AlertMonitorTabKey)}
          items={ALERT_MONITOR_TABS.map((t) => ({ key: t.key, label: t.label }))}
        />
        <Suspense fallback={<Card loading style={{ marginTop: 16 }} />}>
          <ActiveTab />
        </Suspense>
      </Card>
    </div>
  );
}
