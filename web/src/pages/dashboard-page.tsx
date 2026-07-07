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
import { Card, Col, Row, Space, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { BarChart } from "../components/bar-chart";
import { LineChart } from "../components/line-chart";
import { DashboardStatCard } from "../components/ops/dashboard-stat-card";
import { OpsPageHeader } from "../components/ops/ops-page-header";
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
  { key: "users", icon: <TeamOutlined />, accent: "#1677ff" },
  { key: "clusters", icon: <ClusterOutlined />, accent: "#1677ff" },
  { key: "servers", icon: <DesktopOutlined />, accent: "#722ed1" },
  { key: "pendingRegistrations", icon: <SafetyCertificateOutlined />, accent: "#fa8c16" },
] as const;

const k8sStats = [
  { key: "podNormal", icon: <CheckCircleOutlined />, accent: "#52c41a" },
  { key: "podAbnormal", icon: <WarningOutlined />, accent: "#ff4d4f" },
  { key: "eventTotal", icon: <CloudOutlined />, accent: "#1677ff" },
  { key: "eventWarning", icon: <ThunderboltOutlined />, accent: "#fa8c16" },
] as const;

const alertAndAgentStats = [
  { key: "alertFiring", icon: <AlertOutlined />, accent: "#ff4d4f" },
  { key: "alertEventsToday", icon: <AlertOutlined />, accent: "#fa8c16" },
  { key: "logAgentsOnline", icon: <ApiOutlined />, accent: "#52c41a" },
  { key: "logAgentsOffline", icon: <DisconnectOutlined />, accent: "#8c8c8c" },
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
  logAgentsOnline: "/agent-list",
  logAgentsOffline: "/agent-list",
};

export function DashboardPage() {
  const { t } = useTranslation();
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
    () => ({
      chartDark: false,
    }),
    [],
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
    <div className="page-stack dashboard-page">
      <OpsPageHeader
        title={t("dashboard.title")}
        description={t("dashboard.subtitle")}
        meta={
          <Typography.Text type="secondary">
            {loading ? t("dashboard.syncPending") : t("dashboard.syncLive")}
          </Typography.Text>
        }
      />

      <Typography.Title level={5} className="dashboard-section-title">
        <TeamOutlined /> {t("dashboard.sectionAssets")}
      </Typography.Title>
      <Row gutter={[16, 16]}>
        {assetStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <DashboardStatCard
              title={t(`dashboard.stats.${item.key}.title`)}
              value={metrics[item.key as keyof DashboardMetrics] as number}
              hint={t(`dashboard.stats.${item.key}.hint`)}
              icon={item.icon}
              accent={item.accent}
              loading={loading}
              to={dashboardDrillDown[item.key]}
            />
          </Col>
        ))}
      </Row>

      <Typography.Title level={5} className="dashboard-section-title">
        <ClusterOutlined /> {t("dashboard.sectionK8s")}
      </Typography.Title>
      <Row gutter={[16, 16]}>
        {k8sStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <DashboardStatCard
              title={t(`dashboard.stats.${item.key}.title`)}
              value={metrics[item.key as keyof DashboardMetrics] as number}
              hint={t(`dashboard.stats.${item.key}.hint`)}
              icon={item.icon}
              accent={item.accent}
              loading={loading}
              to={dashboardDrillDown[item.key]}
            />
          </Col>
        ))}
      </Row>

      <Typography.Title level={5} className="dashboard-section-title">
        <AlertOutlined /> {t("dashboard.sectionAlert")}
      </Typography.Title>
      <Row gutter={[16, 16]}>
        {alertAndAgentStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <DashboardStatCard
              title={t(`dashboard.stats.${item.key}.title`)}
              value={metrics[item.key as keyof DashboardMetrics] as number}
              hint={t(`dashboard.stats.${item.key}.hint`)}
              icon={item.icon}
              accent={item.accent}
              loading={loading}
              to={dashboardDrillDown[item.key]}
            />
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 8 }}>
        <Col xs={24} xl={14}>
          <Card
            className="table-card"
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
                darkMode={ui.chartDark}
                labels={projectLaunches.days}
                series={launchSeries}
                height={360}
                yAxisLabel={t("dashboard.launchCountLabel")}
              />
            ) : (
              <Typography.Text type="secondary">{t("dashboard.projectLaunchEmpty")}</Typography.Text>
            )}
          </Card>
        </Col>
        <Col xs={24} xl={10}>
          <Card
            className="table-card"
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
                darkMode={ui.chartDark}
                items={personBars}
                height={360}
                valueLabel={t("dashboard.releaseCountLabel")}
              />
            ) : (
              <Typography.Text type="secondary">{t("dashboard.releaseByPersonEmpty")}</Typography.Text>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
}
