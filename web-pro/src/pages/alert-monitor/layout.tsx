// @ts-nocheck
import { Card, Segmented, Select, Space, Tabs, Typography } from "antd";
import { Suspense, lazy, useMemo } from "react";
import { useNavigate, useParams, useSearchParams } from '@umijs/max';
import { PageContainer } from "@ant-design/pro-components";
import { useAlertMonitor } from "./context";
import {
  ALERT_MONITOR_TAB_GROUPS,
  groupForTab,
  normalizeAlertMonitorTab,
  tabPathForKey,
  tabsInGroup,
  type AlertMonitorTabGroup,
  type AlertMonitorTabKey,
} from "./tab-config";

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
  const group = groupForTab(tab);
  const groupTabs = useMemo(() => tabsInGroup(group), [group]);

  function goTab(key: AlertMonitorTabKey) {
    const qs = new URLSearchParams(searchParams);
    if (key === "policies") {
      qs.delete("project_id");
    }
    const tail = qs.toString();
    const path = tabPathForKey(key);
    navigate(tail ? `${path}?${tail}` : path, { replace: true });
  }

  function goGroup(next: AlertMonitorTabGroup) {
    if (next === group) return;
    const first = tabsInGroup(next)[0];
    if (first) goTab(first.key);
  }

  const ActiveTab = tabLazy[tab];
  const groupLabel = ALERT_MONITOR_TAB_GROUPS.find((g) => g.key === group)?.label || group;

  return (
    <PageContainer
      header={{
        title: "告警平台",
        subTitle: "数据源评测 · 规则中心 · 事件与降噪 · 通知（夜莺式引擎，无 Alertmanager）",
      }}
    >
      <Card className="table-card" loading={ctx.loading}>
        {tab !== "policies" ? (
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
        ) : null}
        <Space direction="vertical" size="small" style={{ width: "100%", marginBottom: 8 }}>
          <Segmented
            value={group}
            onChange={(v) => goGroup(v as AlertMonitorTabGroup)}
            options={ALERT_MONITOR_TAB_GROUPS.map((g) => ({ label: g.label, value: g.key }))}
          />
          <Tabs
            activeKey={tab}
            onChange={(k) => goTab(k as AlertMonitorTabKey)}
            items={groupTabs.map((t) => ({ key: t.key, label: t.label }))}
          />
        </Space>
        <Suspense fallback={<Card loading style={{ marginTop: 16 }} />}>
          <ActiveTab />
        </Suspense>
      </Card>
    </PageContainer>
  );
}
