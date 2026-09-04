import { CloudSyncOutlined, PlusOutlined, RobotOutlined, SaveOutlined, ThunderboltOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { Link, useLocation, useSearchParams } from "react-router-dom";
import { getClusters, type ClusterItem } from "../services/clusters";
import {
  applyLogPipeline,
  createLogPipeline,
  deleteLogPipeline,
  fetchLoggiePipelinesYAML,
  getLoggieStatus,
  getProjects,
  listLogParseProfiles,
  listLogPipelines,
  syncLogPipelineFromCluster,
  updateLogPipeline,
  type LoggieStatusItem,
  type LogPipelineItem,
  type ProjectItem,
} from "../services/log-platform";
import { adjustLoggiePipeline } from "../services/projects";
import { extractApiErrorMessage } from "../services/http";

type LocationState = {
  project_id?: number;
  sample_logs?: string[];
  open_ai?: boolean;
  goal?: string;
};

/** Loggie Pipeline 仓库：版本化 YAML + 同步/下发 + AI 调整。 */
export function LogPipelineRepoPage() {
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const jumpState = (location.state || {}) as LocationState;

  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [agents, setAgents] = useState<LoggieStatusItem[]>([]);
  const [rows, setRows] = useState<LogPipelineItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [profiles, setProfiles] = useState<Array<{ value: string; label: string }>>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editing, setEditing] = useState<LogPipelineItem | null>(null);
  const [yamlText, setYamlText] = useState("");
  const [saving, setSaving] = useState(false);
  const [aiOpen, setAiOpen] = useState(false);
  const [aiLoading, setAiLoading] = useState(false);
  const [aiSamples, setAiSamples] = useState("");
  const [aiGoal, setAiGoal] = useState("从样例日志抽出 level/service/trace_id 等可观察字段");
  const [syncOpen, setSyncOpen] = useState(false);
  const [syncKind, setSyncKind] = useState<"k8s" | "host">("k8s");
  const [syncTargetId, setSyncTargetId] = useState<number>();
  const [syncing, setSyncing] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 1000 })
      .then((p) => {
        const list = p?.list || [];
        setProjects(list);
        const qPid = Number(searchParams.get("project_id") || jumpState.project_id || 0);
        const defaultProject = qPid || list[0]?.id;
        if (defaultProject) setProjectId(defaultProject);
      })
      .catch(() => undefined);
    void getClusters({ page: 1, page_size: 1000 })
      .then((res) => setClusters(res?.list || []))
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    if (jumpState.sample_logs?.length) {
      setAiSamples(jumpState.sample_logs.join("\n"));
    }
    if (jumpState.goal) setAiGoal(jumpState.goal);
    if (jumpState.open_ai || searchParams.get("ai") === "1") {
      setAiOpen(true);
      if (!editing) {
        setEditing(null);
        setYamlText("pipelines:\n");
        form.setFieldsValue({
          name: `ai-adjust-${Date.now().toString().slice(-6)}`,
          kind: "k8s",
          cluster_id: undefined,
          parse_profile: "spring",
          status: "draft",
          remark: "来自日志查看器样例",
        });
        setEditorOpen(true);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const [listRes, profileRes, statusRes] = await Promise.all([
        listLogPipelines(projectId),
        listLogParseProfiles(projectId).catch(() => ({ list: [] })),
        getLoggieStatus(projectId).catch(() => ({ list: [] as LoggieStatusItem[] })),
      ]);
      setRows(listRes?.list || []);
      setProfiles(profileRes?.list || []);
      setAgents(statusRes?.list || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载 Pipeline 仓库失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [projectId]);

  function openCreate() {
    setEditing(null);
    setYamlText("pipelines:\n");
    form.setFieldsValue({
      name: "",
      kind: "k8s",
      cluster_id: clusters[0]?.id,
      parse_profile: "spring",
      status: "draft",
      remark: "",
    });
    setEditorOpen(true);
  }

  function openEdit(row: LogPipelineItem) {
    setEditing(row);
    setYamlText(row.content_yml || "pipelines:\n");
    form.setFieldsValue({
      name: row.name,
      kind: row.kind,
      cluster_id: row.cluster_id || undefined,
      server_id: row.server_id || undefined,
      parse_profile: row.parse_profile,
      status: row.status,
      remark: row.remark,
    });
    setEditorOpen(true);
  }

  async function saveEditor() {
    if (!projectId) return;
    const values = await form.validateFields();
    setSaving(true);
    try {
      const payload = {
        ...values,
        content_yml: yamlText,
      };
      if (editing) {
        await updateLogPipeline(projectId, editing.id, payload);
        message.success("已更新");
      } else {
        await createLogPipeline(projectId, payload);
        message.success("已创建");
      }
      setEditorOpen(false);
      await load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "保存失败"));
    } finally {
      setSaving(false);
    }
  }

  function openSync(kind: "k8s" | "host") {
    setSyncKind(kind);
    if (kind === "k8s") {
      setSyncTargetId(clusters[0]?.id);
    } else {
      setSyncTargetId(agents[0]?.server_id);
    }
    setSyncOpen(true);
  }

  async function runSync() {
    if (!projectId || !syncTargetId) {
      message.warning(syncKind === "k8s" ? "请选择集群" : "请选择主机");
      return;
    }
    setSyncing(true);
    try {
      if (syncKind === "k8s") {
        await syncLogPipelineFromCluster(projectId, syncTargetId);
        message.success("已从集群同步到仓库");
      } else {
        const yml = await fetchLoggiePipelinesYAML(projectId, syncTargetId);
        const name = `host-server-${syncTargetId}`;
        const existing = rows.find((r) => r.kind === "host" && r.server_id === syncTargetId && r.name === name);
        const payload = {
          name,
          kind: "host" as const,
          server_id: syncTargetId,
          content_yml: yml,
          status: "draft",
          remark: "从主机 Loggie 生成配置同步",
        };
        if (existing) {
          await updateLogPipeline(projectId, existing.id, payload);
        } else {
          await createLogPipeline(projectId, payload);
        }
        message.success("已从主机同步到仓库（草稿）");
      }
      setSyncOpen(false);
      await load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "同步失败"));
    } finally {
      setSyncing(false);
    }
  }

  async function runAiAdjust() {
    setAiLoading(true);
    try {
      const samples = aiSamples
        .split("\n")
        .map((s) => s.trim())
        .filter(Boolean);
      const res = await adjustLoggiePipeline({
        project_id: projectId,
        kind: form.getFieldValue("kind") || "k8s",
        goal: aiGoal,
        sample_logs: samples,
        current_yml: yamlText,
        parse_profile: form.getFieldValue("parse_profile"),
      });
      if (res?.suggested_yml) {
        setYamlText(res.suggested_yml);
        if (!editorOpen) {
          setEditing(null);
          form.setFieldsValue({
            name: form.getFieldValue("name") || `ai-adjust-${Date.now().toString().slice(-6)}`,
            kind: form.getFieldValue("kind") || "k8s",
            status: "draft",
            parse_profile: res.parse_profile || form.getFieldValue("parse_profile"),
            remark: res.summary || "AI 调整",
          });
          setEditorOpen(true);
        }
      }
      if (res?.parse_profile) {
        form.setFieldsValue({ parse_profile: res.parse_profile });
      }
      message.success(res?.summary || "AI 调整完成");
      if (res?.extracted_fields?.length) {
        message.info(`建议字段：${res.extracted_fields.slice(0, 8).join(", ")}`);
      }
      setAiOpen(false);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "AI 调整失败"));
    } finally {
      setAiLoading(false);
    }
  }

  return (
    <div className="log-pipeline-repo-page">
      <Card
        title="Pipeline 仓库"
        extra={
          <Space wrap>
            <Select
              style={{ width: 220 }}
              value={projectId}
              options={projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` }))}
              onChange={setProjectId}
            />
            <Button icon={<CloudSyncOutlined />} onClick={() => openSync("k8s")}>
              从集群同步
            </Button>
            <Button icon={<CloudSyncOutlined />} onClick={() => openSync("host")}>
              从主机同步
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建
            </Button>
            <Link to={projectId ? `/project-logs?project_id=${projectId}` : "/project-logs"}>返回日志查看器</Link>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginTop: 0 }}>
          管理 Loggie <Typography.Text code>pipelines.yml</Typography.Text>
          ：支持随时编辑、根据样例日志 AI 调整解析字段，并下发到 K8s DaemonSet。主机类型保存后请到「Agent 管理」执行部署/同步。
        </Typography.Paragraph>
        <Table
          rowKey="id"
          loading={loading}
          dataSource={rows}
          size="small"
          columns={[
            { title: "名称", dataIndex: "name", width: 180 },
            {
              title: "类型",
              dataIndex: "kind",
              width: 90,
              render: (v: string) => <Tag>{v}</Tag>,
            },
            {
              title: "状态",
              dataIndex: "status",
              width: 100,
              render: (v: string) => <Tag color={v === "published" ? "success" : "default"}>{v}</Tag>,
            },
            { title: "版本", dataIndex: "version", width: 70 },
            { title: "解析档", dataIndex: "parse_profile", width: 120, ellipsis: true },
            {
              title: "作用域",
              width: 140,
              render: (_: unknown, r: LogPipelineItem) =>
                r.kind === "k8s" ? `cluster:${r.cluster_id || "-"}` : `server:${r.server_id || "-"}`,
            },
            { title: "备注", dataIndex: "remark", ellipsis: true },
            {
              title: "操作",
              width: 300,
              render: (_: unknown, r: LogPipelineItem) => (
                <Space wrap size={4}>
                  <Button type="link" size="small" onClick={() => openEdit(r)}>
                    编辑
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    icon={<RobotOutlined />}
                    onClick={() => {
                      openEdit(r);
                      setAiOpen(true);
                    }}
                  >
                    AI
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    icon={<ThunderboltOutlined />}
                    disabled={r.kind !== "k8s"}
                    onClick={() =>
                      void (async () => {
                        if (!projectId) return;
                        try {
                          await applyLogPipeline(projectId, r.id, { apply_deploy: true });
                          message.success("已下发到集群");
                          await load();
                        } catch (e) {
                          message.error(extractApiErrorMessage(e, "下发失败"));
                        }
                      })()
                    }
                  >
                    下发
                  </Button>
                  <Popconfirm
                    title="确认删除？"
                    onConfirm={() =>
                      void (async () => {
                        if (!projectId) return;
                        await deleteLogPipeline(projectId, r.id);
                        message.success("已删除");
                        await load();
                      })()
                    }
                  >
                    <Button type="link" size="small" danger>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Modal
        title={editing ? `编辑 Pipeline #${editing.id}` : "新建 Pipeline"}
        open={editorOpen}
        onCancel={() => setEditorOpen(false)}
        width={960}
        destroyOnClose
        footer={
          <Space>
            <Button icon={<RobotOutlined />} onClick={() => setAiOpen(true)}>
              AI 调整
            </Button>
            <Button onClick={() => setEditorOpen(false)}>取消</Button>
            <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => void saveEditor()}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical" size="small">
          <Space wrap style={{ width: "100%" }} align="start">
            <Form.Item name="name" label="名称" rules={[{ required: true }]} style={{ width: 220 }}>
              <Input />
            </Form.Item>
            <Form.Item name="kind" label="类型" style={{ width: 120 }}>
              <Select
                options={[
                  { value: "k8s", label: "K8s" },
                  { value: "host", label: "主机" },
                  { value: "template", label: "模板" },
                ]}
              />
            </Form.Item>
            <Form.Item name="cluster_id" label="集群" style={{ width: 200 }}>
              <Select allowClear options={clusters.map((c) => ({ value: c.id, label: c.name }))} />
            </Form.Item>
            <Form.Item name="server_id" label="主机" style={{ width: 200 }}>
              <Select
                allowClear
                options={agents
                  .filter((a) => a.server_id)
                  .map((a) => ({
                    value: a.server_id,
                    label: `${a.server_name || "server"} (${a.server_host || a.server_id})`,
                  }))}
              />
            </Form.Item>
            <Form.Item name="parse_profile" label="解析档" style={{ width: 220 }}>
              <Select allowClear options={profiles} />
            </Form.Item>
            <Form.Item name="status" label="状态" style={{ width: 120 }}>
              <Select
                options={[
                  { value: "draft", label: "draft" },
                  { value: "published", label: "published" },
                ]}
              />
            </Form.Item>
          </Space>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
          <Form.Item label="pipelines.yml">
            <Input.TextArea
              value={yamlText}
              onChange={(e) => setYamlText(e.target.value)}
              autoSize={{ minRows: 16, maxRows: 28 }}
              style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace", fontSize: 12 }}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="AI 调整 Pipeline"
        open={aiOpen}
        onCancel={() => setAiOpen(false)}
        onOk={() => void runAiAdjust()}
        confirmLoading={aiLoading}
        okText="生成建议并填入"
        width={720}
        destroyOnClose
      >
        <Form layout="vertical" size="small">
          <Form.Item label="调整目标">
            <Input.TextArea value={aiGoal} onChange={(e) => setAiGoal(e.target.value)} rows={2} />
          </Form.Item>
          <Form.Item label="样例日志（每行一条，建议 3～10 条）">
            <Input.TextArea
              value={aiSamples}
              onChange={(e) => setAiSamples(e.target.value)}
              autoSize={{ minRows: 8, maxRows: 16 }}
              placeholder="粘贴未解析或解析不佳的原始日志行"
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={syncKind === "k8s" ? "从 K8s 集群同步" : "从主机 Agent 同步"}
        open={syncOpen}
        onCancel={() => setSyncOpen(false)}
        onOk={() => void runSync()}
        confirmLoading={syncing}
        okText="同步到仓库"
      >
        {syncKind === "k8s" ? (
          <Select
            style={{ width: "100%" }}
            placeholder="选择集群"
            value={syncTargetId}
            onChange={setSyncTargetId}
            options={clusters.map((c) => ({ value: c.id, label: c.name }))}
          />
        ) : (
          <Select
            style={{ width: "100%" }}
            placeholder="选择主机"
            value={syncTargetId}
            onChange={setSyncTargetId}
            options={agents
              .filter((a) => a.server_id)
              .map((a) => ({
                value: a.server_id,
                label: `${a.server_name || "server"} (${a.server_host || a.server_id})`,
              }))}
          />
        )}
      </Modal>
    </div>
  );
}
