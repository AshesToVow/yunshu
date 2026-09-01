// @ts-nocheck
/**
 * 告警监控平台：云资源到期规则（cloud-expiry Tab）状态（RF-03 第二步拆分产物）
 *
 * 从 `use-alert-monitor-platform-state.tsx` 原地搬迁，逐字保留语义：
 * - 列表加载固定 page_size=200（该 Tab 无分页 UI，筛选走 provider/keyword）
 * - provider/keyword 为空串时不下发参数，交给后端按「全部」处理
 * - 表格列的项目名回退顺序：接口 project_name → 本地 projects 匹配 → project_id
 *
 * 注意：`loadCloudExpiryRules` 仍由主 Hook 的 Tab 副作用统一调用（受 `loading` 包裹），
 * 因此这里只暴露方法、不自建 Tab 级 effect，避免同一 Tab 触发两次请求。
 */
import { DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { Button, Form, Popconfirm, Space, Tag, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useState } from "react";

import { stringifyPrettyJSON } from "../../../services/alert-mappers";
import {
  createCloudExpiryRule,
  deleteCloudExpiryRule,
  evaluateCloudExpiryRulesNow,
  listCloudExpiryRules,
  updateCloudExpiryRule,
  type CloudExpiryRuleItem,
} from "../../../services/alert-platform";
import type { ProjectItem } from "../../../services/projects";
import { formatDateTime } from "../../../utils/format";

export function useAlertMonitorCloudExpiryState(params: {
  /** 顶栏项目上下文：新建默认值与列表过滤都以它为准 */
  projectContextId?: number;
  /** 项目列表：仅用于表格「项目」列在接口未回填 project_name 时兜底展示 */
  projects: ProjectItem[];
}) {
  const { projectContextId, projects } = params;

  const [cloudExpiryList, setCloudExpiryList] = useState<CloudExpiryRuleItem[]>([]);
  const [cloudExpiryModalOpen, setCloudExpiryModalOpen] = useState(false);
  const [cloudExpiryCurrent, setCloudExpiryCurrent] = useState<CloudExpiryRuleItem | null>(null);
  const [cloudExpiryForm] = Form.useForm();
  const [cloudExpirySubmitting, setCloudExpirySubmitting] = useState(false);
  const [cloudExpiryEvaluating, setCloudExpiryEvaluating] = useState(false);
  const [cloudExpiryProviderFilter, setCloudExpiryProviderFilter] = useState<string>("");
  const [cloudExpiryKeyword, setCloudExpiryKeyword] = useState<string>("");

  const loadCloudExpiryRules = useCallback(async (projectID?: number, provider?: string, keyword?: string) => {
    const r = await listCloudExpiryRules({
      project_id: projectID,
      provider: String(provider || "").trim() || undefined,
      keyword: String(keyword || "").trim() || undefined,
      page: 1,
      page_size: 200,
    });
    setCloudExpiryList(r.list ?? []);
  }, []);

  function openCloudExpiryCreate() {
    setCloudExpiryCurrent(null);
    cloudExpiryForm.resetFields();
    cloudExpiryForm.setFieldsValue({
      project_id: projectContextId,
      provider: "",
      region_scope: "",
      advance_days: 7,
      severity: "warning",
      eval_cron_spec: "0 9 * * *",
      schedule_enabled: true,
      labels_json: "{}",
      enabled: true,
    });
    setCloudExpiryModalOpen(true);
  }

  function openCloudExpiryEdit(row: CloudExpiryRuleItem) {
    setCloudExpiryCurrent(row);
    cloudExpiryForm.setFieldsValue({
      project_id: row.project_id,
      name: row.name,
      provider: row.provider || "",
      region_scope: row.region_scope || "",
      advance_days: row.advance_days,
      severity: row.severity || "warning",
      eval_cron_spec: row.eval_cron_spec ?? "",
      schedule_enabled: row.schedule_enabled !== false,
      labels_json: stringifyPrettyJSON(row.labels ?? {}, "{}"),
      enabled: row.enabled,
    });
    setCloudExpiryModalOpen(true);
  }

  async function submitCloudExpiryRule() {
    setCloudExpirySubmitting(true);
    try {
      const v = await cloudExpiryForm.validateFields();
      const payload = {
        ...v,
        provider: String(v.provider || "").trim(),
        region_scope: String(v.region_scope || "").trim(),
        labels_json: String(v.labels_json || "{}").trim() || "{}",
        eval_cron_spec: String(v.eval_cron_spec ?? "").trim(),
      };
      if (cloudExpiryCurrent) {
        await updateCloudExpiryRule(cloudExpiryCurrent.id, payload);
        message.success("已更新云到期规则");
      } else {
        await createCloudExpiryRule(payload);
        message.success("已创建云到期规则");
      }
      setCloudExpiryModalOpen(false);
      await loadCloudExpiryRules(projectContextId, cloudExpiryProviderFilter, cloudExpiryKeyword);
    } finally {
      setCloudExpirySubmitting(false);
    }
  }

  async function removeCloudExpiryRule(id: number) {
    await deleteCloudExpiryRule(id);
    message.success("已删除");
    await loadCloudExpiryRules(projectContextId, cloudExpiryProviderFilter, cloudExpiryKeyword);
  }

  async function runCloudExpiryEvalNow() {
    setCloudExpiryEvaluating(true);
    try {
      await evaluateCloudExpiryRulesNow();
      message.success({
        content:
          "评估已完成。历史记录仅在存在「剩余天数 ≤ 提前天数」的实例时产生 firing；无到期实例则不会有新记录。请在历史记录中搜索规则名，或数据源选「云资源到期」；未配置 encryption_key 时接口会报错。",
        duration: 9,
      });
    } finally {
      setCloudExpiryEvaluating(false);
    }
  }

  const cloudExpiryColumns: ColumnsType<CloudExpiryRuleItem> = [
    { title: "ID", dataIndex: "id", width: 70 },
    {
      title: "项目",
      dataIndex: "project_name",
      width: 180,
      ellipsis: true,
      render: (v: string, r: CloudExpiryRuleItem) => {
        const name = String(v || "").trim();
        if (name) return name;
        const p = projects.find((it) => it.id === r.project_id);
        return p ? `${p.name} (${p.code})` : String(r.project_id || "-");
      },
    },
    { title: "规则名", dataIndex: "name", width: 180 },
    {
      title: "厂商",
      dataIndex: "provider",
      width: 110,
      render: (v: string) => {
        const p = String(v || "").trim();
        if (!p) return "全部";
        if (p === "alibaba") return "阿里云";
        if (p === "tencent") return "腾讯云";
        if (p === "jd") return "京东云";
        return p;
      },
    },
    { title: "地域范围", dataIndex: "region_scope", width: 180, render: (v: string) => String(v || "").trim() || "全部" },
    { title: "提前天数", dataIndex: "advance_days", width: 100 },
    { title: "级别", dataIndex: "severity", width: 90 },
    { title: "定时", dataIndex: "schedule_enabled", width: 80, render: (v: boolean) => (v !== false ? <Tag color="blue">开</Tag> : <Tag>关</Tag>) },
    {
      title: "Cron",
      dataIndex: "eval_cron_spec",
      width: 160,
      ellipsis: true,
      render: (v: string) => {
        const s = String(v || "").trim();
        return s ? <span title={s}>{s}</span> : <span style={{ color: "#999" }}>—</span>;
      },
    },
    { title: "启用", dataIndex: "enabled", width: 80, render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>) },
    { title: "创建时间", dataIndex: "created_at", width: 170, render: (v: string) => (v ? formatDateTime(v) : "-") },
    { title: "更新时间", dataIndex: "updated_at", width: 170, render: (v: string) => (v ? formatDateTime(v) : "-") },
    {
      title: "操作",
      width: 180,
      fixed: "right",
      render: (_: unknown, r: CloudExpiryRuleItem) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openCloudExpiryEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除云到期规则？" onConfirm={() => void removeCloudExpiryRule(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return {
    cloudExpiryColumns,
    cloudExpiryCurrent,
    cloudExpiryEvaluating,
    cloudExpiryForm,
    cloudExpiryKeyword,
    cloudExpiryList,
    cloudExpiryModalOpen,
    cloudExpiryProviderFilter,
    cloudExpirySubmitting,
    loadCloudExpiryRules,
    openCloudExpiryCreate,
    runCloudExpiryEvalNow,
    setCloudExpiryKeyword,
    setCloudExpiryModalOpen,
    setCloudExpiryProviderFilter,
    submitCloudExpiryRule,
  };
}
