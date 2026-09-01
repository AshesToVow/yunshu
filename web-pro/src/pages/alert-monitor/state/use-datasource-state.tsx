// @ts-nocheck
/**
 * 告警监控平台：Prometheus/VictoriaMetrics 数据源（datasources Tab）状态（RF-03 第三步拆分产物）
 *
 * 从 `use-alert-monitor-platform-state.tsx` 原地搬迁，逐字保留语义：
 * - 列表固定 page_size=200（该 Tab 无分页 UI）
 * - 新建时项目默认取顶栏上下文，无上下文时取项目列表首项
 * - 列表加载完成后为 PromQL 控制台预选首条数据源（不区分启用状态）
 *
 * 注意：`loadDatasources` 仍由主 Hook 的 Tab 副作用统一调用（datasources / promql /
 * silences / rules 四个 Tab 都会拉取），因此这里只暴露方法、不自建 Tab 级 effect。
 */
import { ApiOutlined, DeleteOutlined, EditOutlined } from "@ant-design/icons";
import { Button, Form, Popconfirm, Space, Tag, message } from "antd";
import { useCallback, useMemo, useState } from "react";

import { useDictOptions } from "../../../hooks/use-dict-options";
import {
  createAlertDatasource,
  deleteAlertDatasource,
  listAlertDatasources,
  pingAlertDatasource,
  updateAlertDatasource,
  type AlertDatasourceItem,
} from "../../../services/alert-platform";
import { extractApiErrorMessage } from "../../../services/http";
import type { ProjectItem } from "../../../services/projects";

export function useAlertMonitorDatasourceState(params: {
  /** 顶栏项目上下文：列表过滤与新建默认值都以它为准 */
  projectContextId?: number;
  /** 项目列表：新建时无顶栏上下文的兜底默认项目 */
  projects: ProjectItem[];
  /**
   * 列表加载完成后为 PromQL 控制台预选首条数据源。
   *
   * 原实现直接调用 promql 控制台的 `setPromDsId((prev) => prev ?? id)`；拆分后
   * 「数据源 Hook 需要 promql 的 setter」与「promql Hook 需要 dsList」互为依赖，
   * 故由主 Hook 以稳定回调转发，语义完全一致（只在未选中时填充）。
   */
  applyDefaultPromDatasource: (firstDatasourceId?: number) => void;
}) {
  const { projectContextId, projects, applyDefaultPromDatasource } = params;

  const [dsList, setDsList] = useState<AlertDatasourceItem[]>([]);
  const [dsModalOpen, setDsModalOpen] = useState(false);
  const [dsCurrent, setDsCurrent] = useState<AlertDatasourceItem | null>(null);
  const [dsForm] = Form.useForm();
  const [dsSubmitting, setDsSubmitting] = useState(false);
  const [dsPingId, setDsPingId] = useState<number | null>(null);

  const dsUrlDictOpts = useDictOptions("alert_datasource_base_url");
  const dsBasicUserDictOpts = useDictOptions("alert_datasource_basic_user");
  const dsUrlAutoOpts = useMemo(
    () => dsUrlDictOpts.map((o) => ({ label: o.label, value: String(o.value) })),
    [dsUrlDictOpts],
  );
  const dsBasicUserAutoOpts = useMemo(
    () => dsBasicUserDictOpts.map((o) => ({ label: o.label, value: String(o.value) })),
    [dsBasicUserDictOpts],
  );

  const loadDatasources = useCallback(
    async (projectID?: number) => {
      const r = await listAlertDatasources({ project_id: projectID, page: 1, page_size: 200 });
      setDsList(r.list ?? []);
      // PromQL 控制台首次拿到数据源时预选第一条（不区分启用状态）；
      // promql Tab 的「当前选中项已失效」兜底另见 state/use-promql-console-state.ts
      applyDefaultPromDatasource(r.list?.[0]?.id);
    },
    [applyDefaultPromDatasource],
  );

  async function runDsPing(id: number) {
    setDsPingId(id);
    try {
      const res = await pingAlertDatasource(id);
      if (res.ok) {
        message.success(`连通正常，耗时 ${res.latency_ms} ms`);
      } else {
        message.error(res.message || "连通失败");
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "操作失败"));
    } finally {
      setDsPingId(null);
    }
  }

  const dsColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "项目", dataIndex: "project_name", width: 160, render: (v: string, r: AlertDatasourceItem) => v || String(r.project_id || "-") },
    { title: "名称", dataIndex: "name" },
    {
      title: "类型",
      dataIndex: "type",
      width: 120,
      render: (v: string) => {
        const t = (v || "prometheus").toLowerCase();
        if (t === "victoria" || t === "victoriametrics") return <Tag color="blue">VictoriaMetrics</Tag>;
        return <Tag>Prometheus</Tag>;
      },
    },
    { title: "地址", dataIndex: "base_url", ellipsis: true },
    { title: "启用", dataIndex: "enabled", width: 80, render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>) },
    {
      title: "操作",
      width: 240,
      render: (_: unknown, r: AlertDatasourceItem) => (
        <Space wrap>
          <Button
            type="link"
            size="small"
            icon={<ApiOutlined />}
            loading={dsPingId === r.id}
            onClick={() => void runDsPing(r.id)}
          >
            连通检测
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openDsEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除数据源？" onConfirm={() => void removeDs(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openDsCreate() {
    setDsCurrent(null);
    dsForm.resetFields();
    const fallbackProjectID = projectContextId ?? projects[0]?.id;
    dsForm.setFieldsValue({ project_id: fallbackProjectID, type: "prometheus", skip_tls_verify: false, enabled: true });
    setDsModalOpen(true);
  }

  function openDsEdit(r: AlertDatasourceItem) {
    setDsCurrent(r);
    dsForm.setFieldsValue({
      project_id: r.project_id,
      name: r.name,
      type: r.type,
      base_url: r.base_url,
      alertmanager_url: r.alertmanager_url ?? "",
      basic_user: r.basic_user ?? "",
      skip_tls_verify: r.skip_tls_verify,
      enabled: r.enabled,
      remark: r.remark,
    });
    setDsModalOpen(true);
  }

  async function submitDs() {
    setDsSubmitting(true);
    try {
      const v = await dsForm.validateFields();
      if (dsCurrent) {
        await updateAlertDatasource(dsCurrent.id, v);
        message.success("已更新");
      } else {
        await createAlertDatasource(v);
        message.success("已创建");
      }
      setDsModalOpen(false);
      await loadDatasources(projectContextId);
    } catch (e) {
      if (e && typeof e === "object" && "errorFields" in e) return;
      message.error(extractApiErrorMessage(e, "保存数据源失败"));
    } finally {
      setDsSubmitting(false);
    }
  }

  async function removeDs(id: number) {
    try {
      await deleteAlertDatasource(id);
      message.success("已删除");
      await loadDatasources(projectContextId);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "删除数据源失败"));
    }
  }

  return {
    dsBasicUserAutoOpts,
    dsColumns,
    dsCurrent,
    dsForm,
    dsList,
    dsModalOpen,
    dsSubmitting,
    dsUrlAutoOpts,
    loadDatasources,
    openDsCreate,
    setDsModalOpen,
    submitDs,
  };
}
