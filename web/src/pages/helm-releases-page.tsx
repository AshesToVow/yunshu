import { DeleteOutlined, HistoryOutlined, ReloadOutlined, RollbackOutlined, RocketOutlined } from "@ant-design/icons";
import {
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
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { getClusters, listNamespaces as listClusterNamespaces, type ClusterItem } from "../services/clusters";
import {
  getHelmReleaseHistory,
  getHelmReleaseValues,
  installHelmRelease,
  listHarborCharts,
  listHelmReleases,
  rollbackHelmRelease,
  uninstallHelmRelease,
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

export function HelmReleasesPage() {
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number>();
  const [namespace, setNamespace] = useState<string>("");
  const [namespaceOptions, setNamespaceOptions] = useState<{ label: string; value: string }[]>([]);
  const [keyword, setKeyword] = useState("");
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

  const clusterOptions = useMemo(
    () => clusters.map((c) => ({ label: c.status === 1 ? c.name : `${c.name}（已停用）`, value: c.id, disabled: c.status !== 1 })),
    [clusters],
  );

  const loadReleases = useCallback(async () => {
    if (!clusterId) return;
    setLoading(true);
    try {
      const rows = await listHelmReleases(clusterId, namespace || undefined, keyword.trim() || undefined);
      setList(rows ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, [clusterId, namespace, keyword]);

  useEffect(() => {
    void (async () => {
      try {
        const res = await getClusters({ page: 1, page_size: 200 });
        setClusters(res.list ?? []);
        const first = (res.list ?? []).find((c) => c.status === 1);
        if (first) setClusterId(first.id);
      } catch {
        setClusters([]);
      }
    })();
  }, []);

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
      setChartOptions((charts ?? []).map((c) => ({ label: `${c.name} (${c.latest_version})`, value: c.name })));
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
        chart_name: v.chart_name,
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
      width: 280,
      render: (_, row) => (
        <Space size="small" wrap>
          <Button size="small" icon={<HistoryOutlined />} onClick={() => void openHistory(row)}>
            历史
          </Button>
          <Button size="small" onClick={() => void openValues(row)}>
            Values
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
    <>
      <PageTelemetryHeader
        label="[ K8S / HELM ]"
        title="Helm Release 管理"
        subtitle="Harbor OCI Chart 安装、Release 查看与应急回滚"
      />
      <Card>
        <Space wrap style={{ marginBottom: 16 }}>
          <Select
            style={{ width: 200 }}
            placeholder="选择集群"
            options={clusterOptions}
            value={clusterId}
            onChange={(v) => setClusterId(v)}
          />
          <Select style={{ width: 180 }} options={namespaceOptions} value={namespace} onChange={setNamespace} />
          <Input.Search
            allowClear
            placeholder="搜索 Release / Chart"
            style={{ width: 220 }}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onSearch={() => void loadReleases()}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void loadReleases()}>
            刷新
          </Button>
          <Button type="primary" icon={<RocketOutlined />} disabled={!clusterId} onClick={() => void openInstall()}>
            从 Harbor 安装
          </Button>
        </Space>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          Chart 来自 Harbor OCI 仓库（数据字典 cicd_harbor_*）。业务应用回滚建议仍走 CI/CD 工单；此处用于运维查看与应急操作。
        </Typography.Paragraph>
        <Table rowKey={(r) => `${r.namespace}/${r.name}`} loading={loading} columns={columns} dataSource={list} pagination={{ pageSize: 20 }} />
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
          <Form.Item name="chart_name" label="Harbor Chart" rules={[{ required: true }]}>
            <Select showSearch options={chartOptions} placeholder="选择 Chart" />
          </Form.Item>
          <Form.Item name="chart_version" label="版本（可选，默认 latest）">
            <Input placeholder="1.0.0" />
          </Form.Item>
          <Form.Item name="create_namespace" valuePropName="checked" initialValue>
            <Checkbox>自动创建命名空间</Checkbox>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
