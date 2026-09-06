import { PlusOutlined, ReloadOutlined, SyncOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  type AILLMModelItem,
  createAICenterCase,
  createAICenterKB,
  createAICenterKBDocument,
  createAICenterModel,
  createAICenterPrompt,
  createAICenterSOP,
  createAICenterTool,
  createAIEvalCase,
  deleteAICenterCase,
  deleteAICenterKB,
  deleteAICenterKBDocument,
  deleteAICenterModel,
  deleteAICenterPrompt,
  deleteAICenterSOP,
  deleteAICenterTool,
  deleteAIEvalCase,
  getAICenterOverview,
  getAICenterPrompt,
  getAIToolRuntimeHealth,
  listAICenterCases,
  listAICenterKBDocuments,
  listAICenterKBs,
  listAICenterModels,
  listAICenterPromptVersions,
  listAICenterPrompts,
  listAICenterSOPs,
  listAICenterTools,
  listAIEvalCases,
  publishAICenterPrompt,
  reseedAICenter,
  rollbackAICenterPrompt,
  runAIEval,
  setDefaultAICenterModel,
  syncAIKnowledge,
  updateAICenterCase,
  updateAICenterKB,
  updateAICenterKBDocument,
  updateAICenterModel,
  updateAICenterPrompt,
  updateAICenterSOP,
  updateAICenterTool,
  updateAICenterToolFull,
  updateAIEvalCase,
} from "../services/ai-center";
import { embedAIKnowledge } from "../services/ai";
import { extractApiErrorMessage } from "../services/http";

const PROVIDER_OPTIONS = [
  { value: "openai_compat", label: "OpenAI 兼容" },
  { value: "deepseek", label: "DeepSeek" },
  { value: "anthropic", label: "Anthropic" },
  { value: "qwen", label: "通义千问 (兼容)" },
];

type Row = Record<string, unknown>;

