import { Breadcrumb, Space, Typography } from "antd";
import type { ReactNode } from "react";

export type OpsBreadcrumbItem = {
  title: string;
  href?: string;
};

export type OpsPageHeaderProps = {
  title: string;
  description?: string;
  breadcrumbs?: OpsBreadcrumbItem[];
  extra?: ReactNode;
  meta?: ReactNode;
};

export function OpsPageHeader({ title, description, breadcrumbs, extra, meta }: OpsPageHeaderProps) {
  return (
    <header className="ops-page-header">
      <div className="ops-page-header__main">
        {breadcrumbs?.length ? (
          <Breadcrumb
            className="ops-page-header__breadcrumb"
            items={breadcrumbs.map((item) => ({
              title: item.href ? <a href={item.href}>{item.title}</a> : item.title,
            }))}
          />
        ) : null}
        <Typography.Title level={4} className="ops-page-header__title">
          {title}
        </Typography.Title>
        {description ? (
          <Typography.Paragraph type="secondary" className="ops-page-header__description">
            {description}
          </Typography.Paragraph>
        ) : null}
        {meta ? <div className="ops-page-header__meta">{meta}</div> : null}
      </div>
      {extra ? <Space wrap className="ops-page-header__extra">{extra}</Space> : null}
    </header>
  );
}
