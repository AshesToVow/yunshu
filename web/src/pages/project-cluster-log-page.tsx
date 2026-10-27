import { CloudUploadOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  message,
} from "antd";
import { useCallback, useEffect, useRef, useState } from "react";
import { getClusters, type ClusterItem } from "../services/clusters";
import {
  createClusterLogRule,
  deleteClusterLogRule,
  deployClusterLog,
  getProjects,
  listClusterLogAgents,
  listClusterLogRules,
  previewClusterLogPipelines,
  refreshClusterLogStatus,
  saveClusterLogPipelines,
  updateClusterLogRule,
  type ClusterLogAgent,
  type ClusterLogRule,
  type ProjectItem,
} from "../services/log-platform";
import { extractApiErrorMessage } from "../services/http";

function parseJSONList(raw?: string): string {
  if (!raw) return "";
  try {
    const arr = JSON.parse(raw) as string[];
    return Array.isArray(arr) ? arr.join(", ") : raw;
  } catch {
    return raw;
  }
}

function splitCSV(v?: string): string[] {
  if (!v) return [];
  return v
    .split(/[,;\n]/)
    .map((s) => s.trim())
    .filter(Boolean);
}

function isFormValidationError(error: unknown): boolean {
  return typeof error === "object" && error !== null && "errorFields" in error;
}

