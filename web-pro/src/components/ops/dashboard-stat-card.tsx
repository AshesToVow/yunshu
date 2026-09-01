// @ts-nocheck
import { Card, Statistic, Typography } from "antd";
import type { ReactNode } from "react";
import { Link } from '@umijs/max';

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
}: DashboardStatCardProps) {
  const card = (
    <Card
      className={["dashboard-stat-card", to ? "dashboard-stat-card--clickable" : "", className].filter(Boolean).join(" ")}
      loading={loading}
      bordered
      hoverable={Boolean(to)}
    >
      <div className="dashboard-stat-card__head">
        <span className="dashboard-stat-card__icon" style={{ color: accent, backgroundColor: `${accent}14` }}>
          {icon}
        </span>
        <Statistic
          title={title}
          value={value}
          valueStyle={{ fontSize: 24, fontWeight: 600 }}
        />
      </div>
      {hint ? (
        <Typography.Paragraph type="secondary" className="dashboard-stat-card__hint">
          {hint}
        </Typography.Paragraph>
      ) : null}
    </Card>
  );

  if (!to) return card;
  return (
    <Link to={to} className="dashboard-stat-card-link">
      {card}
    </Link>
  );
}
