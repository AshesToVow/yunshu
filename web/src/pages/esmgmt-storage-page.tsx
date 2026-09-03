import {
  DeleteOutlined,
  LinkOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Row,
  Select,
  Space,
  Statistic,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import { listEsmgmtConnections, type EsmgmtConnection } from "../services/esmgmt";
import {
  deleteESIndex,
  deleteProjectLogRetention,
  getESConfigPreview,
  getESStorageStats,
  getGlobalLogRetention,
  listLogRetentionPolicies,
  runLogRetentionCleanup,
  setLogPlatformESConnection,
  upsertGlobalLogRetention,
  upsertProjectLogRetention,
  type ESConfigPreview,
  type ESIndexStatItem,
  type ESStorageStats,
  type LogRetentionItem,
  type ProjectItem,
  getProjects,
} from "../services/log-platform";
import { extractApiErrorMessage } from "../services/http";
import { formatDateTime } from "../utils/format";

type GlobalForm = {
  retention_days: number;
  enabled: boolean;
  index_pattern?: string;
  remark?: string;
};

export function EsmgmtStoragePage() {
  const [globalForm] = Form.useForm<GlobalForm>();
  const [stats, setStats] = useState<ESStorageStats | null>(null);
  const [esCfg, setEsCfg] = useState<ESConfigPreview | null>(null);
  const [esError, setEsError] = useState("");
  const [esConnections, setEsConnections] = useState<EsmgmtConnection[]>([]);
  const [bindingES, setBindingES] = useState(false);
  const [policies, setPolicies] = useState<LogRetentionItem[]>([]);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [indexPage, setIndexPage] = useState(1);
  const [indexPageSize, setIndexPageSize] = useState(10);
  const [projectId, setProjectId] = useState<number>();
  const [projectDays, setProjectDays] = useState(30);
  const [projectEnabled, setProjectEnabled] = useState(true);

  const reload = useCallback(async () => {
    setLoading(true);
    setEsError("");
    try {
      const [global, list, cfg, proj, conns] = await Promise.all([
        getGlobalLogRetention(),
        listLogRetentionPolicies(),
        getESConfigPreview().catch(() => null),
        getProjects({ page: 1, page_size: 1000 }),
        listEsmgmtConnections().catch(() => [] as EsmgmtConnection[]),
      ]);
      setEsCfg(cfg);
      setEsConnections(conns || []);
      globalForm.setFieldsValue({
        retention_days: global.retention_days,
        enabled: global.enabled,
        index_pattern: global.index_pattern || undefined,
        remark: global.remark || undefined,
      });
      setPolicies(list.list);
      setProjects(proj.list);
      try {
        setStats(await getESStorageStats());
      } catch (e: unknown) {
        setStats(null);
        setEsError(extractApiErrorMessage(e, "无法拉取 ES 存储统计"));
      }
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e));
    } finally {
      setLoading(false);
    }
  }, [globalForm]);

  useEffect(() => {
    void reload();
  }, [reload]);

  async function bindESConnection(connectionId: number) {
    setBindingES(true);
    try {
      const cfg = await setLogPlatformESConnection(connectionId);
      setEsCfg(cfg);
      message.success(
        connectionId > 0
          ? `已绑定 ES 连接 #${connectionId}${cfg.connection_name ? `（${cfg.connection_name}）` : ""}`
          : "已回退为数据字典地址",
      );
      await reload();
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e, "绑定 ES 连接失败"));
    } finally {
      setBindingES(false);
    }
  }

  async function saveGlobal() {
    const values = await globalForm.validateFields();
    await upsertGlobalLogRetention(values);
    message.success("全局保留策略已保存");
    await reload();
  }

  async function saveProjectOverride() {
    if (!projectId) {
      message.warning("请选择项目");
      return;
    }
    await upsertProjectLogRetention(projectId, { retention_days: projectDays, enabled: projectEnabled });
    message.success("项目覆盖策略已保存");
    await reload();
  }

  async function removeProjectOverride() {
    if (!projectId) return;
    await deleteProjectLogRetention(projectId);
    message.success("已删除项目覆盖");
    await reload();
  }

  function canManageESIndex(name: string, matched?: boolean) {
    if (matched) return true;
    const n = String(name || "").trim().toLowerCase();
    return (n.startsWith("yunshu-agent-") || n.startsWith("yunshu-k8s-")) && !n.startsWith(".");
  }

  async function handleDeleteIndex(index: string) {
    try {
      await deleteESIndex(index);
      message.success(`已删除索引：${index}`);
      await reload();
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e));
    }
  }

  const projectOverrides = policies.filter((p) => p.project_id !== 0);

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="日志存储与保留"
        description="管理日志平台 ES 连接、索引存储概览与自动清理策略。"
        breadcrumbs={[{ title: "ES 管理控制台" }, { title: "日志存储" }]}
        extra={
          <Space>
            <Link to="/esmgmt/connections">连接管理</Link>
            <Link to="/esmgmt/overview">集群概览</Link>
            <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void reload()}>
              刷新
            </Button>
          </Space>
        }
      />

      <Card size="small" title="日志平台 ES 连接">
        <Space wrap align="center">
          <span>使用连接：</span>
          <Select
            style={{ minWidth: 360 }}
            loading={loading || bindingES}
            value={esCfg?.connection_id && esCfg.connection_id > 0 ? esCfg.connection_id : 0}
            options={[
              { value: 0, label: "数据字典地址（elasticsearch_addresses）" },
              ...esConnections.map((c) => ({
                value: c.id,
                label: `${c.name} (#${c.id}) — ${c.addresses}${c.is_default ? " [默认]" : ""}`,
              })),
            ]}
            onChange={(v) => void bindESConnection(Number(v) || 0)}
          />
          <Link to="/esmgmt/connections">管理连接</Link>
          <Link to="/dict-entries?keyword=elasticsearch_">索引模式字典</Link>
        </Space>
        {esCfg ? (
          <Alert
            type={esCfg.enabled ? "success" : "warning"}
            showIcon
            style={{ marginTop: 12 }}
            message={
              <Space wrap size={8}>
                <Tag color={esCfg.source === "managed" ? "purple" : "default"}>
                  {esCfg.source === "managed" ? `连接 ${esCfg.connection_name || "#" + esCfg.connection_id}` : "数据字典"}
                </Tag>
                <Tag color={esCfg.enabled ? "green" : "orange"}>{esCfg.enabled ? "enabled" : "disabled"}</Tag>
                <Tag>{(esCfg.addresses || []).join(", ") || "无地址"}</Tag>
                <Tag>{esCfg.index_pattern || "yunshu-agent-*"}</Tag>
              </Space>
            }
          />
        ) : null}
      </Card>

      <Card
        size="small"
        className="table-card"
        title="ES 存储概览"
        extra={
          <Button size="small" icon={<ReloadOutlined />} onClick={() => void reload()} loading={loading}>
            刷新
          </Button>
        }
      >
        {stats ? (
          <>
            <Row gutter={[16, 12]} style={{ marginBottom: 12 }}>
              <Col xs={12} sm={6}>
                <Statistic title="全部索引" value={stats.index_count} suffix="个" />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="全部文档" value={stats.document_count} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic title="全部占用" value={stats.store_human || "-"} />
              </Col>
              <Col xs={12} sm={6}>
                <Statistic
                  title={`平台 ${stats.index_pattern || "yunshu-*"}`}
                  value={stats.pattern_index_count ?? 0}
                  suffix={`/ ${stats.pattern_store_human || "-"}`}
                />
              </Col>
            </Row>
            <Table<ESIndexStatItem>
              size="small"
              rowKey="name"
              loading={loading}
              dataSource={stats.indices ?? []}
              scroll={{ x: 640 }}
              pagination={{
                current: indexPage,
                pageSize: indexPageSize,
                showSizeChanger: true,
                pageSizeOptions: ["10", "20", "50"],
                showTotal: (t) => `共 ${t} 个索引`,
                onChange: (p, ps) => {
                  setIndexPage(p);
                  setIndexPageSize(ps);
                },
              }}
              columns={[
                {
                  title: "索引名",
                  dataIndex: "name",
                  ellipsis: true,
                  render: (name: string, r) => (
                    <Space size={4} wrap>
                      <span>{name}</span>
                      {canManageESIndex(name, r.matched_pattern) ? <Tag color="blue">平台</Tag> : null}
                    </Space>
                  ),
                },
                { title: "文档", dataIndex: "docs_count", width: 110, render: (n: number) => (n ?? 0).toLocaleString() },
                { title: "占用", dataIndex: "store_human", width: 100 },
                {
                  title: "操作",
                  key: "action",
                  width: 88,
                  fixed: "right",
                  render: (_, r) =>
                    canManageESIndex(r.name, r.matched_pattern) ? (
                      <Popconfirm title={`确认删除 ${r.name}？`} onConfirm={() => void handleDeleteIndex(r.name)}>
                        <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                          删除
                        </Button>
                      </Popconfirm>
                    ) : (
                      <span style={{ color: "#999" }}>-</span>
                    ),
                },
              ]}
            />
          </>
        ) : (
          <Alert type="error" showIcon message={esError || "无法连接 ES，请检查连接与 elasticsearch_enabled"} />
        )}
      </Card>

      <Row gutter={[12, 12]}>
        <Col xs={24} lg={12}>
          <Card
            size="small"
            title="全局保留策略"
            extra={
              <Space>
                <Button
                  size="small"
                  icon={<PlayCircleOutlined />}
                  onClick={() =>
                    void (async () => {
                      const res = await runLogRetentionCleanup();
                      message.success(res.message || "清理完成");
                      await reload();
                    })()
                  }
                >
                  立即清理
                </Button>
                <Button size="small" type="primary" icon={<SaveOutlined />} onClick={() => void saveGlobal()}>
                  保存
                </Button>
              </Space>
            }
          >
            <Form form={globalForm} layout="vertical" size="small">
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item label="保留天数" name="retention_days" rules={[{ required: true }]}>
                    <InputNumber min={1} max={3650} style={{ width: "100%" }} addonAfter="天" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label="启用自动清理" name="enabled" valuePropName="checked">
                    <Switch />
                  </Form.Item>
                </Col>
                <Col span={24}>
                  <Form.Item label="索引模式" name="index_pattern" extra="留空=同时清理 yunshu-agent-* 与 yunshu-k8s-*">
                    <Input placeholder="默认 agent+k8s" />
                  </Form.Item>
                </Col>
                <Col span={24}>
                  <Form.Item label="备注" name="remark">
                    <Input placeholder="可选" />
                  </Form.Item>
                </Col>
              </Row>
            </Form>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card size="small" title="项目级覆盖">
            <Space wrap size={8} style={{ marginBottom: 12 }}>
              <Select
                style={{ minWidth: 200 }}
                placeholder="选择项目"
                options={projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` }))}
                value={projectId}
                onChange={setProjectId}
                allowClear
                size="small"
              />
              <InputNumber size="small" min={1} max={3650} value={projectDays} onChange={(v) => setProjectDays(v ?? 30)} addonAfter="天" />
              <span>
                启用 <Switch size="small" checked={projectEnabled} onChange={setProjectEnabled} />
              </span>
              <Button size="small" type="primary" onClick={() => void saveProjectOverride()}>
                保存
              </Button>
              <Button size="small" danger onClick={() => void removeProjectOverride()}>
                删除
              </Button>
            </Space>
            <Table
              rowKey={(r) => `${r.project_id}-${r.id}`}
              size="small"
              dataSource={projectOverrides}
              pagination={{ pageSize: 5, size: "small" }}
              columns={[
                {
                  title: "项目",
                  dataIndex: "project_id",
                  render: (id: number) => {
                    const p = projects.find((x) => x.id === id);
                    return p ? `${p.name}` : id;
                  },
                },
                { title: "天数", dataIndex: "retention_days", width: 72 },
                { title: "启用", dataIndex: "enabled", width: 72, render: (v: boolean) => (v ? "是" : "否") },
              ]}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}

export default EsmgmtStoragePage;