/** 集群采集：规则 + DaemonSet 下发（与主机 Loggie 并列）。 */
export function ProjectClusterLogPage({ embedded }: { embedded?: boolean }) {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number>();
  const [rules, setRules] = useState<ClusterLogRule[]>([]);
  const [agents, setAgents] = useState<ClusterLogAgent[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<ClusterLogRule | null>(null);
  const [pipelineOpen, setPipelineOpen] = useState(false);
  const [pipelineText, setPipelineText] = useState("");
  const [pipelineGenerated, setPipelineGenerated] = useState("");
  const [pipelineCustom, setPipelineCustom] = useState(false);
  const [pipelineSaving, setPipelineSaving] = useState(false);
  const [ruleSubmitting, setRuleSubmitting] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [deploying, setDeploying] = useState(false);
  const [actingRuleId, setActingRuleId] = useState<number | null>(null);
  const [refreshingAgentId, setRefreshingAgentId] = useState<number | null>(null);
  const [rateLimitQps, setRateLimitQps] = useState<number>(2000);
  const loadSeqRef = useRef(0);
  const [form] = Form.useForm();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 1000 })
      .then((p) => {
        const list = p?.list || [];
        setProjects(list);
        if (list[0]) setProjectId(list[0].id);
      })
      .catch((e: unknown) => message.error(extractApiErrorMessage(e, "加载项目列表失败")));
    void getClusters({ page: 1, page_size: 1000 })
      .then((res) => {
        const list = res?.list || [];
        setClusters(list);
        if (list[0]) setClusterId(list[0].id);
      })
      .catch((e: unknown) => message.error(extractApiErrorMessage(e, "加载集群列表失败")));
  }, []);

  const load = useCallback(async () => {
    const seq = ++loadSeqRef.current;
    if (!projectId) {
      setRules([]);
      setAgents([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const [r, a] = await Promise.all([listClusterLogRules(projectId, clusterId), listClusterLogAgents(projectId)]);
      if (seq !== loadSeqRef.current) return;
      setRules(r?.list || []);
      setAgents(a?.list || []);
      const cur = (a?.list || []).find((x) => x.cluster_id === clusterId);
      setRateLimitQps(cur?.rate_limit_qps && cur.rate_limit_qps > 0 ? cur.rate_limit_qps : 2000);
    } catch (e: unknown) {
      if (seq !== loadSeqRef.current) return;
      message.error(extractApiErrorMessage(e, "加载集群采集失败"));
    } finally {
      if (seq === loadSeqRef.current) setLoading(false);
    }
  }, [projectId, clusterId]);

  useEffect(() => {
    void load();
  }, [load]);

  function resetClusterState() {
    loadSeqRef.current += 1;
    setRules([]);
    setAgents([]);
    setRateLimitQps(2000);
    setPipelineText("");
    setPipelineGenerated("");
    setPipelineCustom(false);
    setPipelineOpen(false);
    setEditing(null);
    setOpen(false);
  }

  function handleProjectChange(id: number) {
    resetClusterState();
    setProjectId(id);
  }

  function handleClusterChange(id: number) {
    resetClusterState();
    setClusterId(id);
  }

  function openCreate() {
    if (ruleSubmitting) return;
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ parse_profile: "cri", enabled: true, rate_limit_qps: 0 });
    setOpen(true);
  }

  function openEdit(row: ClusterLogRule) {
    if (ruleSubmitting) return;
    setEditing(row);
    form.setFieldsValue({
      name: row.name,
      match_namespaces: parseJSONList(row.match_namespaces),
      match_workloads: parseJSONList(row.match_workloads),
      exclude_namespaces: parseJSONList(row.exclude_namespaces),
      parse_profile: row.parse_profile || "cri",
      rate_limit_qps: row.rate_limit_qps || 0,
      enabled: row.enabled,
      remark: row.remark,
    });
    setOpen(true);
  }

  async function onSubmitRule() {
    if (!projectId || !clusterId || ruleSubmitting) return;
    setRuleSubmitting(true);
    try {
      const values = await form.validateFields();
      const payload = {
        name: values.name as string,
        match_namespaces: splitCSV(values.match_namespaces),
        match_workloads: splitCSV(values.match_workloads),
        exclude_namespaces: splitCSV(values.exclude_namespaces),
        parse_profile: (values.parse_profile as string) || "cri",
        rate_limit_qps: Number(values.rate_limit_qps) || 0,
        enabled: Boolean(values.enabled ?? true),
        remark: (values.remark as string) || "",
      };
      if (editing) {
        await updateClusterLogRule(projectId, editing.id, payload);
        message.success("规则已更新");
      } else {
        await createClusterLogRule(projectId, {
          cluster_id: clusterId,
          ...payload,
        });
        message.success("规则已创建");
      }
      setOpen(false);
      setEditing(null);
      void load();
    } catch (e) {
      if (isFormValidationError(e)) return;
      message.error(extractApiErrorMessage(e, editing ? "更新失败" : "创建失败"));
    } finally {
      setRuleSubmitting(false);
    }
  }

  async function onPreview() {
    if (!projectId || !clusterId || previewing) return;
    setPreviewing(true);
    try {
      const res = await previewClusterLogPipelines(projectId, clusterId);
      setPipelineText(res?.pipelines_yml || "");
      setPipelineGenerated(res?.generated_yml || res?.pipelines_yml || "");
      setPipelineCustom(Boolean(res?.is_custom));
      setPipelineOpen(true);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "预览失败"));
    } finally {
      setPreviewing(false);
    }
  }

  async function onSavePipelines(apply: boolean, reset = false) {
    if (!projectId || !clusterId || pipelineSaving) return;
    setPipelineSaving(true);
    try {
      const res = await saveClusterLogPipelines(projectId, {
        cluster_id: clusterId,
        pipelines_yml: reset ? undefined : pipelineText,
        reset,
        apply,
        namespace: "yunshu-logging",
        rate_limit_qps: rateLimitQps,
      });
      setPipelineText(res?.pipelines_yml || "");
      setPipelineGenerated(res?.generated_yml || "");
      setPipelineCustom(Boolean(res?.is_custom));
      if (reset) {
        message.success(apply ? "已恢复自动生成并下发" : "已恢复自动生成");
      } else {
        message.success(apply ? "已保存并下发" : "已保存自定义 pipelines.yml");
      }
      if (apply) void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "保存失败"));
    } finally {
      setPipelineSaving(false);
    }
  }

  async function onDeploy() {
    if (!projectId || !clusterId || deploying) return;
    setDeploying(true);
    try {
      const agent = await deployClusterLog(projectId, {
        cluster_id: clusterId,
        namespace: "yunshu-logging",
        rate_limit_qps: rateLimitQps,
      });
      message.success(
        `已下发 DaemonSet yunshu-loggie-p${projectId}（ready ${agent.ready_replicas}/${agent.desired_replicas}，QPS ${agent.rate_limit_qps ?? rateLimitQps}）。若集群仍有旧名 yunshu-loggie，可手工删除。`,
      );
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "部署失败"));
    } finally {
      setDeploying(false);
    }
  }

  async function onToggleRule(row: ClusterLogRule, checked: boolean) {
    if (!projectId || actingRuleId != null) return;
    setActingRuleId(row.id);
    try {
      await updateClusterLogRule(projectId, row.id, { enabled: checked });
      message.success(checked ? "规则已启用" : "规则已停用");
      await load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "切换规则状态失败"));
    } finally {
      setActingRuleId(null);
    }
  }

  async function onDeleteRule(row: ClusterLogRule) {
    if (!projectId || actingRuleId != null) return;
    setActingRuleId(row.id);
    try {
      await deleteClusterLogRule(projectId, row.id);
      message.success("规则已删除");
      await load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "删除规则失败"));
    } finally {
      setActingRuleId(null);
    }
  }

  async function onRefreshAgent(row: ClusterLogAgent) {
    if (!projectId || refreshingAgentId != null) return;
    setRefreshingAgentId(row.cluster_id);
    try {
      await refreshClusterLogStatus(projectId, row.cluster_id);
      message.success("状态已刷新");
      await load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "刷新失败"));
    } finally {
      setRefreshingAgentId(null);
    }
  }

  const body = (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <div className="toolbar">
        <Space wrap>
          <Select
            style={{ minWidth: 200 }}
            placeholder="项目"
            value={projectId}
            options={projects.map((p) => ({ value: p.id, label: p.name }))}
            onChange={handleProjectChange}
          />
          <Select
            style={{ minWidth: 220 }}
            placeholder="集群"
            value={clusterId}
            options={clusters.map((c) => ({ value: c.id, label: c.name }))}
            onChange={handleClusterChange}
          />
          <span>
            项目限流 QPS{" "}
            <InputNumber
              min={100}
              max={100000}
              step={100}
              value={rateLimitQps}
              onChange={(v) => setRateLimitQps(Number(v) || 2000)}
            />
          </span>
          <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
            刷新
          </Button>
          <Button disabled={!clusterId || !projectId} loading={previewing} onClick={() => void onPreview()}>
            编辑 Pipeline
          </Button>
          <Button
            type="primary"
            icon={<CloudUploadOutlined />}
            disabled={!clusterId || !projectId}
            loading={deploying}
            onClick={() => void onDeploy()}
          >
            部署/同步 DaemonSet
          </Button>
          <Button icon={<PlusOutlined />} disabled={!clusterId || !projectId} onClick={openCreate}>
            新建规则
          </Button>
        </Space>
      </div>
      <Card
        className="table-card"
        size="small"
        title="采集规则（按 Namespace / Workload；宽采排除系统 ns；未填 QPS 的规则均分项目配额）"
      >
        <Table
          size="small"
          rowKey="id"
          loading={loading}
          dataSource={rules}
          pagination={{ pageSize: 8 }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "名称", dataIndex: "name" },
            { title: "Namespaces", dataIndex: "match_namespaces", render: parseJSONList, ellipsis: true },
            { title: "排除 ns", dataIndex: "exclude_namespaces", render: parseJSONList, ellipsis: true, width: 160 },
            { title: "Workloads", dataIndex: "match_workloads", render: parseJSONList, ellipsis: true },
            { title: "解析", dataIndex: "parse_profile", width: 90 },
            {
              title: "配置 QPS",
              dataIndex: "rate_limit_qps",
              width: 90,
              render: (v?: number) => (v && v > 0 ? v : "均分"),
            },
            {
              title: "生效 QPS",
              dataIndex: "allocated_qps",
              width: 90,
              render: (v?: number, row?: ClusterLogRule) => (row?.enabled ? (v ?? "—") : "—"),
            },
            {
              title: "启用",
              dataIndex: "enabled",
              width: 80,
              render: (v: boolean, row?: ClusterLogRule) => (
                <Switch
                  size="small"
                  checked={v}
                  loading={!!row && actingRuleId === row.id}
                  disabled={!row || actingRuleId != null}
                  onChange={(checked) => (row ? void onToggleRule(row, checked) : undefined)}
                />
              ),
            },
            {
              title: "操作",
              width: 120,
              fixed: "right",
              render: (_: unknown, row?: ClusterLogRule) =>
                row ? (
                  <Space size={0}>
                    <Button type="link" size="small" disabled={ruleSubmitting} onClick={() => openEdit(row)}>
                      编辑
                    </Button>
                    <Popconfirm
                      title="删除规则？"
                      okText="删除"
                      cancelText="取消"
                      disabled={actingRuleId != null}
                      onConfirm={() => onDeleteRule(row)}
                    >
                      <Button
                        type="link"
                        size="small"
                        danger
                        loading={actingRuleId === row.id}
                        disabled={actingRuleId != null}
                      >
                        删除
                      </Button>
                    </Popconfirm>
                  </Space>
                ) : null,
            },
          ]}
        />
      </Card>
      <Card className="table-card" size="small" title="DaemonSet 状态">
        <Table
          size="small"
          rowKey="id"
          dataSource={agents}
          pagination={false}
          columns={[
            { title: "集群ID", dataIndex: "cluster_id", width: 90 },
            { title: "命名空间", dataIndex: "namespace", width: 140 },
            { title: "状态", dataIndex: "status", width: 110, render: (v: string) => <Tag>{v}</Tag> },
            {
              title: "副本",
              width: 100,
              render: (_: unknown, r?: ClusterLogAgent) => `${r?.ready_replicas ?? 0}/${r?.desired_replicas ?? 0}`,
            },
            { title: "限流 QPS", dataIndex: "rate_limit_qps", width: 100, render: (v?: number) => v ?? "—" },
            { title: "错误", dataIndex: "last_error", ellipsis: true, render: (v: string) => v || "—" },
            {
              title: "操作",
              width: 90,
              fixed: "right",
              render: (_: unknown, r?: ClusterLogAgent) =>
                r ? (
                  <Button
                    type="link"
                    size="small"
                    loading={refreshingAgentId === r.cluster_id}
                    disabled={refreshingAgentId != null}
                    onClick={() => void onRefreshAgent(r)}
                  >
                    刷新
                  </Button>
                ) : null,
            },
          ]}
        />
      </Card>
      <Modal
        title={editing ? "编辑集群采集规则" : "新建集群采集规则"}
        open={open}
        onCancel={() => {
          if (ruleSubmitting) return;
          setOpen(false);
          setEditing(null);
        }}
        onOk={() => void onSubmitRule()}
        confirmLoading={ruleSubmitting}
        okButtonProps={{ disabled: ruleSubmitting }}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="match_namespaces" label="Namespaces（逗号分隔，空=宽采并排除系统 ns）">
            <Input placeholder="default, prod" />
          </Form.Item>
          <Form.Item name="match_workloads" label="Workload 名前缀（逗号分隔，可选）">
            <Input placeholder="my-app" />
          </Form.Item>
          <Form.Item
            name="exclude_namespaces"
            label="排除 Namespaces"
            extra="空=默认排除 kube-system / kube-public / kube-node-lease / yunshu-logging（仅新建时回填默认）"
          >
            <Input placeholder="kube-system, yunshu-logging" />
          </Form.Item>
          <Form.Item
            name="rate_limit_qps"
            label="固定限流 QPS"
            extra="0 或空=从项目配额中与其他「均分」规则平分剩余；高流量可给核心规则设固定值"
          >
            <InputNumber min={0} max={100000} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="parse_profile" label="解析档">
            <Select options={[{ value: "cri", label: "CRI /var/log/pods" }]} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={
          <Space>
            <span>pipelines.yml</span>
            {pipelineCustom ? <Tag color="orange">自定义</Tag> : <Tag>自动生成</Tag>}
          </Space>
        }
        open={pipelineOpen}
        width={900}
        onCancel={() => {
          if (!pipelineSaving) setPipelineOpen(false);
        }}
        footer={
          <Space wrap>
            <Button disabled={pipelineSaving} onClick={() => setPipelineOpen(false)}>
              关闭
            </Button>
            <Button
              disabled={!pipelineGenerated || pipelineSaving}
              onClick={() => {
                setPipelineText(pipelineGenerated);
                message.info("已填入当前规则生成的内容，保存后才会覆盖自定义");
              }}
            >
              填入自动生成
            </Button>
            <Button loading={pipelineSaving} onClick={() => void onSavePipelines(false, true)}>
              恢复自动生成
            </Button>
            <Button loading={pipelineSaving} type="default" onClick={() => void onSavePipelines(false)}>
              仅保存
            </Button>
            <Button loading={pipelineSaving} type="primary" onClick={() => void onSavePipelines(true)}>
              保存并下发
            </Button>
          </Space>
        }
      >
        <Input.TextArea
          value={pipelineText}
          onChange={(e) => setPipelineText(e.target.value)}
          autoSize={{ minRows: 18, maxRows: 28 }}
          style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace", fontSize: 12 }}
        />
        <div style={{ marginTop: 8, color: "rgba(0,0,0,0.45)", fontSize: 12 }}>
          自定义模式下规则变更不会自动覆盖本文件；索引/Topic 前缀见数据字典
          elasticsearch_k8s_index_prefix、kafka_k8s_topic_prefix。
        </div>
      </Modal>
    </Space>
  );

  if (embedded) return body;
  return <Card className="table-card">{body}</Card>;
}

export default ProjectClusterLogPage;
