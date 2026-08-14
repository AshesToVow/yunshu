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
import {
  type AILLMModelItem,
  createAICenterModel,
  deleteAICenterModel,
  getAICenterOverview,
  listAICenterCases,
  listAICenterKBs,
  listAICenterModels,
  listAICenterPromptVersions,
  listAICenterPrompts,
  listAICenterSOPs,
  listAICenterTools,
  listAIEvalCases,
  reseedAICenter,
  runAIEval,
  setDefaultAICenterModel,
  syncAIKnowledge,
  updateAICenterModel,
  updateAICenterTool,
} from "../services/ai-center";
import { extractApiErrorMessage } from "../services/http";

const PROVIDER_OPTIONS = [
  { value: "openai_compat", label: "OpenAI 兼容" },
  { value: "deepseek", label: "DeepSeek" },
  { value: "anthropic", label: "Anthropic" },
  { value: "qwen", label: "通义千问 (兼容)" },
];

export function AiCenterPage() {
  const [overview, setOverview] = useState<Record<string, unknown>>({});
  const [loading, setLoading] = useState(false);
  const [prompts, setPrompts] = useState<Array<Record<string, unknown>>>([]);
  const [versions, setVersions] = useState<Array<Record<string, unknown>>>([]);
  const [selectedPromptId, setSelectedPromptId] = useState<number>();
  const [tools, setTools] = useState<Array<Record<string, unknown>>>([]);
  const [cases, setCases] = useState<Array<Record<string, unknown>>>([]);
  const [sops, setSops] = useState<Array<Record<string, unknown>>>([]);
  const [kbs, setKbs] = useState<Array<Record<string, unknown>>>([]);
  const [models, setModels] = useState<AILLMModelItem[]>([]);
  const [evalCases, setEvalCases] = useState<Array<Record<string, unknown>>>([]);
  const [evalResult, setEvalResult] = useState<Record<string, unknown> | null>(null);
  const [modelModalOpen, setModelModalOpen] = useState(false);
  const [editingModel, setEditingModel] = useState<AILLMModelItem | null>(null);
  const [modelForm] = Form.useForm();

  async function refreshOverview() {
    try {
      setOverview((await getAICenterOverview()) || {});
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载概览失败"));
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

  useEffect(() => {
    void refreshOverview();
    void listAICenterPrompts()
      .then((r) => setPrompts(r?.list || []))
      .catch(() => undefined);
    void listAICenterTools()
      .then((r) => setTools(r?.list || []))
      .catch(() => undefined);
    void listAICenterCases()
      .then((r) => setCases(r?.list || []))
      .catch(() => undefined);
    void listAICenterSOPs()
      .then((r) => setSops(r?.list || []))
      .catch(() => undefined);
    void listAICenterKBs()
      .then((r) => setKbs(r?.list || []))
      .catch(() => undefined);
    void listAICenterModels()
      .then((r) => setModels(r?.list || []))
      .catch(() => undefined);
    void listAIEvalCases()
      .then((r) => setEvalCases(r?.list || []))
      .catch(() => undefined);
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
      // 刷新各列表
      void listAICenterPrompts()
        .then((x) => setPrompts(x?.list || []))
        .catch(() => undefined);
      void listAICenterTools()
        .then((x) => setTools(x?.list || []))
        .catch(() => undefined);
      void listAICenterCases()
        .then((x) => setCases(x?.list || []))
        .catch(() => undefined);
      void listAICenterSOPs()
        .then((x) => setSops(x?.list || []))
        .catch(() => undefined);
      void listAICenterKBs()
        .then((x) => setKbs(x?.list || []))
        .catch(() => undefined);
      void listAIEvalCases()
        .then((x) => setEvalCases(x?.list || []))
        .catch(() => undefined);
      void refreshModels();
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
      if (n === 0) {
        message.warning("ES 同步 0 条：请先「同步 data/ai 种子」写入 KB/案例/SOP，并确认 ES 已启用");
      } else {
        message.success(`已同步 ES：${n} 条`);
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "同步失败"));
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

  return (
    <div>
      <Card className="table-card" title="AI 运维能力中心">
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Space wrap>
            <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void handleReseed()}>
              重载 data/ai 种子
            </Button>
            <Button icon={<SyncOutlined />} loading={loading} onClick={() => void handleSync()}>
              同步知识库到 ES
            </Button>
            <Button loading={loading} onClick={() => void handleEval(false)}>
              离线 Evaluation
            </Button>
            <Button loading={loading} onClick={() => void handleEval(true)}>
              在线 Evaluation（耗 Token）
            </Button>
          </Space>
          <Space wrap>
            {Object.entries(overview).map(([k, v]) => (
              <Tag key={k}>
                {k}: {String(v)}
              </Tag>
            ))}
          </Space>
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
                    <Table
                      rowKey="id"
                      size="small"
                      dataSource={prompts}
                      pagination={false}
                      columns={[
                        { title: "ID", dataIndex: "id", width: 60 },
                        { title: "Code", dataIndex: "code" },
                        { title: "Type", dataIndex: "type", width: 100 },
                        { title: "启用", dataIndex: "enabled", width: 80, render: (v) => (v ? "是" : "否") },
                        {
                          title: "操作",
                          width: 120,
                          render: (_, row) => (
                            <Button type="link" onClick={() => void loadVersions(Number(row.id))}>
                              版本
                            </Button>
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
                          pagination={false}
                          columns={[
                            { title: "Ver", dataIndex: "version", width: 60 },
                            { title: "当前", dataIndex: "is_current", width: 70, render: (v) => (v ? "✓" : "") },
                            { title: "变更", dataIndex: "changelog" },
                            { title: "时间", dataIndex: "created_at" },
                          ]}
                        />
                      </Card>
                    ) : null}
                  </Space>
                ),
              },
              {
                key: "tools",
                label: "Tools",
                children: (
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
                    ]}
                  />
                ),
              },
              {
                key: "cases",
                label: "故障案例",
                children: (
                  <Table
                    rowKey="id"
                    size="small"
                    dataSource={cases}
                    columns={[
                      { title: "CaseID", dataIndex: "case_id", width: 180 },
                      { title: "标题", dataIndex: "title" },
                      { title: "技术", dataIndex: "technology", width: 100 },
                      { title: "置信度", dataIndex: "confidence", width: 90 },
                    ]}
                  />
                ),
              },
              {
                key: "sops",
                label: "SOP",
                children: (
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
                    ]}
                  />
                ),
              },
              {
                key: "kb",
                label: "知识库",
                children: (
                  <Table
                    rowKey="id"
                    size="small"
                    dataSource={kbs}
                    columns={[
                      { title: "Code", dataIndex: "code" },
                      { title: "名称", dataIndex: "name" },
                      { title: "分类", dataIndex: "category" },
                    ]}
                  />
                ),
              },
              {
                key: "models",
                label: "模型",
                children: (
                  <Space direction="vertical" style={{ width: "100%" }}>
                    <Space wrap style={{ width: "100%", justifyContent: "space-between" }}>
                      <Typography.Text type="secondary">
                        录入后助手页可按「名称」选择；API Key 加密存储，列表不回明文。字典 ai_* 仍可作兜底。
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
                                onClick={() => {
                                  Modal.confirm({
                                    title: `删除模型「${row.name}」？`,
                                    onOk: () =>
                                      deleteAICenterModel(row.id)
                                        .then(() => {
                                          message.success("已删除");
                                          return refreshModels();
                                        })
                                        .catch((e) => message.error(extractApiErrorMessage(e, "删除失败"))),
                                  });
                                }}
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
                  <Table
                    rowKey="id"
                    size="small"
                    dataSource={evalCases}
                    columns={[
                      { title: "Code", dataIndex: "case_code", width: 140 },
                      { title: "标题", dataIndex: "title" },
                      { title: "权重", dataIndex: "score_weight", width: 80 },
                    ]}
                  />
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
