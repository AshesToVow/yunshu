import {
  AlertOutlined,
  ApiOutlined,
  BarChartOutlined,
  CheckCircleOutlined,
  CloudOutlined,
  ClusterOutlined,
  DesktopOutlined,
  DisconnectOutlined,
  LineChartOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Card, Col, Row, Space, Statistic, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { BarChart } from "../components/bar-chart";
import { LineChart } from "../components/line-chart";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { useAdminThemeMode } from "../hooks/use-admin-theme-mode";
import {
  getOverview,
  getOverviewProjectLaunches,
  getOverviewReleaseByPerson,
  type OverviewProjectLaunchesResponse,
  type OverviewReleaseByPersonResponse,
} from "../services/overview";

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
  { key: "alertEventsToday", icon: <AlertOutlined />, accent: "#fbbf24" },
  { key: "logAgentsOnline", icon: <ApiOutlined />, accent: "#4ade80" },
  { key: "logAgentsOffline", icon: <DisconnectOutlined />, accent: "#94a3b8" },
] as const;

export function DashboardPage() {
  const { t } = useTranslation();
  const themeMode = useAdminThemeMode();
  const isLight = themeMode === "light";
  const [metrics, setMetrics] = useState<DashboardMetrics>(defaultMetrics);
  const [loading, setLoading] = useState(true);
  const [projectLaunches, setProjectLaunches] = useState<OverviewProjectLaunchesResponse | null>(null);
  const [releaseByPerson, setReleaseByPerson] = useState<OverviewReleaseByPersonResponse | null>(null);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      try {
        const [overview, launches, byPerson] = await Promise.all([
          getOverview().catch(() => null),
          getOverviewProjectLaunches().catch(() => null),
          getOverviewReleaseByPerson().catch(() => null),
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

        setProjectLaunches(launches);
        setReleaseByPerson(byPerson);
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

  const ui = useMemo(
    () =>
      isLight
        ? {
            faint: "rgba(5, 5, 5, 0.32)",
            chartDark: false,
          }
        : {
            faint: "rgba(186, 214, 238, 0.55)",
            chartDark: true,
          },
    [isLight],
  );

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

  return (
    <div className="overview-big-screen overview-cockpit">
      <div className="overview-big-screen__top">
        <PageTelemetryHeader
          className="overview-cockpit__header"
          label={t("dashboard.label")}
          title={t("dashboard.title")}
          subtitle={t("dashboard.subtitle")}
          meta={[loading ? t("dashboard.syncPending") : t("dashboard.syncLive")]}
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

        <Row gutter={[16, 16]} className="overview-big-screen__trend-row" style={{ marginTop: 8 }}>
          <Col xs={24} xl={14}>
            <Card
              className="overview-big-screen__panel overview-big-screen__trend-main-card"
              title={
                <Space>
                  <LineChartOutlined style={{ color: "#38bdf8" }} />
                  <span>{t("dashboard.projectLaunchTitle")}</span>
                </Space>
              }
              loading={loading && !projectLaunches}
              bordered={false}
            >
              {projectLaunches && launchSeries.length > 0 ? (
                <LineChart
                  darkMode={ui.chartDark}
                  labels={projectLaunches.days}
                  series={launchSeries}
                  height={360}
                  yAxisLabel={t("dashboard.launchCountLabel")}
                />
              ) : (
                <Typography.Text type="secondary" style={{ color: ui.faint }}>
                  {t("dashboard.projectLaunchEmpty")}
                </Typography.Text>
              )}
            </Card>
          </Col>
          <Col xs={24} xl={10}>
            <Card
              className="overview-big-screen__panel overview-big-screen__trend-main-card"
              title={
                <Space>
                  <BarChartOutlined style={{ color: "#3b82f6" }} />
                  <span>{t("dashboard.releaseByPersonTitle")}</span>
                </Space>
              }
              loading={loading && !releaseByPerson}
              bordered={false}
            >
              {releaseByPerson && personBars.length > 0 ? (
                <BarChart
                  darkMode={ui.chartDark}
                  items={personBars}
                  height={360}
                  valueLabel={t("dashboard.releaseCountLabel")}
                />
              ) : (
                <Typography.Text type="secondary" style={{ color: ui.faint }}>
                  {t("dashboard.releaseByPersonEmpty")}
                </Typography.Text>
              )}
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  );
}
