// @ts-nocheck
import { Typography } from "antd";
import type { ReactNode } from "react";

export interface PageTelemetryHeaderProps {
  label: string;
  title: string;
  subtitle?: string;
  meta?: string[];
  extra?: ReactNode;
  className?: string;
}

export function PageTelemetryHeader({
  label,
  title,
  subtitle,
  meta,
  extra,
  className,
}: PageTelemetryHeaderProps) {
  const rootClass = ["page-telemetry", className].filter(Boolean).join(" ");

  return (
    <header className={rootClass}>
      <div className="page-telemetry__main">
        <div className="page-telemetry__label">{label}</div>
        <Typography.Title level={3} className="page-telemetry__title">
          {title}
        </Typography.Title>
        {subtitle ? (
          <Typography.Paragraph type="secondary" className="page-telemetry__subtitle">
            {subtitle}
          </Typography.Paragraph>
        ) : null}
      </div>
      {(meta?.length || extra) ? (
        <div className="page-telemetry__aside">
          {meta?.length ? (
            <div className="page-telemetry__meta">
              {meta.map((line) => (
                <div key={line}>{line}</div>
              ))}
            </div>
          ) : null}
          {extra ? <div className="page-telemetry__extra">{extra}</div> : null}
        </div>
      ) : null}
    </header>
  );
}
