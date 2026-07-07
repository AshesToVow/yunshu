import { ArrowUpOutlined, DeleteOutlined, HistoryOutlined, RocketOutlined, RollbackOutlined } from "@ant-design/icons";
import {
  AutoComplete,
  Button,
  Card,
  Checkbox,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { K8sPageToolbar } from "../components/ops/k8s-page-toolbar";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import { useK8sContext } from "../hooks/use-k8s-context";
import { useEditGuardStore } from "../stores/edit-guard-store";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  getHelmReleaseHistory,
  getHelmReleaseValues,
  installHelmRelease,
  listHarborCharts,
  listHelmReleases,
  rollbackHelmRelease,
  uninstallHelmRelease,
  upgradeHelmRelease,
  type HelmReleaseHistoryItem,
  type HelmReleaseItem,
} from "../services/helm";

function statusColor(status: string) {
  const s = (status || "").toLowerCase();
  if (s === "deployed") return "success";
  if (s === "failed") return "error";
  if (s === "pending-install" || s === "pending-upgrade") return "processing";
  return "default";
}

const SEARCH_DEBOUNCE_MS = 400;

/** 从 Release 展示的 chart 标签（name-version）解析 Chart 名称 */
function parseChartName(chartLabel: string): string {
  const s = (chartLabel || "").trim();
  if (!s) return "";
  const m = s.match(/^(.+)-(\d+(?:\.\d+)*|v\d+(?:\.\d+)*)$/i);
  if (m) return m[1];
  return s;
}

function toChartNameOptions(names: string[]) {
  const seen = new Set<string>();
  const opts: { value: string; label: string }[] = [];
  for (const name of names) {
    const n = name.trim();
    if (!n || seen.has(n)) continue;
    seen.add(n);
    opts.push({ value: n, label: n });
  }
  return opts;
}