export function AiCenterPage() {
  const [overview, setOverview] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(false);
  const [prompts, setPrompts] = useState<Row[]>([]);
  const [versions, setVersions] = useState<Row[]>([]);
  const [selectedPromptId, setSelectedPromptId] = useState<number>();
  const [tools, setTools] = useState<Row[]>([]);
  const [cases, setCases] = useState<Row[]>([]);
  const [sops, setSops] = useState<Row[]>([]);
  const [kbs, setKbs] = useState<Row[]>([]);
  const [kbDocs, setKbDocs] = useState<Row[]>([]);
  const [selectedKbId, setSelectedKbId] = useState<number>();
  const [models, setModels] = useState<AILLMModelItem[]>([]);
  const [evalCases, setEvalCases] = useState<Row[]>([]);
  const [evalResult, setEvalResult] = useState<Record<string, unknown> | null>(null);
  const [runtimeHealth, setRuntimeHealth] = useState<Record<string, unknown> | null>(null);

  const [modelModalOpen, setModelModalOpen] = useState(false);
  const [editingModel, setEditingModel] = useState<AILLMModelItem | null>(null);
  const [modelForm] = Form.useForm();

  const [promptModalOpen, setPromptModalOpen] = useState(false);
  const [editingPrompt, setEditingPrompt] = useState<Row | null>(null);
  const [promptForm] = Form.useForm();
  const [publishOpen, setPublishOpen] = useState(false);
  const [publishForm] = Form.useForm();

  const [sopModalOpen, setSopModalOpen] = useState(false);
  const [editingSop, setEditingSop] = useState<Row | null>(null);
  const [sopForm] = Form.useForm();

  const [caseModalOpen, setCaseModalOpen] = useState(false);
  const [editingCase, setEditingCase] = useState<Row | null>(null);
  const [caseForm] = Form.useForm();

  const [kbModalOpen, setKbModalOpen] = useState(false);
  const [editingKb, setEditingKb] = useState<Row | null>(null);
  const [kbForm] = Form.useForm();
  const [docModalOpen, setDocModalOpen] = useState(false);
  const [editingDoc, setEditingDoc] = useState<Row | null>(null);
  const [docForm] = Form.useForm();

  const [toolModalOpen, setToolModalOpen] = useState(false);
  const [editingTool, setEditingTool] = useState<Row | null>(null);
  const [toolForm] = Form.useForm();

  const [evalModalOpen, setEvalModalOpen] = useState(false);
  const [editingEval, setEditingEval] = useState<Row | null>(null);
  const [evalForm] = Form.useForm();

  async function refreshOverview() {
    try {
      setOverview((await getAICenterOverview()) || {});
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载概览失败"));
    }
    try {
      setRuntimeHealth((await getAIToolRuntimeHealth()) || null);
    } catch {
      /* ignore */
    }
  }

  async function refreshModels() {
    try {
      const r = await listAICenterModels();
      setModels(r?.list || []);
    } catch {
      /* ignore */
    }
  }

  async function refreshLists() {
    const tasks = [
      listAICenterPrompts().then((r) => setPrompts(r?.list || [])),
      listAICenterTools().then((r) => setTools(r?.list || [])),
      listAICenterCases().then((r) => setCases(r?.list || [])),
      listAICenterSOPs().then((r) => setSops(r?.list || [])),
      listAICenterKBs().then((r) => setKbs(r?.list || [])),
      listAIEvalCases().then((r) => setEvalCases(r?.list || [])),
      refreshModels(),
    ];
    await Promise.all(tasks.map((p) => p.catch(() => undefined)));
  }

  useEffect(() => {
    void refreshOverview();
    void refreshLists();
  }, []);

  async function loadVersions(id: number) {
    setSelectedPromptId(id);
    try {
      const r = await listAICenterPromptVersions(id);
      setVersions(r?.list || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载版本失败"));
    }
  }

  async function loadKbDocs(kbId: number) {
    setSelectedKbId(kbId);
    try {
      const r = await listAICenterKBDocuments(kbId);
      setKbDocs(r?.list || []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载文档失败"));
    }
  }

  async function handleReseed() {
    setLoading(true);
    try {
      const r = await reseedAICenter();
      const rep = r?.report;
      if (!r?.ok || rep?.data_root_ok === false) {
        const warn = rep?.warnings?.join("；") || r?.error || "种子目录不可用";
        message.error(`重载未完整成功：${warn}`);
      } else {
        message.success(
          `已重载：Prompt ${rep?.prompts ?? 0} / KB ${rep?.knowledge_bases ?? 0} / 案例 ${rep?.cases ?? 0} / SOP ${rep?.sops ?? 0} / Eval ${rep?.eval_cases ?? 0}`,
        );
      }
      await refreshOverview();
      await refreshLists();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "重载失败"));
    } finally {
      setLoading(false);
    }
  }

  async function handleSync() {
    setLoading(true);
    try {
      const r = await syncAIKnowledge();
      const n = r?.indexed ?? 0;
      const failed = r?.failed ?? 0;
      await refreshOverview();
      if (n === 0) {
        message.warning("ES 同步 0 条：请先「同步 data/ai 种子」写入 KB/案例/SOP，并确认 ES 已启用");
      } else if (failed > 0) {
        message.warning(`已同步 ES：${n} 条，失败 ${failed} 条`);
      } else {
        message.success(`已同步 ES：${n} 条`);
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "同步失败"));
    } finally {
      setLoading(false);
    }
  }

  async function handleEmbed() {
    setLoading(true);
    try {
      const r = await embedAIKnowledge({ limit: 200 });
      message.success(r?.message || `向量化完成：updated=${r?.updated ?? 0} skipped=${r?.skipped ?? 0}`);
      await refreshOverview();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "向量化失败"));
    } finally {
      setLoading(false);
    }
  }

  async function handleEval(live: boolean) {
    setLoading(true);
    try {
      const r = await runAIEval(live);
      setEvalResult(r || null);
      message.success(live ? "在线评估完成" : "离线评估完成");
    } catch (e) {
      message.error(extractApiErrorMessage(e, "评估失败"));
    } finally {
      setLoading(false);
    }
  }

  function confirmDelete(title: string, onOk: () => Promise<void>) {
    Modal.confirm({
      title,
      onOk: () =>
        onOk().catch((e) => {
          message.error(extractApiErrorMessage(e, "删除失败"));
          return Promise.reject(e);
        }),
    });
  }

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="AI 运维能力中心"
        description="模型、Prompt、工具、知识库与 Evaluation 统一管理；支持在线增删改查，亦可从 data/ai 重载种子。"
        breadcrumbs={[{ title: "AI" }, { title: "能力中心" }]}
        extra={
          <Space wrap>
            <Link to="/ai/assistant">打开助手</Link>
            <Link to="/ai/investigations">调查记录</Link>
            <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void handleReseed()}>
              重载 data/ai 种子
            </Button>
            <Button icon={<SyncOutlined />} loading={loading} onClick={() => void handleSync()}>
              同步知识库到 ES
            </Button>
            <Button loading={loading} onClick={() => void handleEmbed()}>
              向量化 KB Chunks
            </Button>
            <Button loading={loading} onClick={() => void handleEval(false)}>
              离线 Evaluation
            </Button>
            <Button loading={loading} onClick={() => void handleEval(true)}>
              在线 Evaluation（耗 Token）
            </Button>
          </Space>
        }
      />
      <Card className="table-card">
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Space wrap>
            {Object.entries(overview).map(([k, v]) => (
              <Tag key={k}>
                {k}: {String(v)}
              </Tag>
            ))}
            {runtimeHealth ? (
              <Tag color={runtimeHealth.ok ? "success" : "error"}>
                脚本运行时: {runtimeHealth.python_ok ? `Python OK (${String(runtimeHealth.python_version || runtimeHealth.python_bin || "")})` : "Python 缺失"}
              </Tag>
            ) : null}
          </Space>
          {runtimeHealth && Array.isArray(runtimeHealth.suggestions) && (runtimeHealth.suggestions as string[]).length > 0 ? (
            <Typography.Paragraph type="secondary">
              {(runtimeHealth.suggestions as string[]).join("；")}
            </Typography.Paragraph>
          ) : null}
          {evalResult ? (
            <Typography.Paragraph>
              最近评估：{String(evalResult.summary || "")}（status={String(evalResult.status)}）
            </Typography.Paragraph>
          ) : null}

          <Tabs
            items={[
              {
                key: "prompts",
                label: "Prompt",
                children: (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    <Space style={{ width: "100%", justifyContent: "flex-end" }}>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingPrompt(null);
                          promptForm.resetFields();
                          promptForm.setFieldsValue({ type: "system", enabled: true });
                          setPromptModalOpen(true);
                        }}
                      >
                        新建 Prompt
                      </Button>
                    </Space>
                    <Table
                      rowKey="id"
                      size="small"
                      dataSource={prompts}
                      pagination={{ pageSize: 20, showSizeChanger: true }}
                      columns={[
                        { title: "ID", dataIndex: "id", width: 60 },
                        { title: "Code", dataIndex: "code" },
                        { title: "名称", dataIndex: "name" },
                        { title: "Type", dataIndex: "type", width: 100 },
                        { title: "启用", dataIndex: "enabled", width: 80, render: (v) => (v ? "是" : "否") },
                        {
                          title: "操作",
                          width: 280,
                          render: (_, row) => (
                            <Space size="small" wrap>
                              <Button type="link" size="small" onClick={() => void loadVersions(Number(row.id))}>
                                版本
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  void getAICenterPrompt(Number(row.id)).then((detail) => {
                                    setEditingPrompt(detail || row);
                                    const cur = (detail?.current_version || {}) as Row;
                                    promptForm.setFieldsValue({
                                      ...detail,
                                      content: cur.content || "",
                                    });
                                    setPromptModalOpen(true);
                                  });
                                }}
                              >
                                编辑
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  setSelectedPromptId(Number(row.id));
                                  void getAICenterPrompt(Number(row.id)).then((detail) => {
                                    const cur = (detail?.current_version || {}) as Row;
                                    publishForm.setFieldsValue({
                                      content: cur.content || "",
                                      changelog: "",
                                    });
                                    setPublishOpen(true);
                                  });
                                }}
                              >
                                发布新版
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                danger
                                onClick={() =>
                                  confirmDelete(`删除 Prompt「${String(row.code)}」？`, async () => {
                                    await deleteAICenterPrompt(Number(row.id));
                                    message.success("已删除");
                                    if (selectedPromptId === Number(row.id)) {
                                      setSelectedPromptId(undefined);
                                      setVersions([]);
                                    }
                                    await refreshLists();
                                    await refreshOverview();
                                  })
                                }
                              >
                                删除
                              </Button>
                            </Space>
                          ),
                        },
                      ]}
                    />
                    {selectedPromptId ? (
                      <Card size="small" title={`Prompt #${selectedPromptId} 版本`}>
                        <Table
                          rowKey="id"
                          size="small"
                          dataSource={versions}
                          pagination={{ pageSize: 20, showSizeChanger: true }}
                          columns={[
                            { title: "Ver", dataIndex: "version", width: 60 },
                            { title: "当前", dataIndex: "is_current", width: 70, render: (v) => (v ? "✓" : "") },
                            { title: "变更", dataIndex: "changelog" },
                            { title: "时间", dataIndex: "created_at" },
                            {
                              title: "操作",
                              width: 100,
                              render: (_, row) =>
                                row.is_current ? null : (
                                  <Button
                                    type="link"
                                    size="small"
                                    onClick={() => {
                                      void rollbackAICenterPrompt(selectedPromptId, Number(row.id))
                                        .then(() => {
                                          message.success("已回滚");
                                          return loadVersions(selectedPromptId);
                                        })
                                        .catch((e) => message.error(extractApiErrorMessage(e, "回滚失败")));
                                    }}
                                  >
                                    回滚
                                  </Button>
                                ),
                            },
                          ]}
                        />
                      </Card>
                    ) : null}
                    <Modal
                      title={editingPrompt ? `编辑 Prompt #${editingPrompt.id}` : "新建 Prompt"}
                      open={promptModalOpen}
                      onCancel={() => setPromptModalOpen(false)}
                      destroyOnClose
                      width={720}
                      onOk={() => {
                        void promptForm.validateFields().then(async (values) => {
                          try {
                            if (editingPrompt) {
                              await updateAICenterPrompt(Number(editingPrompt.id), values);
                              message.success("已更新元数据（内容请用「发布新版」）");
                            } else {
                              await createAICenterPrompt(values);
                              message.success("已创建");
                            }
                            setPromptModalOpen(false);
                            await refreshLists();
                            await refreshOverview();
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "保存失败"));
                          }
                        });
                      }}
                    >
                      <Form form={promptForm} layout="vertical">
                        <Form.Item name="code" label="Code" rules={[{ required: !editingPrompt }]}>
                          <Input disabled={!!editingPrompt} placeholder="system/ops-agent" />
                        </Form.Item>
                        <Form.Item name="name" label="名称" rules={[{ required: true }]}>
                          <Input />
                        </Form.Item>
                        <Space wrap style={{ width: "100%" }}>
                          <Form.Item name="type" label="类型" style={{ minWidth: 160 }}>
                            <Select
                              options={[
                                { value: "system", label: "system" },
                                { value: "diagnosis", label: "diagnosis" },
                                { value: "generation", label: "generation" },
                              ]}
                            />
                          </Form.Item>
                          <Form.Item name="scene" label="场景" style={{ minWidth: 200 }}>
                            <Input />
                          </Form.Item>
                          <Form.Item name="enabled" label="启用" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                        </Space>
                        <Form.Item name="remark" label="备注">
                          <Input />
                        </Form.Item>
                        {!editingPrompt ? (
                          <>
                            <Form.Item name="content" label="首版内容">
                              <Input.TextArea rows={10} />
                            </Form.Item>
                            <Form.Item name="changelog" label="变更说明">
                              <Input placeholder="create" />
                            </Form.Item>
                          </>
                        ) : (
                          <Typography.Text type="secondary">
                            修改正文请使用「发布新版」，此处仅更新元数据。
                          </Typography.Text>
                        )}
                      </Form>
                    </Modal>
                    <Modal
                      title={`发布 Prompt #${selectedPromptId} 新版本`}
                      open={publishOpen}
                      onCancel={() => setPublishOpen(false)}
                      destroyOnClose
                      width={800}
                      onOk={() => {
                        void publishForm.validateFields().then(async (values) => {
                          if (!selectedPromptId) return;
                          try {
                            await publishAICenterPrompt(selectedPromptId, values);
                            message.success("已发布");
                            setPublishOpen(false);
                            await loadVersions(selectedPromptId);
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "发布失败"));
                          }
                        });
                      }}
                    >
                      <Form form={publishForm} layout="vertical">
                        <Form.Item name="content" label="内容" rules={[{ required: true }]}>
                          <Input.TextArea rows={16} />
                        </Form.Item>
                        <Form.Item name="changelog" label="变更说明">
                          <Input />
                        </Form.Item>
                      </Form>
                    </Modal>
                  </Space>
                ),
              },
              {
                key: "tools",
                label: "Tools",
                children: (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    <Space style={{ width: "100%", justifyContent: "flex-end" }}>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingTool(null);
                          toolForm.resetFields();
                          toolForm.setFieldsValue({
                            runtime: "script",
                            permission: "READ_ONLY",
                            risk_level: "LOW",
                            timeout_sec: 30,
                            enabled: true,
                            audit_required: true,
                          });
                          setToolModalOpen(true);
                        }}
                      >
                        新建工具
                      </Button>
                    </Space>
                    <Table
                      rowKey="id"
                      size="small"
                      dataSource={tools}
                      pagination={{ pageSize: 20 }}
                      columns={[
                        { title: "名称", dataIndex: "name" },
                        { title: "模块", dataIndex: "module", width: 90 },
                        { title: "运行时", dataIndex: "runtime", width: 90 },
                        { title: "权限", dataIndex: "permission", width: 100 },
                        { title: "风险", dataIndex: "risk_level", width: 90 },
                        {
                          title: "启用",
                          dataIndex: "enabled",
                          width: 90,
                          render: (v, row) => (
                            <Switch
                              checked={!!v}
                              onChange={(checked) => {
                                void updateAICenterTool(Number(row.id), checked)
                                  .then(() => {
                                    setTools((prev) =>
                                      prev.map((t) => (t.id === row.id ? { ...t, enabled: checked } : t)),
                                    );
                                  })
                                  .catch((e) => message.error(extractApiErrorMessage(e, "更新失败")));
                              }}
                            />
                          ),
                        },
                        {
                          title: "操作",
                          width: 140,
                          render: (_, row) => (
                            <Space size="small">
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  setEditingTool(row);
                                  toolForm.setFieldsValue(row);
                                  setToolModalOpen(true);
                                }}
                              >
                                编辑
                              </Button>
                              {String(row.runtime) !== "builtin" ? (
                                <Button
                                  type="link"
                                  size="small"
                                  danger
                                  onClick={() =>
                                    confirmDelete(`删除工具「${String(row.name)}」？`, async () => {
                                      await deleteAICenterTool(Number(row.id));
                                      message.success("已删除");
                                      await refreshLists();
                                    })
                                  }
                                >
                                  删除
                                </Button>
                              ) : null}
                            </Space>
                          ),
                        },
                      ]}
                    />
                    <Modal
                      title={editingTool ? `编辑工具 #${editingTool.id}` : "新建工具"}
                      open={toolModalOpen}
                      onCancel={() => setToolModalOpen(false)}
                      destroyOnClose
                      width={720}
                      onOk={() => {
                        void toolForm.validateFields().then(async (values) => {
                          try {
                            if (editingTool) {
                              await updateAICenterToolFull(Number(editingTool.id), values);
                              message.success("已更新");
                            } else {
                              await createAICenterTool(values);
                              message.success("已创建");
                            }
                            setToolModalOpen(false);
                            await refreshLists();
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "保存失败"));
                          }
                        });
                      }}
                    >
                      <Form form={toolForm} layout="vertical">
                        <Form.Item name="name" label="名称" rules={[{ required: true }]}>
                          <Input disabled={!!editingTool} />
                        </Form.Item>
                        <Form.Item name="description" label="描述">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Space wrap style={{ width: "100%" }}>
                          <Form.Item name="module" label="模块" style={{ minWidth: 140 }}>
                            <Input />
                          </Form.Item>
                          <Form.Item name="runtime" label="运行时" style={{ minWidth: 140 }}>
                            <Select
                              options={[
                                { value: "builtin", label: "builtin" },
                                { value: "script", label: "script" },
                              ]}
                            />
                          </Form.Item>
                          <Form.Item name="permission" label="权限" style={{ minWidth: 140 }}>
                            <Select
                              options={[
                                { value: "READ_ONLY", label: "READ_ONLY" },
                                { value: "WRITE", label: "WRITE" },
                              ]}
                            />
                          </Form.Item>
                          <Form.Item name="risk_level" label="风险" style={{ minWidth: 140 }}>
                            <Select
                              options={["LOW", "MEDIUM", "HIGH", "CRITICAL"].map((v) => ({ value: v, label: v }))}
                            />
                          </Form.Item>
                          <Form.Item name="timeout_sec" label="超时(秒)" style={{ minWidth: 120 }}>
                            <InputNumber min={1} max={600} style={{ width: "100%" }} />
                          </Form.Item>
                        </Space>
                        <Form.Item name="handler_key" label="Handler Key">
                          <Input />
                        </Form.Item>
                        <Form.Item name="script_lang" label="脚本语言">
                          <Select
                            allowClear
                            options={[
                              { value: "python27", label: "python27" },
                              { value: "go", label: "go" },
                              { value: "shell", label: "shell" },
                            ]}
                          />
                        </Form.Item>
                        <Form.Item name="script_path" label="脚本路径">
                          <Input placeholder="相对 data/ai/tools/..." />
                        </Form.Item>
                        <Form.Item name="input_schema_json" label="Input Schema JSON">
                          <Input.TextArea rows={4} />
                        </Form.Item>
                        <Form.Item name="remark" label="备注">
                          <Input />
                        </Form.Item>
                        <Space>
                          <Form.Item name="require_confirmation" label="需确认" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                          <Form.Item name="audit_required" label="需审计" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                          <Form.Item name="enabled" label="启用" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                        </Space>
                      </Form>
                    </Modal>
                  </Space>
                ),
              },
              {
                key: "cases",
                label: "故障案例",
                children: (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    <Space style={{ width: "100%", justifyContent: "flex-end" }}>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingCase(null);
                          caseForm.resetFields();
                          caseForm.setFieldsValue({ enabled: true, confidence: 0.8, source: "manual" });
                          setCaseModalOpen(true);
                        }}
                      >
                        新建案例
                      </Button>
                    </Space>
                    <Table
                      rowKey="id"
                      size="small"
                      dataSource={cases}
                      columns={[
                        { title: "CaseID", dataIndex: "case_id", width: 180 },
                        { title: "标题", dataIndex: "title" },
                        { title: "技术", dataIndex: "technology", width: 100 },
                        { title: "置信度", dataIndex: "confidence", width: 90 },
                        {
                          title: "操作",
                          width: 140,
                          render: (_, row) => (
                            <Space size="small">
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  setEditingCase(row);
                                  caseForm.setFieldsValue(row);
                                  setCaseModalOpen(true);
                                }}
                              >
                                编辑
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                danger
                                onClick={() =>
                                  confirmDelete(`删除案例「${String(row.case_id)}」？`, async () => {
                                    await deleteAICenterCase(Number(row.id));
                                    message.success("已删除");
                                    await refreshLists();
                                  })
                                }
                              >
                                删除
                              </Button>
                            </Space>
                          ),
                        },
                      ]}
                    />
                    <Modal
                      title={editingCase ? `编辑案例 #${editingCase.id}` : "新建案例"}
                      open={caseModalOpen}
                      onCancel={() => setCaseModalOpen(false)}
                      destroyOnClose
                      width={800}
                      onOk={() => {
                        void caseForm.validateFields().then(async (values) => {
                          try {
                            if (editingCase) {
                              await updateAICenterCase(Number(editingCase.id), values);
                              message.success("已更新");
                            } else {
                              await createAICenterCase(values);
                              message.success("已创建");
                            }
                            setCaseModalOpen(false);
                            await refreshLists();
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "保存失败"));
                          }
                        });
                      }}
                    >
                      <Form form={caseForm} layout="vertical">
                        <Form.Item name="case_id" label="CaseID">
                          <Input disabled={!!editingCase} placeholder="留空自动生成" />
                        </Form.Item>
                        <Form.Item name="title" label="标题" rules={[{ required: true }]}>
                          <Input />
                        </Form.Item>
                        <Space wrap style={{ width: "100%" }}>
                          <Form.Item name="category" label="分类" style={{ minWidth: 160 }}>
                            <Input />
                          </Form.Item>
                          <Form.Item name="technology" label="技术" style={{ minWidth: 160 }}>
                            <Input />
                          </Form.Item>
                          <Form.Item name="confidence" label="置信度" style={{ minWidth: 140 }}>
                            <InputNumber min={0} max={1} step={0.05} style={{ width: "100%" }} />
                          </Form.Item>
                          <Form.Item name="enabled" label="启用" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                        </Space>
                        <Form.Item name="symptom" label="症状">
                          <Input.TextArea rows={3} />
                        </Form.Item>
                        <Form.Item name="environment" label="环境">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="diagnosis" label="诊断">
                          <Input.TextArea rows={3} />
                        </Form.Item>
                        <Form.Item name="root_cause" label="根因">
                          <Input.TextArea rows={3} />
                        </Form.Item>
                        <Form.Item name="solution" label="方案">
                          <Input.TextArea rows={3} />
                        </Form.Item>
                        <Form.Item name="verification" label="验证">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="risk" label="风险">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="related_tools" label="关联工具 (JSON)">
                          <Input />
                        </Form.Item>
                        <Form.Item name="related_sop" label="关联 SOP">
                          <Input />
                        </Form.Item>
                        <Form.Item name="source" label="来源">
                          <Input />
                        </Form.Item>
                      </Form>
                    </Modal>
                  </Space>
                ),
              },
              {
                key: "sops",
                label: "SOP",
                children: (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    <Space style={{ width: "100%", justifyContent: "flex-end" }}>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingSop(null);
                          sopForm.resetFields();
                          sopForm.setFieldsValue({ enabled: true, approval_needed: false });
                          setSopModalOpen(true);
                        }}
                      >
                        新建 SOP
                      </Button>
                    </Space>
                    <Table
                      rowKey="id"
                      size="small"
                      dataSource={sops}
                      columns={[
                        { title: "Code", dataIndex: "code", width: 180 },
                        { title: "标题", dataIndex: "title" },
                        {
                          title: "需审批",
                          dataIndex: "approval_needed",
                          width: 90,
                          render: (v) => (v ? "是" : "否"),
                        },
                        {
                          title: "操作",
                          width: 140,
                          render: (_, row) => (
                            <Space size="small">
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  setEditingSop(row);
                                  sopForm.setFieldsValue(row);
                                  setSopModalOpen(true);
                                }}
                              >
                                编辑
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                danger
                                onClick={() =>
                                  confirmDelete(`删除 SOP「${String(row.code)}」？`, async () => {
                                    await deleteAICenterSOP(Number(row.id));
                                    message.success("已删除");
                                    await refreshLists();
                                  })
                                }
                              >
                                删除
                              </Button>
                            </Space>
                          ),
                        },
                      ]}
                    />
                    <Modal
                      title={editingSop ? `编辑 SOP #${editingSop.id}` : "新建 SOP"}
                      open={sopModalOpen}
                      onCancel={() => setSopModalOpen(false)}
                      destroyOnClose
                      width={800}
                      onOk={() => {
                        void sopForm.validateFields().then(async (values) => {
                          try {
                            if (editingSop) {
                              await updateAICenterSOP(Number(editingSop.id), values);
                              message.success("已更新");
                            } else {
                              await createAICenterSOP(values);
                              message.success("已创建");
                            }
                            setSopModalOpen(false);
                            await refreshLists();
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "保存失败"));
                          }
                        });
                      }}
                    >
                      <Form form={sopForm} layout="vertical">
                        <Form.Item name="code" label="Code">
                          <Input disabled={!!editingSop} placeholder="留空自动生成" />
                        </Form.Item>
                        <Form.Item name="title" label="标题" rules={[{ required: true }]}>
                          <Input />
                        </Form.Item>
                        <Form.Item name="scenario" label="场景">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="preconditions" label="前置条件">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="input_params" label="输入参数">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="check_steps" label="检查步骤">
                          <Input.TextArea rows={3} />
                        </Form.Item>
                        <Form.Item name="exec_steps" label="执行步骤">
                          <Input.TextArea rows={4} />
                        </Form.Item>
                        <Form.Item name="verify_steps" label="验证步骤">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="exception_handle" label="异常处理">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="rollback" label="回滚">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="risk" label="风险">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Space>
                          <Form.Item name="approval_needed" label="需审批" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                          <Form.Item name="enabled" label="启用" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                        </Space>
                      </Form>
                    </Modal>
                  </Space>
                ),
              },
              {
                key: "kb",
                label: "知识库",
                children: (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    <Space style={{ width: "100%", justifyContent: "flex-end" }}>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingKb(null);
                          kbForm.resetFields();
                          kbForm.setFieldsValue({ category: "ops", enabled: true });
                          setKbModalOpen(true);
                        }}
                      >
                        新建知识库
                      </Button>
                    </Space>
                    <Table
                      rowKey="id"
                      size="small"
                      dataSource={kbs}
                      columns={[
                        { title: "Code", dataIndex: "code" },
                        { title: "名称", dataIndex: "name" },
                        { title: "分类", dataIndex: "category" },
                        {
                          title: "操作",
                          width: 220,
                          render: (_, row) => (
                            <Space size="small">
                              <Button type="link" size="small" onClick={() => void loadKbDocs(Number(row.id))}>
                                文档
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  setEditingKb(row);
                                  kbForm.setFieldsValue(row);
                                  setKbModalOpen(true);
                                }}
                              >
                                编辑
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                danger
                                onClick={() =>
                                  confirmDelete(`删除知识库「${String(row.code)}」及下属文档？`, async () => {
                                    await deleteAICenterKB(Number(row.id));
                                    message.success("已删除");
                                    if (selectedKbId === Number(row.id)) {
                                      setSelectedKbId(undefined);
                                      setKbDocs([]);
                                    }
                                    await refreshLists();
                                  })
                                }
                              >
                                删除
                              </Button>
                            </Space>
                          ),
                        },
                      ]}
                    />
                    {selectedKbId ? (
                      <Card
                        size="small"
                        title={`知识库 #${selectedKbId} 文档`}
                        extra={
                          <Button
                            size="small"
                            type="primary"
                            icon={<PlusOutlined />}
                            onClick={() => {
                              setEditingDoc(null);
                              docForm.resetFields();
                              docForm.setFieldsValue({
                                kb_id: selectedKbId,
                                version: "v1",
                                source: "manual",
                                enabled: true,
                                confidence: 0.8,
                              });
                              setDocModalOpen(true);
                            }}
                          >
                            新建文档
                          </Button>
                        }
                      >
                        <Table
                          rowKey="id"
                          size="small"
                          dataSource={kbDocs}
                          columns={[
                            { title: "标题", dataIndex: "title" },
                            { title: "来源", dataIndex: "source", width: 120 },
                            { title: "版本", dataIndex: "version", width: 80 },
                            {
                              title: "操作",
                              width: 140,
                              render: (_, row) => (
                                <Space size="small">
                                  <Button
                                    type="link"
                                    size="small"
                                    onClick={() => {
                                      setEditingDoc(row);
                                      docForm.setFieldsValue(row);
                                      setDocModalOpen(true);
                                    }}
                                  >
                                    编辑
                                  </Button>
                                  <Button
                                    type="link"
                                    size="small"
                                    danger
                                    onClick={() =>
                                      confirmDelete(`删除文档「${String(row.title)}」？`, async () => {
                                        await deleteAICenterKBDocument(Number(row.id));
                                        message.success("已删除");
                                        if (selectedKbId) await loadKbDocs(selectedKbId);
                                      })
                                    }
                                  >
                                    删除
                                  </Button>
                                </Space>
                              ),
                            },
                          ]}
                        />
                      </Card>
                    ) : null}
                    <Modal
                      title={editingKb ? `编辑知识库 #${editingKb.id}` : "新建知识库"}
                      open={kbModalOpen}
                      onCancel={() => setKbModalOpen(false)}
                      destroyOnClose
                      onOk={() => {
                        void kbForm.validateFields().then(async (values) => {
                          try {
                            if (editingKb) {
                              await updateAICenterKB(Number(editingKb.id), values);
                              message.success("已更新");
                            } else {
                              await createAICenterKB(values);
                              message.success("已创建");
                            }
                            setKbModalOpen(false);
                            await refreshLists();
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "保存失败"));
                          }
                        });
                      }}
                    >
                      <Form form={kbForm} layout="vertical">
                        <Form.Item name="code" label="Code">
                          <Input disabled={!!editingKb} placeholder="留空自动生成" />
                        </Form.Item>
                        <Form.Item name="name" label="名称" rules={[{ required: true }]}>
                          <Input />
                        </Form.Item>
                        <Form.Item name="category" label="分类">
                          <Input />
                        </Form.Item>
                        <Form.Item name="remark" label="备注">
                          <Input />
                        </Form.Item>
                        <Form.Item name="enabled" label="启用" valuePropName="checked">
                          <Switch />
                        </Form.Item>
                      </Form>
                    </Modal>
                    <Modal
                      title={editingDoc ? `编辑文档 #${editingDoc.id}` : "新建文档"}
                      open={docModalOpen}
                      onCancel={() => setDocModalOpen(false)}
                      destroyOnClose
                      width={800}
                      onOk={() => {
                        void docForm.validateFields().then(async (values) => {
                          try {
                            if (editingDoc) {
                              await updateAICenterKBDocument(Number(editingDoc.id), values);
                              message.success("已更新（需重新向量化）");
                            } else {
                              await createAICenterKBDocument(values);
                              message.success("已创建");
                            }
                            setDocModalOpen(false);
                            if (selectedKbId) await loadKbDocs(selectedKbId);
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "保存失败"));
                          }
                        });
                      }}
                    >
                      <Form form={docForm} layout="vertical">
                        <Form.Item name="kb_id" label="知识库 ID" rules={[{ required: true }]}>
                          <InputNumber style={{ width: "100%" }} disabled />
                        </Form.Item>
                        <Form.Item name="title" label="标题" rules={[{ required: true }]}>
                          <Input />
                        </Form.Item>
                        <Space wrap style={{ width: "100%" }}>
                          <Form.Item name="source" label="来源" style={{ minWidth: 160 }}>
                            <Input />
                          </Form.Item>
                          <Form.Item name="version" label="版本" style={{ minWidth: 120 }}>
                            <Input />
                          </Form.Item>
                          <Form.Item name="confidence" label="置信度" style={{ minWidth: 120 }}>
                            <InputNumber min={0} max={1} step={0.05} style={{ width: "100%" }} />
                          </Form.Item>
                          <Form.Item name="enabled" label="启用" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                        </Space>
                        <Form.Item name="content" label="内容">
                          <Input.TextArea rows={12} />
                        </Form.Item>
                        <Form.Item name="meta_json" label="Meta JSON">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                      </Form>
                    </Modal>
                  </Space>
                ),
              },
              {
                key: "models",
                label: "模型",
                children: (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    <Space wrap style={{ width: "100%", justifyContent: "space-between" }}>
                      <Typography.Text type="secondary">
                        录入后助手页可按「名称」选择；API Key 加密存储，列表不回明文。
                      </Typography.Text>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingModel(null);
                          modelForm.resetFields();
                          modelForm.setFieldsValue({
                            provider: "openai_compat",
                            model_type: "chat",
                            temperature: 0.2,
                            max_tokens: 4096,
                            context_length: 128000,
                            enabled: true,
                            is_default: false,
                          });
                          setModelModalOpen(true);
                        }}
                      >
                        新建模型
                      </Button>
                    </Space>
                    <Table
                      rowKey="id"
                      size="small"
                      dataSource={models}
                      columns={[
                        { title: "名称", dataIndex: "name" },
                        { title: "Provider", dataIndex: "provider", width: 120 },
                        { title: "Model", dataIndex: "model_name" },
                        {
                          title: "Key",
                          dataIndex: "has_api_key",
                          width: 70,
                          render: (v) => (v ? <Tag color="success">已配</Tag> : <Tag>无</Tag>),
                        },
                        {
                          title: "启用",
                          dataIndex: "enabled",
                          width: 70,
                          render: (v) => (v ? "是" : "否"),
                        },
                        {
                          title: "默认",
                          dataIndex: "is_default",
                          width: 70,
                          render: (v) => (v ? <Tag color="blue">默认</Tag> : ""),
                        },
                        {
                          title: "操作",
                          width: 220,
                          render: (_, row) => (
                            <Space size="small">
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  setEditingModel(row);
                                  modelForm.setFieldsValue({
                                    ...row,
                                    api_key: "",
                                  });
                                  setModelModalOpen(true);
                                }}
                              >
                                编辑
                              </Button>
                              {!row.is_default ? (
                                <Button
                                  type="link"
                                  size="small"
                                  onClick={() => {
                                    void setDefaultAICenterModel(row.id)
                                      .then(() => {
                                        message.success("已设为默认");
                                        return refreshModels();
                                      })
                                      .catch((e) => message.error(extractApiErrorMessage(e, "设置失败")));
                                  }}
                                >
                                  设默认
                                </Button>
                              ) : null}
                              <Button
                                type="link"
                                size="small"
                                danger
                                onClick={() =>
                                  confirmDelete(`删除模型「${row.name}」？`, async () => {
                                    await deleteAICenterModel(row.id);
                                    message.success("已删除");
                                    await refreshModels();
                                  })
                                }
                              >
                                删除
                              </Button>
                            </Space>
                          ),
                        },
                      ]}
                    />
                    <Modal
                      title={editingModel ? `编辑模型 #${editingModel.id}` : "新建模型"}
                      open={modelModalOpen}
                      onCancel={() => setModelModalOpen(false)}
                      destroyOnClose
                      width={640}
                      onOk={() => {
                        void modelForm.validateFields().then(async (values) => {
                          try {
                            if (editingModel) {
                              await updateAICenterModel(editingModel.id, values);
                              message.success("已更新");
                            } else {
                              await createAICenterModel(values);
                              message.success("已创建");
                            }
                            setModelModalOpen(false);
                            await refreshModels();
                            await refreshOverview();
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "保存失败"));
                          }
                        });
                      }}
                    >
                      <Form form={modelForm} layout="vertical">
                        <Form.Item name="name" label="名称（助手下拉显示）" rules={[{ required: true }]}>
                          <Input placeholder="如 deepseek-chat-prod" />
                        </Form.Item>
                        <Form.Item name="provider" label="Provider" rules={[{ required: true }]}>
                          <Select options={PROVIDER_OPTIONS} />
                        </Form.Item>
                        <Form.Item name="base_url" label="API Base URL">
                          <Input placeholder="留空则按 Provider 默认" />
                        </Form.Item>
                        <Form.Item
                          name="api_key"
                          label={editingModel?.has_api_key ? "API Key（留空则不修改）" : "API Key"}
                          rules={editingModel ? [] : [{ required: true, message: "请填写 API Key" }]}
                        >
                          <Input.Password placeholder="不会在列表回显" />
                        </Form.Item>
                        <Form.Item name="model_name" label="模型名" rules={[{ required: true }]}>
                          <Input placeholder="如 deepseek-chat / gpt-4o-mini" />
                        </Form.Item>
                        <Space wrap style={{ width: "100%" }}>
                          <Form.Item name="model_type" label="类型" style={{ minWidth: 140 }}>
                            <Select
                              options={[
                                { value: "chat", label: "chat" },
                                { value: "embedding", label: "embedding" },
                              ]}
                            />
                          </Form.Item>
                          <Form.Item name="temperature" label="Temperature" style={{ minWidth: 140 }}>
                            <InputNumber min={0} max={2} step={0.1} style={{ width: "100%" }} />
                          </Form.Item>
                          <Form.Item name="max_tokens" label="Max Tokens" style={{ minWidth: 140 }}>
                            <InputNumber min={256} max={128000} style={{ width: "100%" }} />
                          </Form.Item>
                          <Form.Item name="context_length" label="上下文长度" style={{ minWidth: 140 }}>
                            <InputNumber min={1024} style={{ width: "100%" }} />
                          </Form.Item>
                        </Space>
                        <Form.Item name="model_version" label="版本备注">
                          <Input />
                        </Form.Item>
                        <Form.Item name="remark" label="备注">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Space>
                          <Form.Item name="enabled" label="启用" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                          <Form.Item name="is_default" label="设为默认" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                        </Space>
                      </Form>
                    </Modal>
                  </Space>
                ),
              },
              {
                key: "eval",
                label: "Evaluation",
                children: (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    <Space style={{ width: "100%", justifyContent: "flex-end" }}>
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingEval(null);
                          evalForm.resetFields();
                          evalForm.setFieldsValue({ suite: "default", score_weight: 10, enabled: true });
                          setEvalModalOpen(true);
                        }}
                      >
                        新建用例
                      </Button>
                    </Space>
                    <Table
                      rowKey="id"
                      size="small"
                      dataSource={evalCases}
                      columns={[
                        { title: "Code", dataIndex: "case_code", width: 140 },
                        { title: "标题", dataIndex: "title" },
                        { title: "权重", dataIndex: "score_weight", width: 80 },
                        {
                          title: "操作",
                          width: 140,
                          render: (_, row) => (
                            <Space size="small">
                              <Button
                                type="link"
                                size="small"
                                onClick={() => {
                                  setEditingEval(row);
                                  evalForm.setFieldsValue(row);
                                  setEvalModalOpen(true);
                                }}
                              >
                                编辑
                              </Button>
                              <Button
                                type="link"
                                size="small"
                                danger
                                onClick={() =>
                                  confirmDelete(`删除评估用例「${String(row.case_code)}」？`, async () => {
                                    await deleteAIEvalCase(Number(row.id));
                                    message.success("已删除");
                                    await refreshLists();
                                  })
                                }
                              >
                                删除
                              </Button>
                            </Space>
                          ),
                        },
                      ]}
                    />
                    <Modal
                      title={editingEval ? `编辑评估用例 #${editingEval.id}` : "新建评估用例"}
                      open={evalModalOpen}
                      onCancel={() => setEvalModalOpen(false)}
                      destroyOnClose
                      width={720}
                      onOk={() => {
                        void evalForm.validateFields().then(async (values) => {
                          try {
                            if (editingEval) {
                              await updateAIEvalCase(Number(editingEval.id), values);
                              message.success("已更新");
                            } else {
                              await createAIEvalCase(values);
                              message.success("已创建");
                            }
                            setEvalModalOpen(false);
                            await refreshLists();
                          } catch (e) {
                            message.error(extractApiErrorMessage(e, "保存失败"));
                          }
                        });
                      }}
                    >
                      <Form form={evalForm} layout="vertical">
                        <Form.Item name="case_code" label="Case Code">
                          <Input disabled={!!editingEval} placeholder="留空自动生成" />
                        </Form.Item>
                        <Form.Item name="title" label="标题">
                          <Input />
                        </Form.Item>
                        <Space wrap style={{ width: "100%" }}>
                          <Form.Item name="suite" label="Suite" style={{ minWidth: 160 }}>
                            <Input />
                          </Form.Item>
                          <Form.Item name="score_weight" label="权重" style={{ minWidth: 120 }}>
                            <InputNumber min={1} max={100} style={{ width: "100%" }} />
                          </Form.Item>
                          <Form.Item name="expect_risk" label="期望风险" style={{ minWidth: 140 }}>
                            <Input />
                          </Form.Item>
                          <Form.Item name="enabled" label="启用" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                        </Space>
                        <Form.Item name="input_question" label="输入问题" rules={[{ required: true }]}>
                          <Input.TextArea rows={4} />
                        </Form.Item>
                        <Form.Item name="expect_keywords" label="期望关键词 (JSON)">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="forbid_keywords" label="禁止关键词 (JSON)">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                        <Form.Item name="expect_tools" label="期望工具 (JSON)">
                          <Input.TextArea rows={2} />
                        </Form.Item>
                      </Form>
                    </Modal>
                  </Space>
                ),
              },
            ]}
          />
        </Space>
      </Card>
    </div>
  );
}

export default AiCenterPage;
