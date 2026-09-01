// @ts-nocheck
import { PageContainer } from "@ant-design/pro-components";
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
import { Card, Col, Row, Space, Typography, Alert } from "antd";
import { useEffect, useMemo, useState } from "react";
import {
  CHART_BRAND,
  CHART_ERROR,
  CHART_INFO,
  CHART_MUTED,
  CHART_SECONDARY,
  CHART_SUCCESS,
  CHART_WARNING,
} from "@/constants/chart-colors";
import { useTranslation } from "react-i18next";
import { BarChart } from "@/components/bar-chart";
import { LineChart } from "@/components/line-chart";
import { DashboardStatCard } from "@/components/ops/dashboard-stat-card";
import {
  getOverview,
  getOverviewProjectLaunches,
  getOverviewReleaseByPerson,
  type OverviewProjectLaunchesResponse,
  type OverviewReleaseByPersonResponse,
} from "@/services/overview";
import { extractApiErrorMessage } from "@/services/http";

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

export default function DashboardPage() {
  const { t } = useTranslation();
  const [metrics, setMetrics] = useState<DashboardMetrics>(defaultMetrics);
  const [loading, setLoading] = useState(true);
  const [projectLaunches, setProjectLaunches] = useState<OverviewProjectLaunchesResponse | null>(null);
  const [releaseByPerson, setReleaseByPerson] = useState<OverviewReleaseByPersonResponse | null>(null);

  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      setLoadError(null);
      try {
        const [overview, launches, byPerson] = await Promise.all([
          getOverview(),
          getOverviewProjectLaunches(),
          getOverviewReleaseByPerson(),
        ]);

        if (!active) {
          return;
        }

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
        if (active) {
          setLoadError(extractApiErrorMessage(e, "加载概览失败"));
          setMetrics(defaultMetrics);
          setProjectLaunches(null);
          setReleaseByPerson(null);
        }
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
    <PageContainer
      header={{
        title: t("dashboard.title"),
        subTitle: t("dashboard.subtitle"),
      }}
      className="dashboard-page"
    >
      {loadError ? (
        <Alert type="error" showIcon style={{ marginBottom: 16 }} message="概览数据加载失败" description={loadError} />
      ) : null}
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
    </PageContainer>
  );
}