export function HelmReleasesPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { clusterId, setClusterId, clusterOptions } = useK8sContext({ needNamespace: false, syncUrl: true });
  const beginEdit = useEditGuardStore((s) => s.beginEdit);
  const endEdit = useEditGuardStore((s) => s.endEdit);

  const [namespace, setNamespace] = useState(() => searchParams.get("ns") ?? "");
  const [namespaceOptions, setNamespaceOptions] = useState<{ label: string; value: string }[]>([{ label: "全部命名空间", value: "" }]);
  const [keyword, setKeyword] = useState("");
  const [debouncedKeyword, setDebouncedKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<HelmReleaseItem[]>([]);

  const [historyOpen, setHistoryOpen] = useState(false);
  const [historyLoading, setHistoryLoading] = useState(false);
  const [historyRows, setHistoryRows] = useState<HelmReleaseHistoryItem[]>([]);
  const [historyTarget, setHistoryTarget] = useState<HelmReleaseItem | null>(null);

  const [valuesOpen, setValuesOpen] = useState(false);
  const [valuesText, setValuesText] = useState("");

  const [installOpen, setInstallOpen] = useState(false);
  const [installSubmitting, setInstallSubmitting] = useState(false);
  const [chartOptions, setChartOptions] = useState<{ label: string; value: string }[]>([]);
  const [installForm] = Form.useForm();

  const [upgradeOpen, setUpgradeOpen] = useState(false);
  const [upgradeSubmitting, setUpgradeSubmitting] = useState(false);
  const [upgradeTarget, setUpgradeTarget] = useState<HelmReleaseItem | null>(null);
  const [upgradeForm] = Form.useForm();

  const anyPanelOpen = historyOpen || valuesOpen || installOpen || upgradeOpen;

  useEffect(() => {
    if (!anyPanelOpen) return;
    beginEdit();
    return () => endEdit();
  }, [anyPanelOpen, beginEdit, endEdit]);

  const setNamespaceFilter = useCallback(
    (ns: string) => {
      setNamespace(ns);
      const next = new URLSearchParams(searchParams);
      if (ns) next.set("ns", ns);
      else next.delete("ns");
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams],
  );

  const loadReleases = useCallback(async () => {
    if (!clusterId) return;
    setLoading(true);
    try {
      const rows = await listHelmReleases(clusterId, namespace || undefined, debouncedKeyword || undefined);
      setList(rows ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, [clusterId, namespace, debouncedKeyword]);

  useEffect(() => {
    const delay = keyword === "" ? 0 : SEARCH_DEBOUNCE_MS;
    const id = window.setTimeout(() => setDebouncedKeyword(keyword.trim()), delay);
    return () => window.clearTimeout(id);
  }, [keyword]);

  useEffect(() => {
    if (!clusterId) return;
    void (async () => {
      try {
        const res = await listClusterNamespaces(clusterId);
        const opts = [{ label: "全部命名空间", value: "" }, ...(res.list ?? []).map((n) => ({ label: n.name, value: n.name }))];
        setNamespaceOptions(opts);
      } catch {
        setNamespaceOptions([{ label: "全部命名空间", value: "" }]);
      }
    })();
  }, [clusterId]);

  useEffect(() => {
    void loadReleases();
  }, [loadReleases]);

  const statusSummary = useMemo(() => {
    const deployed = list.filter((r) => (r.status || "").toLowerCase() === "deployed").length;
    const failed = list.filter((r) => (r.status || "").toLowerCase() === "failed").length;
    const pending = list.length - deployed - failed;
    return { total: list.length, deployed, failed, pending };
  }, [list]);

  const openHistory = async (row: HelmReleaseItem) => {
    if (!clusterId) return;
    setHistoryTarget(row);
    setHistoryOpen(true);
    setHistoryLoading(true);
    try {
      setHistoryRows(await getHelmReleaseHistory(clusterId, row.namespace, row.name));
    } catch {
      setHistoryRows([]);
    } finally {
      setHistoryLoading(false);
    }
  };

  const openValues = async (row: HelmReleaseItem) => {
    if (!clusterId) return;
    setValuesOpen(true);
    setValuesText("加载中…");
    try {
      const res = await getHelmReleaseValues(clusterId, row.namespace, row.name);
      setValuesText(JSON.stringify(res.values ?? {}, null, 2));
    } catch {
      setValuesText("");
    }
  };

  const doRollback = async (row: HelmReleaseItem, revision: number) => {
    if (!clusterId) return;
    try {
      await rollbackHelmRelease({ cluster_id: clusterId, namespace: row.namespace, release_name: row.name, revision });
      message.success(`已回滚 ${row.name} 到 revision ${revision}`);
      setHistoryOpen(false);
      void loadReleases();
    } catch {
      // toast by http
    }
  };

  const doUninstall = async (row: HelmReleaseItem) => {
    if (!clusterId) return;
    try {
      await uninstallHelmRelease(clusterId, row.namespace, row.name);
      message.success(`已卸载 ${row.name}`);
      void loadReleases();
    } catch {
      // toast
    }
  };

  const openInstall = async () => {
    setInstallOpen(true);
    installForm.resetFields();
    installForm.setFieldsValue({ namespace: namespace || "default", create_namespace: true });
    try {
      const charts = await listHarborCharts();
      setChartOptions(toChartNameOptions((charts ?? []).map((c) => c.name)));
    } catch {
      setChartOptions([]);
    }
  };

  const submitInstall = async () => {
    if (!clusterId) return;
    const v = await installForm.validateFields();
    setInstallSubmitting(true);
    try {
      await installHelmRelease({
        cluster_id: clusterId,
        namespace: v.namespace,
        release_name: v.release_name,
        chart_name: v.chart_name.trim(),
        chart_version: v.chart_version || undefined,
        create_namespace: !!v.create_namespace,
      });
      message.success("Helm 安装成功");
      setInstallOpen(false);
      void loadReleases();
    } finally {
      setInstallSubmitting(false);
    }
  };

  const openUpgrade = async (row: HelmReleaseItem) => {
    setUpgradeTarget(row);
    setUpgradeOpen(true);
    upgradeForm.resetFields();
    upgradeForm.setFieldsValue({
      release_name: row.name,
      namespace: row.namespace,
      chart_name: parseChartName(row.chart),
      chart_version: "",
      values_json: "",
    });
    try {
      const charts = await listHarborCharts();
      setChartOptions(toChartNameOptions((charts ?? []).map((c) => c.name)));
    } catch {
      setChartOptions([]);
    }
  };

  const submitUpgrade = async () => {
    if (!clusterId) return;
    const v = await upgradeForm.validateFields();
    let values: Record<string, unknown> | undefined;
    if (v.values_json?.trim()) {
      try {
        values = JSON.parse(v.values_json) as Record<string, unknown>;
      } catch {
        message.error("Values 不是合法 JSON");
        return;
      }
    }
    setUpgradeSubmitting(true);
    try {
      await upgradeHelmRelease({
        cluster_id: clusterId,
        namespace: v.namespace,
        release_name: v.release_name,
        chart_name: v.chart_name?.trim() || undefined,
        chart_version: v.chart_version || undefined,
        values,
      });
      message.success(`已升级 ${v.release_name}`);
      setUpgradeOpen(false);
      void loadReleases();
    } finally {
      setUpgradeSubmitting(false);
    }
  };

  const columns: ColumnsType<HelmReleaseItem> = [
    { title: "命名空间", dataIndex: "namespace", width: 120 },
    { title: "Release", dataIndex: "name", width: 160 },
    { title: "Chart", dataIndex: "chart", ellipsis: true },
    { title: "Revision", dataIndex: "version", width: 90 },
    {
      title: "状态",
      dataIndex: "status",
      width: 120,
      render: (v: string) => <Tag color={statusColor(v)}>{v}</Tag>,
    },
    { title: "更新时间", dataIndex: "updated", width: 180 },
    {
      title: "操作",
      key: "actions",
      width: 340,
      render: (_, row) => (
        <Space size="small" wrap>
          <Button size="small" icon={<HistoryOutlined />} onClick={() => void openHistory(row)}>
            历史
          </Button>
          <Button size="small" onClick={() => void openValues(row)}>
            Values
          </Button>
          <Button size="small" icon={<ArrowUpOutlined />} onClick={() => void openUpgrade(row)}>
            升级
          </Button>
          <Popconfirm title={`确认卸载 ${row.name}？`} onConfirm={() => void doUninstall(row)}>
            <Button size="small" danger icon={<DeleteOutlined />}>
              卸载
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Helm Release 管理"
        description="Harbor OCI Chart 安装、Release 查看与应急回滚"
        extra={<Link to="/helm/charts">Harbor Chart 目录</Link>}
        meta={
          <Space size="middle">
            <Typography.Text type="secondary">Release {statusSummary.total}</Typography.Text>
            <Typography.Text type="secondary">Deployed {statusSummary.deployed}</Typography.Text>
            {statusSummary.failed > 0 ? (
              <Typography.Text type="danger">Failed {statusSummary.failed}</Typography.Text>
            ) : null}
          </Space>
        }
      />
      <Card className="table-card yaml-crud-card" bordered={false}>
        <K8sPageToolbar
          clusterId={clusterId}
          namespace={namespace}
          clusterOptions={clusterOptions}
          namespaceOptions={namespaceOptions}
          needNamespace
          searchPlaceholder="搜索 Release / Chart"
          onClusterChange={setClusterId}
          onNamespaceChange={setNamespaceFilter}
          onSearch={(v) => {
            setKeyword(v);
            setDebouncedKeyword(v.trim());
          }}
          onRefresh={() => void loadReleases()}
          primaryAction={
            <Button type="primary" icon={<RocketOutlined />} disabled={!clusterId} onClick={() => void openInstall()}>
              从 Harbor 安装
            </Button>
          }
        />
        <Typography.Paragraph type="secondary" style={{ margin: "12px 0" }}>
          Chart 来自 Harbor OCI 仓库（数据字典 cicd_harbor_*）。业务应用回滚建议仍走 CI/CD 工单。
        </Typography.Paragraph>
        <div className="k8s-table-scroll-host">
          <Table rowKey={(r) => `${r.namespace}/${r.name}`} loading={loading} columns={columns} dataSource={list} pagination={{ pageSize: 20, showSizeChanger: true }} scroll={{ x: 1200 }} tableLayout="fixed" />
        </div>
      </Card>

      <Drawer title={`历史：${historyTarget?.name ?? ""}`} width={720} open={historyOpen} onClose={() => setHistoryOpen(false)}>
        <Table
          rowKey="revision"
          loading={historyLoading}
          size="small"
          pagination={false}
          dataSource={historyRows}
          columns={[
            { title: "Revision", dataIndex: "revision", width: 90 },
            { title: "状态", dataIndex: "status", width: 100, render: (v) => <Tag>{v}</Tag> },
            { title: "Chart", dataIndex: "chart", ellipsis: true },
            { title: "时间", dataIndex: "updated", width: 170 },
            {
              title: "操作",
              width: 100,
              render: (_, h) =>
                historyTarget && h.revision !== historyTarget.version ? (
                  <Popconfirm title={`回滚到 revision ${h.revision}？`} onConfirm={() => void doRollback(historyTarget, h.revision)}>
                    <Button size="small" icon={<RollbackOutlined />}>
                      回滚
                    </Button>
                  </Popconfirm>
                ) : (
                  <Tag color="blue">当前</Tag>
                ),
            },
          ]}
        />
      </Drawer>

      <Drawer title="Release Values" width={560} open={valuesOpen} onClose={() => setValuesOpen(false)}>
        <Input.TextArea value={valuesText} readOnly autoSize={{ minRows: 16, maxRows: 40 }} style={{ fontFamily: "monospace" }} />
      </Drawer>

      <Modal title="从 Harbor 安装 Chart" open={installOpen} onCancel={() => setInstallOpen(false)} onOk={() => void submitInstall()} confirmLoading={installSubmitting} destroyOnClose>
        <Form form={installForm} layout="vertical">
          <Form.Item name="release_name" label="Release 名称" rules={[{ required: true }]}>
            <Input placeholder="如 springbootdemo-prod" />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true }]}>
            <Input placeholder="cityos" />
          </Form.Item>
          <Form.Item name="chart_name" label="Harbor Chart" rules={[{ required: true, message: "请输入 Chart 名称" }]}>
            <AutoComplete
              options={chartOptions}
              placeholder="输入 Chart 名称，如 springbootdemo（OCI 推送名）"
              filterOption={(input, option) =>
                String(option?.value ?? "")
                  .toLowerCase()
                  .includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="chart_version" label="版本（可选，默认 latest）">
            <Input placeholder="1.0.0" />
          </Form.Item>
          <Form.Item name="create_namespace" valuePropName="checked" initialValue>
            <Checkbox>自动创建命名空间</Checkbox>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`升级 Release：${upgradeTarget?.name ?? ""}`}
        open={upgradeOpen}
        onCancel={() => setUpgradeOpen(false)}
        onOk={() => void submitUpgrade()}
        confirmLoading={upgradeSubmitting}
        destroyOnClose
        width={560}
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          留空 Chart 版本则沿用当前 Chart；填写新版本将从 Harbor OCI 拉取并升级。Values 留空表示保留现有配置；填写 JSON 将与现有 values 合并覆盖。
        </Typography.Paragraph>
        <Form form={upgradeForm} layout="vertical">
          <Form.Item name="release_name" label="Release 名称" rules={[{ required: true }]}>
            <Input disabled />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true }]}>
            <Input disabled />
          </Form.Item>
          <Form.Item name="chart_name" label="Harbor Chart（可选，默认沿用当前）">
            <AutoComplete
              allowClear
              options={chartOptions}
              placeholder="留空沿用当前 Chart，或输入新 Chart 名"
              filterOption={(input, option) =>
                String(option?.value ?? "")
                  .toLowerCase()
                  .includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="chart_version" label="目标版本（可选，默认 latest 或当前版本）">
            <Input placeholder="1.0.1" />
          </Form.Item>
          <Form.Item name="values_json" label="Values 覆盖（可选 JSON）">
            <Input.TextArea placeholder='{"replicaCount": 2}' autoSize={{ minRows: 4, maxRows: 12 }} style={{ fontFamily: "monospace" }} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
