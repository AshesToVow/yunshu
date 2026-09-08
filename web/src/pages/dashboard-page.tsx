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
import { Alert, Button, Card, Col, Row, Space, Typography } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
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

const assetStats = [
  { key: "users", icon: <TeamOutlined />, accent: CHART_BRAND },
  { key: "clusters", icon: <ClusterOutlined />, accent: CHART_BRAND },
  { key: "servers", icon: <DesktopOutlined />, accent: CHART_SECONDARY },
  { key: "pendingRegistrations", icon: <SafetyCertificateOutlined />, accent: CHART_WARNING },
] as const;

const k8sStats = [
  { key: "podNormal", icon: <CheckCircleOutlined />, accent: CHART_SUCCESS },
  { key: "podAbnormal", icon: <WarningOutlined />, accent: CHART_ERROR },
  { key: "eventTotal", icon: <CloudOutlined />, accent: CHART_INFO },
  { key: "eventWarning", icon: <ThunderboltOutlined />, accent: CHART_WARNING },
] as const;

const alertAndAgentStats = [
  { key: "alertFiring", icon: <AlertOutlined />, accent: CHART_ERROR },
  { key: "alertEventsToday", icon: <AlertOutlined />, accent: CHART_WARNING },
  { key: "loggieAgentsOnline", icon: <ApiOutlined />, accent: CHART_SUCCESS },
  { key: "loggieAgentsOffline", icon: <DisconnectOutlined />, accent: CHART_MUTED },
] as const;

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

export function DashboardPage() {
  const { t } = useTranslation();
  const [metrics, setMetrics] = useState<DashboardMetrics>(defaultMetrics);
  const [loading, setLoading] = useState(true);
  const [projectLaunches, setProjectLaunches] = useState<OverviewProjectLaunchesResponse | null>(null);
  const [releaseByPerson, setReleaseByPerson] = useState<OverviewReleaseByPersonResponse | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [now, setNow] = useState(() => new Date());
  const [screenRef, setScreenRef] = useState<HTMLDivElement | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
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
      setMetrics(defaultMetrics);
      setProjectLaunches(null);
      setReleaseByPerson(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
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
    const el = screenRef;
    if (!el) return;
    void el.requestFullscreen?.();
  };

  return (
    <div
      ref={setScreenRef}
      className="overview-big-screen overview-cockpit"
    >
      <div className="overview-big-screen__top">
        <div className="overview-big-screen__hero overview-cockpit__header">
          <div className="overview-big-screen__hero-main">
            <Typography.Text className="overview-big-screen__eyebrow">
              {t("dashboard.label")}
            </Typography.Text>
            <Typography.Title level={3} className="overview-big-screen__title">
              <ThunderboltOutlined />
              {t("dashboard.titleScreen")}
            </Typography.Title>
            <Typography.Text className="overview-big-screen__subtitle">
              {t("dashboard.subtitleScreen")}
            </Typography.Text>
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
              <Button
                size="small"
                icon={<ExpandOutlined />}
                onClick={enterFullscreen}
                className="overview-big-screen__action"
              >
                {t("dashboard.fullscreen")}
              </Button>
            </Space>
          </div>
        </div>

        {loadError ? (
          <Alert
            type="error"
            showIcon
            style={{ marginBottom: 16 }}
            message={t("dashboard.loadFailed")}
            description={loadError}
          />
        ) : null}

        <Typography.Text className="overview-big-screen__section-label">
          <TeamOutlined /> {t("dashboard.sectionAssets")}
        </Typography.Text>
        <Row gutter={[16, 16]} className="overview-big-screen__metrics">
          {assetStats.map((item) => (
            <Col xs={24} sm={12} xl={6} key={item.key}>
              <DashboardStatCard
                variant="cockpit"
                title={t(`dashboard.stats.${item.key}.title`)}
                value={metrics[item.key]}
                hint={t(`dashboard.stats.${item.key}.hint`)}
                icon={item.icon}
                accent={item.accent}
                loading={loading}
                to={dashboardDrillDown[item.key]}
              />
            </Col>
          ))}
        </Row>

        <Typography.Text className="overview-big-screen__section-label">
          <ClusterOutlined /> {t("dashboard.sectionK8s")}
        </Typography.Text>
        <Row gutter={[16, 16]} className="overview-big-screen__metrics">
          {k8sStats.map((item) => (
            <Col xs={24} sm={12} xl={6} key={item.key}>
              <DashboardStatCard
                variant="cockpit"
                tone="k8s"
                title={t(`dashboard.stats.${item.key}.title`)}
                value={metrics[item.key]}
                hint={t(`dashboard.stats.${item.key}.hint`)}
                icon={item.icon}
                accent={item.accent}
                loading={loading}
                to={dashboardDrillDown[item.key]}
              />
            </Col>
          ))}
        </Row>

        <Typography.Text className="overview-big-screen__section-label">
          <AlertOutlined /> {t("dashboard.sectionAlert")}
        </Typography.Text>
        <Row gutter={[16, 16]} className="overview-big-screen__metrics">
          {alertAndAgentStats.map((item) => (
            <Col xs={24} sm={12} xl={6} key={item.key}>
              <DashboardStatCard
                variant="cockpit"
                tone="alert"
                title={t(`dashboard.stats.${item.key}.title`)}
                value={metrics[item.key]}
                hint={t(`dashboard.stats.${item.key}.hint`)}
                icon={item.icon}
                accent={item.accent}
                loading={loading}
                to={dashboardDrillDown[item.key]}
              />
            </Col>
          ))}
        </Row>
      </div>

      <Row gutter={[16, 16]} align="stretch" className="overview-big-screen__trend-row" style={{ marginTop: 8 }}>
        <Col xs={24} xl={15} className="overview-big-screen__trend-main-col">
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
                height={360}
                yAxisLabel={t("dashboard.launchCountLabel")}
              />
            ) : (
              <Typography.Text className="overview-big-screen__empty">
                {t("dashboard.projectLaunchEmpty")}
              </Typography.Text>
            )}
          </Card>
        </Col>
        <Col xs={24} xl={9} className="overview-big-screen__trend-rail-col">
          <div className="overview-big-screen__trend-rail">
            <Card
              className="overview-big-screen__panel overview-big-screen__trend-rail-card"
              bordered={false}
              title={
                <Space>
                  <CheckCircleOutlined />
                  <span>{t("dashboard.healthTitle")}</span>
                </Space>
              }
              loading={loading}
            >
              <div className="overview-big-screen__trend-rail-pod-body">
                <div className="overview-big-screen__health-ring" data-tone={podHealthPct >= 95 ? "ok" : podHealthPct >= 80 ? "warn" : "bad"}>
                  <strong>{podHealthPct}%</strong>
                  <span>{t("dashboard.podHealth")}</span>
                </div>
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
                    <b className={metrics.alertFiring > 0 ? "is-danger" : ""}>{metrics.alertFiring}</b>
                  </div>
                  <div>
                    <span>{t("dashboard.stats.loggieAgentsOnline.title")}</span>
                    <b className="is-ok">{metrics.loggieAgentsOnline}</b>
                  </div>
                </div>
              </div>
            </Card>
            <Card
              className="overview-big-screen__panel overview-big-screen__trend-rail-card"
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
                <BarChart
                  darkMode
                  items={personBars}
                  height={220}
                  valueLabel={t("dashboard.releaseCountLabel")}
                />
              ) : (
                <Typography.Text className="overview-big-screen__empty">
                  {t("dashboard.releaseByPersonEmpty")}
                </Typography.Text>
              )}
            </Card>
          </div>
        </Col>
      </Row>
    </div>
  );
}
