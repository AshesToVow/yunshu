import type { BreadcrumbProps } from "antd";
import type { ReactNode } from "react";
import { PageTelemetryHeader } from "./page-telemetry-header";

interface PageHeroProps {
  title: string;
  subtitle?: string;
  label?: string;
  breadcrumbItems?: BreadcrumbProps["items"];
  meta?: string[];
  extra?: ReactNode;
}

export function PageHero({ title, subtitle, label, breadcrumbItems, meta, extra }: PageHeroProps) {
  const crumb =
    breadcrumbItems?.length
      ? breadcrumbItems.map((item) => String(item?.title ?? "")).filter(Boolean).join(" / ")
      : "";
  const mergedMeta = [...(crumb ? [`PATH / ${crumb}`] : []), ...(meta ?? [])];

  return (
    <PageTelemetryHeader
      label={label ?? "[ MODULE ]"}
      title={title}
      subtitle={subtitle}
      meta={mergedMeta.length ? mergedMeta : undefined}
      extra={extra}
    />
  );
}
