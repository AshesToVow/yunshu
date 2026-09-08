import {
  AlertOutlined,
  ApiOutlined,
  BarChartOutlined,
  CheckCircleOutlined,
  CloudOutlined,
  ClusterOutlined,
  DesktopOutlined,
  DisconnectOutlined,
  ExpandOutlined,
  LineChartOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Alert, Button, Card, Space, Typography } from "antd";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  CHART_BRAND,
  CHART_ERROR,
  CHART_INFO,
  CHART_MUTED,
  CHART_SECONDARY,
  CHART_SUCCESS,
  CHART_WARNING,
} from "../constants/chart-colors";
import { BarChart } from "../components/bar-chart";
import { LineChart } from "../components/line-chart";
import { DashboardStatCard } from "../components/ops/dashboard-stat-card";
import {
  getOverview,
  getOverviewProjectLaunches,
  getOverviewReleaseByPerson,
  type OverviewProjectLaunchesResponse,
  type OverviewReleaseByPersonResponse,
} from "../services/overview";
import { extractApiErrorMessage } from "../services/http";

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
  loggieAgentsOnline: number;
  loggieAgentsOffline: number;
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
  loggieAgentsOnline: 0,
  loggieAgentsOffline: 0,
};

type StatDef = {
  key: keyof DashboardMetrics;
  icon: ReactNode;
  accent: string;
  tone?: "default" | "k8s" | "alert";
  dangerWhenPositive?: boolean;
};

const assetStats: StatDef[] = [
  { key: "users", icon: <TeamOutlined />, accent: CHART_BRAND },
  { key: "clusters", icon: <ClusterOutlined />, accent: CHART_BRAND },
  { key: "servers", icon: <DesktopOutlined />, accent: CHART_SECONDARY },
  { key: "pendingRegistrations", icon: <SafetyCertificateOutlined />, accent: CHART_WARNING, dangerWhenPositive: true },
];

const k8sStats: StatDef[] = [
  { key: "podNormal", icon: <CheckCircleOutlined />, accent: CHART_SUCCESS, tone: "k8s" },
  { key: "podAbnormal", icon: <WarningOutlined />, accent: CHART_ERROR, tone: "k8s", dangerWhenPositive: true },
  { key: "eventTotal", icon: <CloudOutlined />, accent: CHART_INFO, tone: "k8s" },
  { key: "eventWarning", icon: <ThunderboltOutlined />, accent: CHART_WARNING, tone: "k8s", dangerWhenPositive: true },
];

const alertAndAgentStats: StatDef[] = [
  { key: "alertFiring", icon: <AlertOutlined />, accent: CHART_ERROR, tone: "alert", dangerWhenPositive: true },
  { key: "alertEventsToday", icon: <AlertOutlined />, accent: CHART_WARNING, tone: "alert", dangerWhenPositive: true },
  { key: "loggieAgentsOnline", icon: <ApiOutlined />, accent: CHART_SUCCESS, tone: "alert" },
  { key: "loggieAgentsOffline", icon: <DisconnectOutlined />, accent: CHART_MUTED, tone: "alert", dangerWhenPositive: true },
];

const dashboardDrillDown: Partial<Record<keyof DashboardMetrics, string>> = {
  users: "/users",
  clusters: "/clusters",
  servers: "/project-servers",
  pendingRegistrations: "/registrations",
  podNormal: "/pods",
  podAbnormal: "/pods",
  eventTotal: "/events",
  eventWarning: "/events",
  alertFiring: "/alert-monitor-platform/history",
  alertEventsToday: "/alert-monitor-platform/history",
  loggieAgentsOnline: "/loggie-status",
  loggieAgentsOffline: "/loggie-status",
};

