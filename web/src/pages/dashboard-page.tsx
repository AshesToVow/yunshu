import {
  AlertOutlined,
  ApiOutlined,
  CalendarOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloudOutlined,
  ClusterOutlined,
  DashboardOutlined,
  DesktopOutlined,
  DisconnectOutlined,
  InfoOutlined,
  LoginOutlined,
  ProfileOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Card, Col, Divider, Progress, Row, Space, Statistic, Table, Tag, Typography } from "antd";
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getHealth } from "../services/auth";
import { getOverview, getOverviewTrends } from "../services/overview";
import type { OverviewTrendsResponse } from "../services/overview";
import { LineChart } from "../components/line-chart";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { getOperationLogs, type OperationLogItem } from "../services/operation-logs";
import { getLoginLogs, type LoginLogItem } from "../services/login-logs";
import { useAdminThemeMode } from "../hooks/use-admin-theme-mode";
import { formatDateTime } from "../utils/format";

interface DashboardMetrics {
  users: number;
  clusters: number;
  pendingRegistrations: number;
  servers: number;
  podNormal: number;
  podAbnormal: number;
  podClusterErrors: number;
  eventTotal: number;
  eventWarning: number;
  eventClusterErrors: number;
  alertFiring: number;
  alertEventsToday: number;
  logAgentsOnline: number;
  logAgentsOffline: number;
}

interface SystemHealth {
  status: string;
  version: string;
  uptime: number;
  loading: boolean;
}

const defaultMetrics: DashboardMetrics = {
  users: 0,
  clusters: 0,
  pendingRegistrations: 0,
  servers: 0,
  podNormal: 0,
  podAbnormal: 0,
  podClusterErrors: 0,
  eventTotal: 0,
  eventWarning: 0,
  eventClusterErrors: 0,
  alertFiring: 0,
  alertEventsToday: 0,
  logAgentsOnline: 0,
  logAgentsOffline: 0,
};

const assetStats = [
  { key: "users", icon: <TeamOutlined />, accent: "#22d3ee" },
  { key: "clusters", icon: <ClusterOutlined />, accent: "#38bdf8" },
  { key: "servers", icon: <DesktopOutlined />, accent: "#a78bfa" },
  { key: "pendingRegistrations", icon: <SafetyCertificateOutlined />, accent: "#fbbf24" },
] as const;

const k8sStats = [
  { key: "podNormal", icon: <CheckCircleOutlined />, accent: "#34d399" },
  { key: "podAbnormal", icon: <WarningOutlined />, accent: "#fb7185" },
  { key: "eventTotal", icon: <CloudOutlined />, accent: "#818cf8" },
  { key: "eventWarning", icon: <ThunderboltOutlined />, accent: "#f97316" },
] as const;

const alertAndAgentStats = [
  { key: "alertFiring", icon: <AlertOutlined />, accent: "#f87171" },
  { key: "alertEventsToday", icon: <CalendarOutlined />, accent: "#fbbf24" },
  { key: "logAgentsOnline", icon: <ApiOutlined />, accent: "#4ade80" },
  { key: "logAgentsOffline", icon: <DisconnectOutlined />, accent: "#94a3b8" },
] as const;

function formatUptime(seconds: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return t("dashboard.uptimeFormat", { hours, minutes });
}

const OVERVIEW_LOG_PAGE_SIZE = 14;

