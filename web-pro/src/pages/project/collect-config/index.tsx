// @ts-nocheck
import { PageContainer } from "@ant-design/pro-components";
import { Card, Tabs } from "antd";
import { useMemo } from "react";
import { useLocation, useNavigate, useSearchParams } from '@umijs/max';
import { ProjectClusterLogPage } from "@/pages/project-cluster-log-page";
import { ProjectLogSourcesPage } from "@/pages/project-log-sources-page";
import { ProjectServicesPage } from "@/pages/project-services-page";

/** 服务配置 + 主机日志源 + 集群采集整合页。 */
export default function ProjectCollectConfigPage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const location = useLocation();
  const tabFromPath = location.pathname.includes("log-sources")
    ? "log-sources"
    : params.get("tab") === "cluster-log"
      ? "cluster-log"
      : params.get("tab") === "log-sources"
        ? "log-sources"
        : "services";

  const items = useMemo(
    () => [
      {
        key: "services",
        label: "服务配置",
        children: <ProjectServicesPage embedded />,
      },
      {
        key: "log-sources",
        label: "主机日志源",
        children: <ProjectLogSourcesPage embedded />,
      },
      {
        key: "cluster-log",
        label: "集群采集",
        children: <ProjectClusterLogPage embedded />,
      },
    ],
    [],
  );

  return (
    <PageContainer header={{ title: "服务与日志采集", subTitle: "服务配置 · 主机日志源 · 集群采集" }}>
    <Card className="table-card" bordered={false} styles={{ body: { paddingTop: 8 } }}>
      <Tabs
        activeKey={tabFromPath}
        items={items}
        onChange={(key) => {
          if (key === "log-sources") {
            navigate("/project-services?tab=log-sources", { replace: true });
          } else if (key === "cluster-log") {
            navigate("/project-services?tab=cluster-log", { replace: true });
          } else {
            navigate("/project-services?tab=services", { replace: true });
          }
        }}
      />
    </Card>
    </PageContainer>
  );
}