function formatClock(d: Date) {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function HealthGauge({ pct, label }: { pct: number; label: string }) {
  const tone = pct >= 95 ? "ok" : pct >= 80 ? "warn" : "bad";
  const r = 54;
  const c = 2 * Math.PI * r;
  const clamped = Math.max(0, Math.min(100, pct));
  const offset = c * (1 - clamped / 100);
  const stroke = tone === "ok" ? "#34d399" : tone === "warn" ? "#fbbf24" : "#f87171";

  return (
    <div className="overview-health-gauge" data-tone={tone}>
      <svg viewBox="0 0 140 140" className="overview-health-gauge__svg" aria-hidden>
        <circle className="overview-health-gauge__track" cx="70" cy="70" r={r} />
        <circle
          className="overview-health-gauge__progress"
          cx="70"
          cy="70"
          r={r}
          stroke={stroke}
          strokeDasharray={c}
          strokeDashoffset={offset}
        />
      </svg>
      <div className="overview-health-gauge__center">
        <strong>{clamped}%</strong>
        <span>{label}</span>
      </div>
    </div>
  );
}

function PanelFrame({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <div className={`overview-panel-frame ${className}`.trim()}>
      <span className="overview-panel-frame__corner is-tl" />
      <span className="overview-panel-frame__corner is-tr" />
      <span className="overview-panel-frame__corner is-bl" />
      <span className="overview-panel-frame__corner is-br" />
      {children}
    </div>
  );
}

function StatSection({
  label,
  icon,
  items,
  metrics,
  loading,
  t,
}: {
  label: string;
  icon: ReactNode;
  items: StatDef[];
  metrics: DashboardMetrics;
  loading: boolean;
  t: (key: string) => string;
}) {
  return (
    <section className="overview-kpi-section">
      <div className="overview-big-screen__section-label">
        {icon} {label}
      </div>
      <div className="overview-kpi-grid">
        {items.map((item) => {
          const value = metrics[item.key] as number;
          const danger = Boolean(item.dangerWhenPositive && value > 0);
          return (
            <DashboardStatCard
              key={item.key}
              variant="cockpit"
              tone={item.tone}
              compact
              danger={danger}
              title={t(`dashboard.stats.${item.key}.title`)}
              value={value}
              hint={t(`dashboard.stats.${item.key}.hint`)}
              icon={item.icon}
              accent={item.accent}
              loading={loading}
              to={dashboardDrillDown[item.key]}
            />
          );
        })}
      </div>
    </section>
  );
}

export function DashboardPage() {
  const { t } = useTranslation();
  const [metrics, setMetrics] = useState<DashboardMetrics>(defaultMetrics);
  const [loading, setLoading] = useState(true);
  const [projectLaunches, setProjectLaunches] = useState<OverviewProjectLaunchesResponse | null>(null);
  const [releaseByPerson, setReleaseByPerson] = useState<OverviewReleaseByPersonResponse | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [now, setNow] = useState(() => new Date());
  const [screenRef, setScreenRef] = useState<HTMLDivElement | null>(null);

  const load = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    setLoadError(null);
    try {
      const [overview, launches, byPerson] = await Promise.all([
        getOverview(),
        getOverviewProjectLaunches(),
        getOverviewReleaseByPerson(),
      ]);
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
        loggieAgentsOnline: overview.loggie_agents_online_count ?? 0,
        loggieAgentsOffline: overview.loggie_agents_offline_count ?? 0,
      });
      setProjectLaunches(launches);
      setReleaseByPerson(byPerson);
    } catch (e) {
      setLoadError(extractApiErrorMessage(e, "加载概览失败"));
      if (!silent) {
        setMetrics(defaultMetrics);
        setProjectLaunches(null);
        setReleaseByPerson(null);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
    const refresh = window.setInterval(() => void load(true), 30_000);
    return () => window.clearInterval(refresh);
  }, [load]);

  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const syncLabel = loading
    ? t("dashboard.syncPending")
    : loadError
      ? t("dashboard.syncFailed")
      : t("dashboard.syncLive");

  const launchSeries = useMemo(
    () =>
      (projectLaunches?.series ?? []).map((s) => ({
        name: s.project_name,
        data: s.data.map((v) => Number(v) || 0),
        color: s.color,
      })),
    [projectLaunches],
  );

  const personBars = useMemo(
    () =>
      (releaseByPerson?.items ?? []).map((item) => ({
        label: item.person || t("dashboard.unknownPerson"),
        value: Number(item.count) || 0,
      })),
    [releaseByPerson, t],
  );

  const podHealthPct = useMemo(() => {
    const total = metrics.podNormal + metrics.podAbnormal;
    if (total <= 0) return 100;
    return Math.round((metrics.podNormal / total) * 100);
  }, [metrics.podNormal, metrics.podAbnormal]);

  const enterFullscreen = () => {
    if (!screenRef) return;
    void screenRef.requestFullscreen?.();
  };

  return (
    <div ref={setScreenRef} className="overview-big-screen overview-cockpit overview-cockpit--v2">
      <header className="overview-big-screen__hero overview-cockpit__header">
        <div className="overview-big-screen__hero-main">
          <Typography.Text className="overview-big-screen__eyebrow">{t("dashboard.label")}</Typography.Text>
          <Typography.Title level={3} className="overview-big-screen__title">
            <ThunderboltOutlined />
            {t("dashboard.titleScreen")}
          </Typography.Title>
          <Typography.Text className="overview-big-screen__subtitle">{t("dashboard.subtitleScreen")}</Typography.Text>
        </div>
        <div className="overview-big-screen__hero-meta">
          <div className={`overview-big-screen__sync ${loading ? "is-pending" : loadError ? "is-failed" : "is-live"}`}>
            <span className="overview-big-screen__sync-dot" />
            {syncLabel}
          </div>
          <div className="overview-big-screen__clock">{formatClock(now)}</div>
          <Space size={8}>
            <Button
              size="small"
              icon={<ReloadOutlined spin={loading} />}
              onClick={() => void load()}
              className="overview-big-screen__action"
            >
              {t("dashboard.refresh")}
            </Button>
            <Button size="small" icon={<ExpandOutlined />} onClick={enterFullscreen} className="overview-big-screen__action">
              {t("dashboard.fullscreen")}
            </Button>
          </Space>
        </div>
      </header>

      {loadError ? (
        <Alert type="error" showIcon style={{ marginBottom: 12 }} message={t("dashboard.loadFailed")} description={loadError} />
      ) : null}

      <div className="overview-cockpit__body">
        <aside className="overview-cockpit__rail overview-cockpit__rail--left">
          <PanelFrame>
            <StatSection
              label={t("dashboard.sectionAssets")}
              icon={<TeamOutlined />}
              items={assetStats}
              metrics={metrics}
              loading={loading}
              t={t}
            />
          </PanelFrame>
          <PanelFrame>
            <StatSection
              label={t("dashboard.sectionK8s")}
              icon={<ClusterOutlined />}
              items={k8sStats}
              metrics={metrics}
              loading={loading}
              t={t}
            />
          </PanelFrame>
        </aside>

        <main className="overview-cockpit__center">
          <PanelFrame className="overview-cockpit__center-main">
            <Card
              className="overview-big-screen__panel overview-big-screen__trend-main-card"
              bordered={false}
              title={
                <Space>
                  <LineChartOutlined />
                  <span>{t("dashboard.projectLaunchTitle")}</span>
                </Space>
              }
              loading={loading && !projectLaunches}
            >
              {projectLaunches && launchSeries.length > 0 ? (
                <LineChart
                  darkMode
                  labels={projectLaunches.days}
                  series={launchSeries}
                  height={320}
                  yAxisLabel={t("dashboard.launchCountLabel")}
                />
              ) : (
                <div className="overview-empty-state">
                  <LineChartOutlined />
                  <p>{t("dashboard.projectLaunchEmpty")}</p>
                </div>
              )}
            </Card>
          </PanelFrame>
          <PanelFrame className="overview-cockpit__center-side">
            <Card
              className="overview-big-screen__panel"
              bordered={false}
              title={
                <Space>
                  <BarChartOutlined />
                  <span>{t("dashboard.releaseByPersonTitle")}</span>
                </Space>
              }
              loading={loading && !releaseByPerson}
            >
              {releaseByPerson && personBars.length > 0 ? (
                <BarChart darkMode items={personBars} height={220} valueLabel={t("dashboard.releaseCountLabel")} />
              ) : (
                <div className="overview-empty-state overview-empty-state--compact">
                  <BarChartOutlined />
                  <p>{t("dashboard.releaseByPersonEmpty")}</p>
                </div>
              )}
            </Card>
          </PanelFrame>
        </main>

        <aside className="overview-cockpit__rail overview-cockpit__rail--right">
          <PanelFrame>
            <Card
              className="overview-big-screen__panel"
              bordered={false}
              title={
                <Space>
                  <CheckCircleOutlined />
                  <span>{t("dashboard.healthTitle")}</span>
                </Space>
              }
              loading={loading}
            >
              <HealthGauge pct={podHealthPct} label={t("dashboard.podHealth")} />
              <div className="overview-big-screen__health-grid">
                <div>
                  <span>{t("dashboard.stats.podNormal.title")}</span>
                  <b>{metrics.podNormal}</b>
                </div>
                <div>
                  <span>{t("dashboard.stats.podAbnormal.title")}</span>
                  <b className="is-danger">{metrics.podAbnormal}</b>
                </div>
                <div>
                  <span>{t("dashboard.stats.alertFiring.title")}</span>
                  <b className={metrics.alertFiring > 0 ? "is-danger" : ""}>{metrics.alertFiring.toLocaleString()}</b>
                </div>
                <div>
                  <span>{t("dashboard.stats.loggieAgentsOnline.title")}</span>
                  <b className="is-ok">{metrics.loggieAgentsOnline}</b>
                </div>
              </div>
            </Card>
          </PanelFrame>
          <PanelFrame>
            <StatSection
              label={t("dashboard.sectionAlert")}
              icon={<AlertOutlined />}
              items={alertAndAgentStats}
              metrics={metrics}
              loading={loading}
              t={t}
            />
          </PanelFrame>
        </aside>
      </div>
    </div>
  );
}
