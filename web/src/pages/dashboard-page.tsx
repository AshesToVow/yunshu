import {
  AlertOutlined,
  ApiOutlined,
  BarChartOutlined,
  CheckCircleOutlined,
  ClusterOutlined,
  DesktopOutlined,
  DisconnectOutlined,
  ExpandOutlined,
  HistoryOutlined,
  LineChartOutlined,
  ReloadOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Alert, Button, Card, Space, Tag, Typography } from "antd";
import { Link } from "react-router-dom";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  CHART_BRAND,
  CHART_ERROR,
  CHART_MUTED,
  CHART_SECONDARY,
  CHART_SUCCESS,
  CHART_WARNING,
} from "../constants/chart-colors";
import { BarChart } from "../components/bar-chart";
import { LineChart } from "../components/line-chart";
import { DashboardStatCard } from "../components/ops/dashboard-stat-card";
import {
  getOverviewProjectLaunches,
  getOverviewReleaseByPerson,
  getOverviewScreen,
  type OverviewLabelCount,
  type OverviewProjectLaunchesResponse,
  type OverviewReleaseByPersonResponse,
  type OverviewScreenResponse,
} from "../services/overview";
import { extractApiErrorMessage } from "../services/http";

function formatClock(d: Date) {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function formatTime(iso?: string) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function severityTone(sev: string): "critical" | "warning" | "info" | "default" {
  const s = (sev || "").toLowerCase();
  if (s.includes("critical") || s.includes("fatal") || s === "p0" || s === "p1") return "critical";
  if (s.includes("warn") || s === "p2") return "warning";
  if (s.includes("info") || s.includes("low")) return "info";
  return "default";
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

function HealthGauge({ pct, label }: { pct: number; label: string }) {
  const tone = pct >= 95 ? "ok" : pct >= 80 ? "warn" : "bad";
  const r = 42;
  const c = 2 * Math.PI * r;
  const clamped = Math.max(0, Math.min(100, pct));
  const offset = c * (1 - clamped / 100);
  const stroke = tone === "ok" ? "#34d399" : tone === "warn" ? "#fbbf24" : "#f87171";

  return (
    <div className="overview-health-gauge overview-health-gauge--sm" data-tone={tone}>
      <svg viewBox="0 0 110 110" className="overview-health-gauge__svg" aria-hidden>
        <circle className="overview-health-gauge__track" cx="55" cy="55" r={r} />
        <circle
          className="overview-health-gauge__progress"
          cx="55"
          cy="55"
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

function BreakdownBars({ items, empty }: { items: OverviewLabelCount[]; empty: string }) {
  const max = Math.max(1, ...items.map((i) => Number(i.count) || 0));
  if (!items.length) {
    return <div className="overview-feed-empty">{empty}</div>;
  }
  return (
    <ul className="overview-breakdown">
      {items.map((item) => {
        const count = Number(item.count) || 0;
        const width = Math.max(6, Math.round((count / max) * 100));
        return (
          <li key={item.label || "unknown"}>
            <span className="overview-breakdown__label">{item.label || "unknown"}</span>
            <span className="overview-breakdown__track">
              <i style={{ width: `${width}%` }} />
            </span>
            <span className="overview-breakdown__count">{count.toLocaleString()}</span>
          </li>
        );
      })}
    </ul>
  );
}

export function DashboardPage() {
  const { t } = useTranslation();
  const [screen, setScreen] = useState<OverviewScreenResponse | null>(null);
  const [projectLaunches, setProjectLaunches] = useState<OverviewProjectLaunchesResponse | null>(null);
  const [releaseByPerson, setReleaseByPerson] = useState<OverviewReleaseByPersonResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [now, setNow] = useState(() => new Date());
  const [screenRef, setScreenRef] = useState<HTMLDivElement | null>(null);

  const load = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    setLoadError(null);
    try {
      const [screenData, launches, byPerson] = await Promise.all([
        getOverviewScreen(),
        getOverviewProjectLaunches(),
        getOverviewReleaseByPerson(),
      ]);
      setScreen(screenData);
      setProjectLaunches(launches);
      setReleaseByPerson(byPerson);
    } catch (e) {
      setLoadError(extractApiErrorMessage(e, "加载概览失败"));
      if (!silent) {
        setScreen(null);
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

  const kpi = screen?.kpi;
  const health = screen?.health;

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

  const enterFullscreen = () => {
    if (!screenRef) return;
    void screenRef.requestFullscreen?.();
  };

  const kpiStrip = [
    {
      key: "users",
      title: t("dashboard.stats.users.title"),
      value: kpi?.users_count ?? 0,
      icon: <TeamOutlined />,
      accent: CHART_BRAND,
      to: "/users",
    },
    {
      key: "clusters",
      title: t("dashboard.stats.clusters.title"),
      value: kpi?.clusters_count ?? 0,
      icon: <ClusterOutlined />,
      accent: CHART_BRAND,
      to: "/clusters",
    },
    {
      key: "servers",
      title: t("dashboard.stats.servers.title"),
      value: kpi?.servers_count ?? 0,
      hint:
        kpi && (kpi.servers_enabled > 0 || kpi.servers_disabled > 0)
          ? t("dashboard.serversSplit", { enabled: kpi.servers_enabled, disabled: kpi.servers_disabled })
          : t("dashboard.stats.servers.hint"),
      icon: <DesktopOutlined />,
      accent: CHART_SECONDARY,
      to: "/project-servers",
    },
    {
      key: "pods",
      title: t("dashboard.podPairTitle"),
      value: `${kpi?.pod_normal_count ?? 0} / ${kpi?.pod_abnormal_count ?? 0}`,
      hint: t("dashboard.podPairHint"),
      icon: <CheckCircleOutlined />,
      accent: (kpi?.pod_abnormal_count ?? 0) > 0 ? CHART_ERROR : CHART_SUCCESS,
      danger: (kpi?.pod_abnormal_count ?? 0) > 0,
      to: "/pods",
    },
    {
      key: "alerts",
      title: t("dashboard.stats.alertFiring.title"),
      value: kpi?.alert_firing_count ?? 0,
      hint: t("dashboard.alertTodayHint", { count: kpi?.alert_events_today_count ?? 0 }),
      icon: <AlertOutlined />,
      accent: CHART_ERROR,
      danger: (kpi?.alert_firing_count ?? 0) > 0,
      tone: "alert" as const,
      to: "/alert-monitor-platform/history",
    },
    {
      key: "loggie",
      title: t("dashboard.loggiePairTitle"),
      value: `${kpi?.loggie_agents_online_count ?? 0} / ${kpi?.loggie_agents_offline_count ?? 0}`,
      hint: t("dashboard.loggiePairHint"),
      icon: <ApiOutlined />,
      accent: (kpi?.loggie_agents_offline_count ?? 0) > 0 ? CHART_WARNING : CHART_SUCCESS,
      danger: (kpi?.loggie_agents_offline_count ?? 0) > 0,
      tone: "alert" as const,
      to: "/loggie-status",
    },
    {
      key: "pending",
      title: t("dashboard.stats.pendingRegistrations.title"),
      value: kpi?.pending_registrations_count ?? 0,
      icon: <SafetyCertificateOutlined />,
      accent: CHART_WARNING,
      danger: (kpi?.pending_registrations_count ?? 0) > 0,
      to: "/registrations",
    },
    {
      key: "ai",
      title: t("dashboard.aiTodayTitle"),
      value: kpi?.ai_investigations_today ?? 0,
      hint: t("dashboard.aiOpenHint", { count: kpi?.ai_investigations_open ?? 0 }),
      icon: <RobotOutlined />,
      accent: CHART_MUTED,
      to: "/ai/investigations",
    },
  ];

  return (
    <div ref={setScreenRef} className="overview-big-screen overview-cockpit overview-cockpit--v3">
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

      <section className="overview-kpi-strip">
        {kpiStrip.map((item) => (
          <DashboardStatCard
            key={item.key}
            variant="cockpit"
            tone={item.tone}
            compact
            danger={item.danger}
            title={item.title}
            value={item.value}
            hint={item.hint}
            icon={item.icon}
            accent={item.accent}
            loading={loading && !screen}
            to={item.to}
          />
        ))}
      </section>

      <div className="overview-cockpit__body">
        <aside className="overview-cockpit__rail overview-cockpit__rail--left">
          <PanelFrame>
            <Card
              className="overview-big-screen__panel"
              bordered={false}
              title={
                <Space>
                  <AlertOutlined />
                  <span>{t("dashboard.alertsTopTitle")}</span>
                  <Tag color="error">{(kpi?.alert_firing_count ?? 0).toLocaleString()}</Tag>
                </Space>
              }
              extra={
                <Link to="/alert-monitor-platform/history" className="overview-panel-link">
                  {t("dashboard.viewAll")}
                </Link>
              }
              loading={loading && !screen}
            >
              {(screen?.alerts_top ?? []).length > 0 ? (
                <ul className="overview-feed overview-feed--alerts">
                  {screen!.alerts_top.map((a) => (
                    <li key={a.id} data-tone={severityTone(a.severity)}>
                      <div className="overview-feed__row">
                        <strong title={a.alertname}>{a.alertname || "—"}</strong>
                        <Tag>{a.severity || "unknown"}</Tag>
                      </div>
                      <div className="overview-feed__meta">
                        <span>{a.cluster || t("dashboard.noCluster")}</span>
                        <span>{formatTime(a.starts_at)}</span>
                      </div>
                      {a.summary ? <p className="overview-feed__summary">{a.summary}</p> : null}
                    </li>
                  ))}
                </ul>
              ) : (
                <div className="overview-empty-state overview-empty-state--compact">
                  <CheckCircleOutlined />
                  <p>{t("dashboard.alertsTopEmpty")}</p>
                </div>
              )}
            </Card>
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
                  height={260}
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

          <div className="overview-cockpit__center-bottom">
            <PanelFrame>
              <Card
                className="overview-big-screen__panel"
                bordered={false}
                title={
                  <Space>
                    <HistoryOutlined />
                    <span>{t("dashboard.recentReleasesTitle")}</span>
                  </Space>
                }
                loading={loading && !screen}
              >
                {(screen?.recent_releases ?? []).length > 0 ? (
                  <ul className="overview-feed">
                    {screen!.recent_releases.map((r) => (
                      <li key={r.id}>
                        <div className="overview-feed__row">
                          <strong title={r.title}>{r.title || `#${r.id}`}</strong>
                          <Tag>{r.status}</Tag>
                        </div>
                        <div className="overview-feed__meta">
                          <span>{r.project_name || `P${r.project_id}`}</span>
                          <span>{r.submitter_name}</span>
                          <span>{formatTime(r.finished_at || r.created_at)}</span>
                        </div>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <div className="overview-feed-empty">{t("dashboard.recentReleasesEmpty")}</div>
                )}
              </Card>
            </PanelFrame>

            <PanelFrame>
              <Card
                className="overview-big-screen__panel"
                bordered={false}
                title={
                  <Space>
                    <WarningOutlined />
                    <span>{t("dashboard.recentChangesTitle")}</span>
                  </Space>
                }
                loading={loading && !screen}
              >
                {(screen?.recent_changes ?? []).length > 0 ? (
                  <ul className="overview-feed">
                    {screen!.recent_changes.map((c) => (
                      <li key={c.id}>
                        <div className="overview-feed__row">
                          <strong title={c.summary || c.action}>{c.summary || c.action || c.source}</strong>
                          <Tag color={c.risk_level === "high" || c.risk_level === "critical" ? "error" : "default"}>
                            {c.risk_level || c.status}
                          </Tag>
                        </div>
                        <div className="overview-feed__meta">
                          <span>{c.source}</span>
                          <span>{c.action}</span>
                          <span>{formatTime(c.started_at)}</span>
                        </div>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <div className="overview-feed-empty">{t("dashboard.recentChangesEmpty")}</div>
                )}
              </Card>
            </PanelFrame>
          </div>
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
              loading={loading && !screen}
            >
              <div className="overview-dual-gauge">
                <HealthGauge pct={health?.pod_health_pct ?? 100} label={t("dashboard.podHealth")} />
                <HealthGauge pct={health?.agent_online_pct ?? 100} label={t("dashboard.agentHealth")} />
              </div>
              <div className="overview-breakdown-block">
                <div className="overview-big-screen__section-label">{t("dashboard.alertSeverityTitle")}</div>
                <BreakdownBars items={health?.alert_by_severity ?? []} empty={t("dashboard.breakdownEmpty")} />
              </div>
              <div className="overview-breakdown-block">
                <div className="overview-big-screen__section-label">{t("dashboard.loggieHealthTitle")}</div>
                <BreakdownBars items={health?.loggie_by_health ?? []} empty={t("dashboard.breakdownEmpty")} />
              </div>
            </Card>
          </PanelFrame>

          <PanelFrame>
            <Card
              className="overview-big-screen__panel"
              bordered={false}
              title={
                <Space>
                  <DisconnectOutlined />
                  <span>{t("dashboard.loggieOfflineTitle")}</span>
                </Space>
              }
              extra={
                <Link to="/loggie-status" className="overview-panel-link">
                  {t("dashboard.viewAll")}
                </Link>
              }
              loading={loading && !screen}
            >
              {(screen?.loggie_offline_sample ?? []).length > 0 ? (
                <ul className="overview-feed overview-feed--compact">
                  {screen!.loggie_offline_sample.map((a) => (
                    <li key={a.id}>
                      <div className="overview-feed__row">
                        <strong>
                          S{a.server_id} · P{a.project_id}
                        </strong>
                        <Tag>{a.health_status || "offline"}</Tag>
                      </div>
                      <div className="overview-feed__meta">
                        <span>{formatTime(a.last_seen_at)}</span>
                      </div>
                      {a.last_error ? <p className="overview-feed__summary">{a.last_error}</p> : null}
                    </li>
                  ))}
                </ul>
              ) : (
                <div className="overview-feed-empty">{t("dashboard.loggieOfflineEmpty")}</div>
              )}
            </Card>
          </PanelFrame>

          <PanelFrame className="overview-cockpit__person-panel">
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
                <BarChart darkMode items={personBars} height={160} valueLabel={t("dashboard.releaseCountLabel")} />
              ) : (
                <div className="overview-feed-empty">{t("dashboard.releaseByPersonEmpty")}</div>
              )}
            </Card>
          </PanelFrame>
        </aside>
      </div>
    </div>
  );
}
