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
import { Alert, Button, Space, Tag, Typography } from "antd";
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
  getOverview,
  getOverviewProjectLaunches,
  getOverviewReleaseByPerson,
  getOverviewScreen,
  type OverviewLabelCount,
  type OverviewProjectLaunchesResponse,
  type OverviewReleaseByPersonResponse,
  type OverviewResponse,
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
  if (s.includes("warn") || s === "p2" || s === "high") return "warning";
  if (s.includes("info") || s.includes("low")) return "info";
  return "default";
}

function Panel({
  title,
  icon,
  extra,
  children,
  className = "",
}: {
  title: ReactNode;
  icon?: ReactNode;
  extra?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={`overview-panel ${className}`.trim()}>
      <span className="overview-panel__corner is-tl" />
      <span className="overview-panel__corner is-tr" />
      <span className="overview-panel__corner is-bl" />
      <span className="overview-panel__corner is-br" />
      <header className="overview-panel__head">
        <div className="overview-panel__title">
          {icon}
          <span>{title}</span>
        </div>
        {extra ? <div className="overview-panel__extra">{extra}</div> : null}
      </header>
      <div className="overview-panel__body">{children}</div>
    </section>
  );
}

function HealthGauge({ pct, label }: { pct: number; label: string }) {
  const tone = pct >= 95 ? "ok" : pct >= 80 ? "warn" : "bad";
  const r = 40;
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
    return <div className="overview-feed-empty overview-feed-empty--sm">{empty}</div>;
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

type Settled<T> = { ok: true; value: T } | { ok: false; error: unknown };

async function settled<T>(p: Promise<T>): Promise<Settled<T>> {
  try {
    return { ok: true, value: await p };
  } catch (error) {
    return { ok: false, error };
  }
}

export function DashboardPage() {
  const { t } = useTranslation();
  const [screen, setScreen] = useState<OverviewScreenResponse | null>(null);
  const [fallback, setFallback] = useState<OverviewResponse | null>(null);
  const [projectLaunches, setProjectLaunches] = useState<OverviewProjectLaunchesResponse | null>(null);
  const [releaseByPerson, setReleaseByPerson] = useState<OverviewReleaseByPersonResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [now, setNow] = useState(() => new Date());
  const [screenRef, setScreenRef] = useState<HTMLDivElement | null>(null);

  const load = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    const [screenRes, launchesRes, byPersonRes, overviewRes] = await Promise.all([
      settled(getOverviewScreen()),
      settled(getOverviewProjectLaunches()),
      settled(getOverviewReleaseByPerson()),
      settled(getOverview()),
    ]);

    const errors: string[] = [];
    if (screenRes.ok) {
      setScreen(screenRes.value);
    } else {
      errors.push(extractApiErrorMessage(screenRes.error, "screen"));
      if (!silent) setScreen(null);
    }
    if (launchesRes.ok) setProjectLaunches(launchesRes.value);
    else if (!silent) setProjectLaunches(null);
    if (byPersonRes.ok) setReleaseByPerson(byPersonRes.value);
    else if (!silent) setReleaseByPerson(null);
    if (overviewRes.ok) setFallback(overviewRes.value);
    else if (!silent) setFallback(null);

    setLoadError(errors.length ? errors.join("；") : null);
    setLoading(false);
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

  const kpi = useMemo(() => {
    if (screen?.kpi) return screen.kpi;
    if (!fallback) return null;
    return {
      users_count: fallback.users_count,
      clusters_count: fallback.clusters_count,
      servers_count: fallback.servers_count,
      servers_enabled: 0,
      servers_disabled: 0,
      pending_registrations_count: fallback.pending_registrations_count,
      pod_normal_count: fallback.pod_normal_count,
      pod_abnormal_count: fallback.pod_abnormal_count,
      event_warning_count: fallback.event_warning_count,
      alert_firing_count: fallback.alert_firing_count ?? 0,
      alert_events_today_count: fallback.alert_events_today_count ?? 0,
      loggie_agents_online_count: fallback.loggie_agents_online_count ?? 0,
      loggie_agents_offline_count: fallback.loggie_agents_offline_count ?? 0,
      ai_investigations_today: 0,
      ai_investigations_open: 0,
    };
  }, [screen, fallback]);

  const health = screen?.health;
  const hasLaunchChart = Boolean(projectLaunches && (projectLaunches.series?.length ?? 0) > 0);

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
    <div ref={setScreenRef} className="overview-big-screen overview-cockpit overview-cockpit--v4">
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
        <Alert type="warning" showIcon style={{ marginBottom: 12 }} message={t("dashboard.loadPartial")} description={loadError} />
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
            loading={loading && !kpi}
            to={item.to}
          />
        ))}
      </section>

      <div className={`overview-cockpit__body ${hasLaunchChart ? "has-chart" : "no-chart"}`}>
        <Panel
          className="overview-cockpit__alerts"
          icon={<AlertOutlined />}
          title={
            <>
              {t("dashboard.alertsTopTitle")}
              <Tag color="error">{(kpi?.alert_firing_count ?? 0).toLocaleString()}</Tag>
            </>
          }
          extra={
            <Link to="/alert-monitor-platform/history" className="overview-panel-link">
              {t("dashboard.viewAll")}
            </Link>
          }
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
        </Panel>

        <div className="overview-cockpit__center">
          {hasLaunchChart ? (
            <Panel
              className="overview-cockpit__chart"
              icon={<LineChartOutlined />}
              title={t("dashboard.projectLaunchTitle")}
            >
              <LineChart
                darkMode
                labels={projectLaunches!.days}
                series={launchSeries}
                height={220}
                yAxisLabel={t("dashboard.launchCountLabel")}
              />
            </Panel>
          ) : null}

          <div className="overview-cockpit__feeds">
            <Panel icon={<HistoryOutlined />} title={t("dashboard.recentReleasesTitle")}>
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
            </Panel>

            <Panel icon={<WarningOutlined />} title={t("dashboard.recentChangesTitle")}>
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
            </Panel>
          </div>
        </div>

        <div className="overview-cockpit__rail overview-cockpit__rail--right">
          <Panel icon={<CheckCircleOutlined />} title={t("dashboard.healthTitle")}>
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
          </Panel>

          <Panel
            icon={<DisconnectOutlined />}
            title={t("dashboard.loggieOfflineTitle")}
            extra={
              <Link to="/loggie-status" className="overview-panel-link">
                {t("dashboard.viewAll")}
              </Link>
            }
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
              <div className="overview-feed-empty overview-feed-empty--sm">{t("dashboard.loggieOfflineEmpty")}</div>
            )}
          </Panel>

          <Panel icon={<BarChartOutlined />} title={t("dashboard.releaseByPersonTitle")} className="overview-cockpit__person-panel">
            {releaseByPerson && personBars.length > 0 ? (
              <BarChart darkMode items={personBars} height={200} valueLabel={t("dashboard.releaseCountLabel")} />
            ) : (
              <div className="overview-feed-empty overview-feed-empty--sm">{t("dashboard.releaseByPersonEmpty")}</div>
            )}
          </Panel>
        </div>
      </div>
    </div>
  );
}
