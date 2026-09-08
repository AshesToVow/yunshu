import { Card, Statistic, Typography } from "antd";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

export type DashboardStatCardProps = {
  title: ReactNode;
  value: number;
  hint?: ReactNode;
  icon: ReactNode;
  accent: string;
  loading?: boolean;
  /** 点击 KPI 下钻路由 */
  to?: string;
  className?: string;
  /** 驾驶舱大屏样式 */
  variant?: "default" | "cockpit";
  tone?: "default" | "k8s" | "alert";
};

export function DashboardStatCard({
  title,
  value,
  hint,
  icon,
  accent,
  loading,
  to,
  className,
  variant = "default",
  tone = "default",
}: DashboardStatCardProps) {
  const isCockpit = variant === "cockpit";
  const cardClass = [
    isCockpit ? "overview-big-screen__stat-card" : "dashboard-stat-card",
    isCockpit && tone === "k8s" ? "overview-big-screen__stat-card--k8s" : "",
    isCockpit && tone === "alert" ? "overview-big-screen__stat-card--alert" : "",
    to ? (isCockpit ? "overview-big-screen__stat-card--clickable" : "dashboard-stat-card--clickable") : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  const card = (
    <Card className={cardClass} loading={loading} bordered={!isCockpit} hoverable={Boolean(to)}>
      <div className={isCockpit ? "overview-big-screen__stat-head" : "dashboard-stat-card__head"}>
        <span
          className={isCockpit ? "overview-big-screen__stat-icon" : "dashboard-stat-card__icon"}
          style={
            isCockpit
              ? { color: accent, boxShadow: `0 0 24px ${accent}44`, borderColor: `${accent}55` }
              : { color: accent, backgroundColor: `${accent}14` }
          }
        >
          {icon}
        </span>
        <Statistic
          title={
            isCockpit ? <span className="overview-big-screen__stat-title">{title}</span> : title
          }
          value={value}
          valueStyle={
            isCockpit
              ? { fontSize: 28, fontWeight: 700, color: "#f8fafc", fontVariantNumeric: "tabular-nums" }
              : { fontSize: 24, fontWeight: 600 }
          }
        />
      </div>
      {hint ? (
        <Typography.Paragraph
          type={isCockpit ? undefined : "secondary"}
          className={isCockpit ? "overview-big-screen__stat-hint" : "dashboard-stat-card__hint"}
        >
          {hint}
        </Typography.Paragraph>
      ) : null}
    </Card>
  );

  if (!to) return card;
  return (
    <Link to={to} className={isCockpit ? "overview-big-screen__stat-link" : "dashboard-stat-card-link"}>
      {card}
    </Link>
  );
}
