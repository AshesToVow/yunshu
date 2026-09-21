import { Tabs } from "antd";
import { useLocation, useNavigate } from "react-router-dom";

const TABS = [
  { key: "/workflow/inbox", label: "我的待办" },
  { key: "/workflow/tickets", label: "全部工单" },
  { key: "/workflow/definitions", label: "流程配置" },
] as const;

/**
 * 成熟审批中心三件套导航：待办（审批）/ 工单（审计）/ 流程（配置）。
 * 执行类动作不放在此中心，统一深链到业务详情。
 */
export function ApprovalCenterNav() {
  const navigate = useNavigate();
  const { pathname, search } = useLocation();
  const active = TABS.find((t) => pathname.startsWith(t.key))?.key ?? "/workflow/inbox";

  return (
    <Tabs
      activeKey={active}
      style={{ marginBottom: 12 }}
      items={TABS.map((t) => ({ key: t.key, label: t.label }))}
      onChange={(key) => {
        // 保留 domain / project 查询，便于域内跳转后仍可筛选
        const params = new URLSearchParams(search);
        const next = new URLSearchParams();
        const domain = params.get("domain");
        const project = params.get("project");
        if (domain) next.set("domain", domain);
        if (project) next.set("project", project);
        const qs = next.toString();
        navigate(qs ? `${key}?${qs}` : key);
      }}
    />
  );
}
