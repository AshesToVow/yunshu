import { Card, Tabs } from "antd";
import { useMemo } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { ProjectLogSourcesPage } from "./project-log-sources-page";
import { ProjectServicesPage } from "./project-services-page";

/** 服务配置 + 日志源配置整合页（Tab）。兼容旧路由 /project-services、/project-log-sources。 */
export function ProjectCollectConfigPage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const location = useLocation();
  const tab =
    location.pathname.includes("log-sources") || params.get("tab") === "log-sources" ? "log-sources" : "services";

  const items = useMemo(
    () => [
      {
        key: "services",
        label: "服务配置",
        children: <ProjectServicesPage embedded />,
      },
      {
        key: "log-sources",
        label: "日志源配置",
        children: <ProjectLogSourcesPage embedded />,
      },
    ],
    [],
  );

  return (
    <Card className="table-card" title="服务与日志源" styles={{ body: { paddingTop: 8 } }}>
      <Tabs
        activeKey={tab}
        items={items}
        onChange={(key) => {
          if (key === "log-sources") {
            navigate("/project-services?tab=log-sources", { replace: true });
          } else {
            navigate("/project-services?tab=services", { replace: true });
          }
        }}
      />
    </Card>
  );
}