export function DashboardPage() {
  const { t } = useTranslation();
  const themeMode = useAdminThemeMode();
  const isLight = themeMode === "light";
  const [metrics, setMetrics] = useState<DashboardMetrics>(defaultMetrics);
  const [health, setHealth] = useState<SystemHealth>({ status: "", version: "", uptime: 0, loading: true });
  const [loading, setLoading] = useState(true);
  const [trends, setTrends] = useState<OverviewTrendsResponse | null>(null);
  const [recentOps, setRecentOps] = useState<OperationLogItem[]>([]);
  const [recentLogins, setRecentLogins] = useState<LoginLogItem[]>([]);
  const tableScrollWrapRef = useRef<HTMLDivElement>(null);
  const [tableScrollY, setTableScrollY] = useState(360);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      setHealth((prev) => ({ ...prev, loading: true }));
      try {
        const [overview, healthData, trendsData, ops, logins] = await Promise.all([
          getOverview().catch(() => null),
          getHealth().catch(() => null),
          getOverviewTrends().catch(() => null),
          getOperationLogs({ page: 1, page_size: OVERVIEW_LOG_PAGE_SIZE }).catch(() => null),
          getLoginLogs({ page: 1, page_size: OVERVIEW_LOG_PAGE_SIZE }).catch(() => null),
        ]);

        if (!active) {
          return;
        }

        if (overview) {
          setMetrics({
            users: overview.users_count,
            clusters: overview.clusters_count,
            pendingRegistrations: overview.pending_registrations_count,
            servers: overview.servers_count,
            podNormal: overview.pod_normal_count,
            podAbnormal: overview.pod_abnormal_count,
            podClusterErrors: overview.pod_cluster_errors,
            eventTotal: overview.event_total_count,
            eventWarning: overview.event_warning_count,
            eventClusterErrors: overview.event_cluster_errors,
            alertFiring: overview.alert_firing_count ?? 0,
            alertEventsToday: overview.alert_events_today_count ?? 0,
            logAgentsOnline: overview.log_agents_online_count ?? 0,
            logAgentsOffline: overview.log_agents_offline_count ?? 0,
          });
        } else {
          setMetrics(defaultMetrics);
        }

        if (healthData) {
          setHealth({
            status: healthData.status || "unknown",
            version: healthData.version || "-",
            uptime: healthData.uptime || 0,
            loading: false,
          });
        } else {
          setHealth((prev) => ({ ...prev, status: "error", loading: false }));
        }

        setTrends(trendsData);
        setRecentOps(ops?.list || []);
        setRecentLogins(logins?.list || []);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();
    return () => {
      active = false;
    };
  }, []);

  useLayoutEffect(() => {
    const el = tableScrollWrapRef.current;
    if (!el) return;
    const measure = () => {
      const h = el.clientHeight;
      if (h > 56) {
        setTableScrollY(Math.max(140, Math.floor(h - 6)));
      }
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [recentOps, recentLogins, loading]);

  const podTotal = metrics.podNormal + metrics.podAbnormal;
  const podHealthyPct = useMemo(() => {
    if (podTotal <= 0) return 0;
    return Math.round((metrics.podNormal / podTotal) * 100);
  }, [metrics.podNormal, podTotal]);

  const warnRatio = useMemo(() => {
    if (metrics.eventTotal <= 0) return 0;
    return Math.round((metrics.eventWarning / metrics.eventTotal) * 100);
  }, [metrics.eventTotal, metrics.eventWarning]);

  const agentOnlinePct = useMemo(() => {
    const t = metrics.logAgentsOnline + metrics.logAgentsOffline;
    if (t <= 0) return 0;
    return Math.round((metrics.logAgentsOnline / t) * 100);
  }, [metrics.logAgentsOnline, metrics.logAgentsOffline]);

  const uptimePct = Math.min(100, (health.uptime / 86400) * 100);

  const trendDailyRows = useMemo(() => {
    if (!trends?.days?.length) return [];
    return trends.days.map((day, i) => ({
      key: `${day}-${i}`,
      day,
      login_success: trends.login_success[i] ?? 0,
      operation_total: trends.operation_total[i] ?? 0,
      login_fail: trends.login_fail[i] ?? 0,
    }));
  }, [trends]);

  const trendTotals = useMemo(() => {
    if (!trends?.days?.length) return null;
    const sum = (arr: number[]) => arr.reduce((a, b) => a + (Number(b) || 0), 0);
    const n = trends.days.length;
    return {
      days: n,
      login_success: sum(trends.login_success),
      operation_total: sum(trends.operation_total),
      login_fail: sum(trends.login_fail),
    };
  }, [trends]);

  const ui = useMemo(
    () =>
      isLight
        ? {
            text: "#050505",
            muted: "rgba(5, 5, 5, 0.56)",
            subtle: "rgba(5, 5, 5, 0.42)",
            faint: "rgba(5, 5, 5, 0.32)",
            divider: "rgba(5, 5, 5, 0.14)",
            trail: "rgba(5, 5, 5, 0.1)",
            zero: "rgba(5, 5, 5, 0.28)",
            chartDark: false,
          }
        : {
            text: "rgba(248, 250, 252, 0.96)",
            muted: "rgba(186, 214, 238, 0.85)",
            subtle: "rgba(186, 214, 238, 0.72)",
            faint: "rgba(186, 214, 238, 0.55)",
            divider: "rgba(56, 189, 248, 0.15)",
            trail: "rgba(148, 163, 184, 0.15)",
            zero: "rgba(248, 250, 252, 0.5)",
            chartDark: true,
          },
    [isLight],
  );

  const cellTextStyle = { color: ui.text };

  const healthLine =
    health.status === "ok" || health.status === "healthy"
      ? "HEALTH / OK"
      : health.status
        ? `HEALTH / ${health.status.toUpperCase()}`
        : "HEALTH / UNKNOWN";

  return (
    <div className="overview-big-screen overview-cockpit">
      <div className="overview-big-screen__top">
        <PageTelemetryHeader
          className="overview-cockpit__header"
          label={t("dashboard.label")}
          title={t("dashboard.title")}
          subtitle={t("dashboard.subtitle")}
          meta={[
            healthLine,
            `VERSION / ${health.version || "-"}`,
            `UPTIME / ${formatUptime(health.uptime, t)}`,
            loading ? t("dashboard.syncPending") : t("dashboard.syncLive"),
          ]}
        />

        <Typography.Text className="overview-big-screen__section-label">
          <TeamOutlined /> {t("dashboard.sectionAssets")}
        </Typography.Text>
        <Row gutter={[16, 16]} className="overview-big-screen__metrics">
        {assetStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <Card className="overview-big-screen__stat-card" loading={loading} bordered={false}>
              <div className="overview-big-screen__stat-head">
                <span
                  className="overview-big-screen__stat-icon"
                  style={isLight ? undefined : { boxShadow: `0 0 24px ${item.accent}44` }}
                >
                  {item.icon}
                </span>
                <Statistic
                  title={<span className="overview-big-screen__stat-title">{t(`dashboard.stats.${item.key}.title`)}</span>}
                  value={metrics[item.key as keyof DashboardMetrics] as number}
                  valueStyle={{
                    color: item.accent,
                    fontSize: 36,
                    fontWeight: 700,
                    fontFamily: "var(--overview-num-font, ui-monospace, monospace)",
                  }}
                />
              </div>
              <Typography.Paragraph className="overview-big-screen__stat-hint">{t(`dashboard.stats.${item.key}.hint`)}</Typography.Paragraph>
            </Card>
          </Col>
        ))}
        </Row>

        <Typography.Text className="overview-big-screen__section-label">
          <ClusterOutlined /> {t("dashboard.sectionK8s")}
        </Typography.Text>
        <Row gutter={[16, 16]} className="overview-big-screen__metrics">
        {k8sStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <Card className="overview-big-screen__stat-card overview-big-screen__stat-card--k8s" loading={loading} bordered={false}>
              <div className="overview-big-screen__stat-head">
                <span
                  className="overview-big-screen__stat-icon"
                  style={isLight ? undefined : { boxShadow: `0 0 28px ${item.accent}55` }}
                >
                  {item.icon}
                </span>
                <Statistic
                  title={<span className="overview-big-screen__stat-title">{t(`dashboard.stats.${item.key}.title`)}</span>}
                  value={metrics[item.key as keyof DashboardMetrics] as number}
                  valueStyle={{
                    color: item.accent,
                    fontSize: 34,
                    fontWeight: 700,
                    fontFamily: "var(--overview-num-font, ui-monospace, monospace)",
                  }}
                />
              </div>
              <Typography.Paragraph className="overview-big-screen__stat-hint">{t(`dashboard.stats.${item.key}.hint`)}</Typography.Paragraph>
            </Card>
          </Col>
        ))}
        </Row>

        <Typography.Text className="overview-big-screen__section-label">
          <AlertOutlined /> {t("dashboard.sectionAlert")}
        </Typography.Text>
        <Row gutter={[16, 16]} className="overview-big-screen__metrics">
        {alertAndAgentStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <Card className="overview-big-screen__stat-card overview-big-screen__stat-card--alert" loading={loading} bordered={false}>
              <div className="overview-big-screen__stat-head">
                <span
                  className="overview-big-screen__stat-icon"
                  style={isLight ? undefined : { boxShadow: `0 0 28px ${item.accent}55` }}
                >
                  {item.icon}
                </span>
                <Statistic
                  title={<span className="overview-big-screen__stat-title">{t(`dashboard.stats.${item.key}.title`)}</span>}
                  value={metrics[item.key as keyof DashboardMetrics] as number}
                  valueStyle={{
                    color: item.accent,
                    fontSize: 34,
                    fontWeight: 700,
                    fontFamily: "var(--overview-num-font, ui-monospace, monospace)",
                  }}
                />
              </div>
              <Typography.Paragraph className="overview-big-screen__stat-hint">{t(`dashboard.stats.${item.key}.hint`)}</Typography.Paragraph>
            </Card>
          </Col>
        ))}
        </Row>

        <Row gutter={[16, 16]} align="stretch" className="overview-big-screen__trend-row" style={{ marginTop: 8 }}>
          <Col xs={24} xl={15} className="overview-big-screen__trend-main-col">
            <Card
              className="overview-big-screen__panel overview-big-screen__trend-main-card"
              title={
                <Space>
                  <InfoOutlined style={{ color: "#38bdf8" }} />
                  <span>{t("dashboard.trendTitle")}</span>
                </Space>
              }
              loading={loading && !trends}
              bordered={false}
              styles={{ body: { display: "flex", flexDirection: "column", flex: 1, minHeight: 0 } }}
            >
              {trends ? (
                <>
                  <LineChart
                    darkMode={ui.chartDark}
                    labels={trends.days}
                    series={[
                      { name: t("dashboard.seriesLoginSuccess"), data: trends.login_success, color: "#38bdf8" },
                      { name: t("dashboard.seriesOperationTotal"), data: trends.operation_total, color: "#34d399" },
                      { name: t("dashboard.seriesLoginFail"), data: trends.login_fail, color: "#f87171" },
                    ]}
                    height={300}
                  />
                  <Divider style={{ borderColor: ui.divider, margin: "12px 0 8px" }} />
                  {trendTotals ? (
                    <Typography.Paragraph style={{ marginBottom: 0, color: ui.subtle, fontSize: 12 }}>
                      {t("dashboard.trendSummary", {
                        days: trendTotals.days,
                        loginSuccess: trendTotals.login_success,
                        operationTotal: trendTotals.operation_total,
                        loginFail: trendTotals.login_fail,
                        avgLogin: Math.round(trendTotals.login_success / trendTotals.days),
                        avgOps: Math.round(trendTotals.operation_total / trendTotals.days),
                        avgFail: Math.round(trendTotals.login_fail / trendTotals.days),
                      })}
                    </Typography.Paragraph>
                  ) : null}
                  <div style={{ flex: 1, minHeight: 0 }} aria-hidden />
                </>
              ) : (
                <Typography.Text type="secondary" style={{ color: ui.faint }}>
                  {t("dashboard.trendEmpty")}
                </Typography.Text>
              )}
            </Card>
          </Col>
          <Col xs={24} xl={9} className="overview-big-screen__trend-rail-col">
            <div className="overview-big-screen__trend-rail">
              <Card
                className="overview-big-screen__panel overview-big-screen__trend-rail-card"
                title={
                  <Space>
                    <CheckCircleOutlined /> {t("dashboard.systemStatus")}
                  </Space>
                }
                loading={health.loading}
                bordered={false}
                styles={{ body: { flex: 1, display: "flex", flexDirection: "column", minHeight: 0 } }}
              >
                <Row gutter={[12, 12]} style={{ flex: 1 }}>
                  <Col span={24}>
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 8 }}>
                      <Typography.Text style={{ color: ui.muted }}>{t("dashboard.service")}</Typography.Text>
                      {health.status === "ok" || health.status === "healthy" ? (
                        <Tag color="success">{t("dashboard.statusOk")}</Tag>
                      ) : health.status === "error" ? (
                        <Tag color="error">{t("dashboard.statusError")}</Tag>
                      ) : (
                        <Tag color="warning">{health.status}</Tag>
                      )}
                    </div>
                  </Col>
                  <Col span={24}>
                    <div style={{ display: "flex", justifyContent: "space-between", gap: 8, flexWrap: "wrap" }}>
                      <Typography.Text style={{ color: ui.muted }}>{t("dashboard.version")}</Typography.Text>
                      <Tag style={{ marginInlineEnd: 0 }}>
                        <InfoOutlined /> {health.version}
                      </Tag>
                    </div>
                  </Col>
                  <Col span={24} style={{ marginTop: "auto" }}>
                    <Typography.Text style={{ color: ui.muted, display: "block", marginBottom: 8 }}>
                      {t("dashboard.uptime")} · {formatUptime(health.uptime, t)}
                    </Typography.Text>
                    <Progress
                      percent={uptimePct}
                      strokeColor={{ "0%": "#38bdf8", "100%": "#34d399" }}
                      trailColor={ui.trail}
                      format={(p) => `${Math.floor(((p ?? 0) / 100) * 24)}h / 24h`}
                    />
                  </Col>
                </Row>
              </Card>

              <Card
                className="overview-big-screen__panel overview-big-screen__trend-rail-card"
                title={
                  <Space>
                    <ThunderboltOutlined /> {t("dashboard.podEventSummary")}
                  </Space>
                }
                loading={loading}
                bordered={false}
                styles={{ body: { flex: 1, display: "flex", flexDirection: "column", minHeight: 0 } }}
              >
                <div className="overview-big-screen__trend-rail-pod-body">
                  <div>
                    <Typography.Text style={{ color: ui.muted }}>{t("dashboard.podHealthRatio")}</Typography.Text>
                    <Progress
                      percent={metrics.clusters === 0 ? 0 : podHealthyPct}
                      strokeColor="#34d399"
                      trailColor="rgba(251, 113, 133, 0.35)"
                      format={() => (metrics.clusters === 0 ? t("dashboard.noCluster") : `${podHealthyPct}%`)}
                    />
                  </div>
                  <div>
                    <Typography.Text style={{ color: ui.muted }}>{t("dashboard.warningEventRatio")}</Typography.Text>
                    <Progress
                      percent={metrics.eventTotal === 0 ? 0 : warnRatio}
                      strokeColor="#f97316"
                      trailColor={ui.trail}
                      format={() => (metrics.eventTotal === 0 ? t("dashboard.noSample") : `${warnRatio}%`)}
                    />
                  </div>
                  <div>
                    <Typography.Text style={{ color: ui.muted }}>{t("dashboard.agentOnlineRatio")}</Typography.Text>
                    <Progress
                      percent={metrics.logAgentsOnline + metrics.logAgentsOffline === 0 ? 0 : agentOnlinePct}
                      strokeColor="#4ade80"
                      trailColor={ui.trail}
                      format={() =>
                        metrics.logAgentsOnline + metrics.logAgentsOffline === 0 ? t("dashboard.noAgent") : `${agentOnlinePct}%`
                      }
                    />
                  </div>
                  <Divider style={{ borderColor: ui.divider, margin: "8px 0" }} />
                  {metrics.clusters > 0 && (metrics.podClusterErrors > 0 || metrics.eventClusterErrors > 0) ? (
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginTop: "auto" }}>
                      <Typography.Text style={{ color: ui.subtle }}>{t("dashboard.clusterErrors")}</Typography.Text>
                      <Tag color="warning">
                        {metrics.podClusterErrors} / {metrics.eventClusterErrors}
                      </Tag>
                    </div>
                  ) : (
                    <Typography.Text style={{ color: ui.faint, fontSize: 12, marginTop: "auto" }}>
                      {t("dashboard.eventHint")}
                    </Typography.Text>
                  )}
                </div>
              </Card>
            </div>
          </Col>
        </Row>

        {trends ? (
          <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
            <Col span={24}>
              <Card
                className="overview-big-screen__panel"
                title={
                  <Space>
                    <CalendarOutlined style={{ color: "#34d399" }} />
                    <span>{t("dashboard.dailyTitle")}</span>
                  </Space>
                }
                loading={false}
                bordered={false}
              >
                <Table
                  size="small"
                  rowKey="key"
                  pagination={false}
                  dataSource={trendDailyRows}
                  tableLayout="fixed"
                  locale={{ emptyText: t("dashboard.dailyEmpty") }}
                  columns={[
                    {
                      title: t("dashboard.colDate"),
                      dataIndex: "day",
                      width: 108,
                      render: (v: string) => <span style={cellTextStyle}>{v || "-"}</span>,
                    },
                    {
                      title: t("dashboard.colLoginSuccess"),
                      dataIndex: "login_success",
                      width: 96,
                      align: "right" as const,
                      render: (v: number) => <span style={{ ...cellTextStyle, color: "#38bdf8" }}>{v}</span>,
                    },
                    {
                      title: t("dashboard.colOperationTotal"),
                      dataIndex: "operation_total",
                      width: 96,
                      align: "right" as const,
                      render: (v: number) => <span style={{ ...cellTextStyle, color: "#34d399" }}>{v}</span>,
                    },
                    {
                      title: t("dashboard.colLoginFail"),
                      dataIndex: "login_fail",
                      width: 96,
                      align: "right" as const,
                      render: (v: number) => (
                        <span style={{ ...cellTextStyle, color: v > 0 ? "#f87171" : ui.zero }}>{v}</span>
                      ),
                    },
                  ]}
                />
              </Card>
            </Col>
          </Row>
        ) : null}
      </div>

      <div className="overview-big-screen__tables">
        <Row gutter={[16, 16]} className="overview-big-screen__tables-row">
          <Col xs={24} lg={12} className="overview-big-screen__table-col">
            <Card
              className="overview-big-screen__panel overview-big-screen__table-card overview-big-screen__table-panel"
              title={
                <Space>
                  <ProfileOutlined style={{ color: "#34d399" }} />
                  {t("dashboard.recentOps")}
                  <Typography.Text type="secondary" style={{ fontSize: 12, color: ui.faint }}>
                    {t("dashboard.recentCount", { count: OVERVIEW_LOG_PAGE_SIZE })}
                  </Typography.Text>
                </Space>
              }
              loading={loading}
              bordered={false}
            >
              <div ref={tableScrollWrapRef} className="overview-big-screen__table-scroll">
                <Table<OperationLogItem>
                  size="small"
                  rowKey="id"
                  dataSource={recentOps}
                  pagination={false}
                  tableLayout="fixed"
                  scroll={{ y: tableScrollY }}
                  columns={[
                    {
                      title: t("dashboard.colUser"),
                      dataIndex: "username",
                      width: 88,
                      render: (v: string) => <span style={cellTextStyle}>{v || "-"}</span>,
                    },
                    {
                      title: t("dashboard.colMethod"),
                      dataIndex: "method",
                      width: 72,
                      render: (v: string) => (
                        <Tag bordered={false} color="processing">
                          {v}
                        </Tag>
                      ),
                    },
                    {
                      title: t("dashboard.colPath"),
                      dataIndex: "path",
                      ellipsis: true,
                      render: (v: string) => (
                        <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: "100%", ...cellTextStyle }}>
                          {v}
                        </Typography.Text>
                      ),
                    },
                    {
                      title: t("dashboard.colTime"),
                      dataIndex: "created_at",
                      width: 156,
                      render: (v: string) => <span style={cellTextStyle}>{formatDateTime(v)}</span>,
                    },
                  ]}
                />
              </div>
            </Card>
          </Col>
          <Col xs={24} lg={12} className="overview-big-screen__table-col">
            <Card
              className="overview-big-screen__panel overview-big-screen__table-card overview-big-screen__table-panel"
              title={
                <Space>
                  <LoginOutlined style={{ color: "#38bdf8" }} />
                  {t("dashboard.recentLogins")}
                  <Typography.Text type="secondary" style={{ fontSize: 12, color: ui.faint }}>
                    {t("dashboard.recentCount", { count: OVERVIEW_LOG_PAGE_SIZE })}
                  </Typography.Text>
                </Space>
              }
              loading={loading}
              bordered={false}
            >
              <div className="overview-big-screen__table-scroll">
                <Table<LoginLogItem>
                  size="small"
                  rowKey="id"
                  dataSource={recentLogins}
                  pagination={false}
                  tableLayout="fixed"
                  scroll={{ y: tableScrollY }}
                  columns={[
                    {
                      title: t("dashboard.colUser"),
                      dataIndex: "username",
                      width: 120,
                      render: (v: string) => <span style={cellTextStyle}>{v || "-"}</span>,
                    },
                    {
                      title: t("dashboard.colStatus"),
                      dataIndex: "status",
                      width: 72,
                      render: (v: number) =>
                        v === 1 ? <Tag color="success">{t("dashboard.statusSuccess")}</Tag> : <Tag color="error">{t("dashboard.statusFail")}</Tag>,
                    },
                    {
                      title: t("dashboard.colSource"),
                      dataIndex: "source",
                      width: 96,
                      render: (v: string) => (
                        <Tag bordered={false} color="cyan">
                          {v || "-"}
                        </Tag>
                      ),
                    },
                    {
                      title: t("dashboard.colTime"),
                      dataIndex: "created_at",
                      width: 156,
                      render: (v: string) => <span style={cellTextStyle}>{formatDateTime(v)}</span>,
                    },
                  ]}
                />
              </div>
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  );
}
