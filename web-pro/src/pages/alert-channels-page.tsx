// @ts-nocheck
import { DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from "antd";
import { useEffect, useRef, useState } from "react";
import { Link } from '@umijs/max';
import { AlertNotificationTemplateStudio } from "../components/alert/alert-notification-template-studio";
import {
  DEFAULT_FIRING_TEMPLATE,
  DEFAULT_RESOLVED_TEMPLATE,
  type TemplatePreviewStatus,
} from "../components/alert/notification-template-presets";
import {
  createAlertChannel,
  deleteAlertChannel,
  listAlertChannels,
  testAlertChannel,
  updateAlertChannel,
  type AlertChannelItem,
  type AlertChannelTestResult,
} from "../services/alerts";
import { useDictOptions } from "../hooks/use-dict-options";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { formatDateTime } from "../utils/format";
import { DictFillSelect } from "../components/dict-fill-select";
import { getProjects, type ProjectItem } from "../services/projects";

export function AlertChannelsPage() {
  const [list, setList] = useState<AlertChannelItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  const [testSending, setTestSending] = useState(false);
  const [testReceipt, setTestReceipt] = useState<AlertChannelTestResult | null>(null);
  const [testRow, setTestRow] = useState<AlertChannelItem | null>(null);
  const [templateOpen, setTemplateOpen] = useState(false);
  const [templateSaving, setTemplateSaving] = useState(false);
  const [templateChannelID, setTemplateChannelID] = useState<number | undefined>();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [editingTemplateTarget, setEditingTemplateTarget] = useState<TemplatePreviewStatus>("firing");
  const [firingPreset, setFiringPreset] = useState<string>("default");
  const [resolvedPreset, setResolvedPreset] = useState<string>("default");
  const [current, setCurrent] = useState<AlertChannelItem | null>(null);
  const [form] = Form.useForm();
  const [testForm] = Form.useForm();
  const [templateForm] = Form.useForm();
  const firingTemplateRef = useRef<any>(null);
  const resolvedTemplateRef = useRef<any>(null);
  const channelType = Form.useWatch("type", form);
  const wecomMode = Form.useWatch("wecom_mode", form);
  const dingMode = Form.useWatch("ding_mode", form);
  const webhookURLDictOptions = useDictOptions("alert_webhook_url");
  const wecomWebhookURLDictOptions = useDictOptions("wecom_webhook_url");
  const dingtalkWebhookURLDictOptions = useDictOptions("dingtalk_webhook_url");
  const channelTypeOptions = useDictOptions("alert_channel_type");
  const wecomModeOptions = useDictOptions("wecom_notify_mode");
  const dingModeOptions = useDictOptions("dingtalk_notify_mode");
  const wecomCorpIDOptions = useDictOptions("wecom_corp_id");
  const wecomCorpSecretOptions = useDictOptions("wecom_corp_secret");
  const wecomAgentIDOptions = useDictOptions("wecom_agent_id");
  const dingAppKeyOptions = useDictOptions("dingtalk_app_key");
  const dingAppSecretOptions = useDictOptions("dingtalk_app_secret");
  const dingChatIDOptions = useDictOptions("dingtalk_chat_id");
  const dingSignSecretOptions = useDictOptions("dingtalk_sign_secret");
  const urlFillOptions =
    channelType === "wechat_work" || channelType === "wechat"
      ? wecomWebhookURLDictOptions
      : channelType === "dingding"
        ? dingtalkWebhookURLDictOptions
        : webhookURLDictOptions;

  function parseChannelSettings(raw?: string) {
    if (!raw?.trim()) return {};
    try {
      const obj = JSON.parse(raw);
      if (obj && typeof obj === "object" && !Array.isArray(obj)) return obj as Record<string, unknown>;
      return {};
    } catch {
      return {};
    }
  }

  function stringifySettings(v: Record<string, unknown>) {
    const cleaned: Record<string, unknown> = {};
    for (const [k, val] of Object.entries(v)) {
      if (val === undefined || val === null) continue;
      if (typeof val === "string" && !val.trim()) continue;
      if (Array.isArray(val) && val.length === 0) continue;
      cleaned[k] = val;
    }
    return JSON.stringify(cleaned);
  }

  async function load() {
    setLoading(true);
    try {
      const res = await listAlertChannels();
      setList(res.list ?? []);
    } catch {
      // http 拦截器已 toast
    } finally {
      setLoading(false);
    }
  }

  async function loadProjects() {
    try {
      const res = await getProjects({ page: 1, page_size: 500 });
      setProjects(res.list ?? []);
    } catch {
      setProjects([]);
    }
  }

  useEffect(() => {
    void load();
    void loadProjects();
  }, []);

  function openCreate() {
    setCurrent(null);
    form.setFieldsValue({
      name: "",
      type: "generic_webhook",
      url: "",
      secret: "",
      headers_json: "",
      extra_headers_json: "{}",
      at_mobiles: [],
      at_user_ids: [],
      is_at_all: false,
      wecom_mode: "robot",
      corp_id: "",
      corp_secret: "",
      agent_id: "",
      ding_mode: "robot",
      ding_app_key: "",
      ding_app_secret: "",
      ding_chat_id: "",
      ding_sign_secret: "",
      email_to: [],
      enabled: true,
      timeout_ms: 5000,
    });
    setOpen(true);
  }

  function openEdit(row: AlertChannelItem) {
    setCurrent(row);
    const settings = parseChannelSettings(row.headers_json || "");
    const atMobiles = Array.isArray(settings.atMobiles) ? (settings.atMobiles as unknown[]).map((v) => String(v)) : [];
    const atUserIds = Array.isArray(settings.atUserIds) ? (settings.atUserIds as unknown[]).map((v) => String(v)) : [];
    const emailTo =
      Array.isArray(settings.to)
        ? (settings.to as unknown[]).map((v) => String(v))
        : typeof settings.to === "string"
          ? String(settings.to).split(",").map((s) => s.trim()).filter(Boolean)
          : [];
    const extra = { ...settings };
    delete (extra as any).atMobiles;
    delete (extra as any).atUserIds;
    delete (extra as any).isAtAll;
    delete (extra as any).corpID;
    delete (extra as any).corpSecret;
    delete (extra as any).chatId;
    delete (extra as any).signSecret;
    delete (extra as any).to;
    delete (extra as any).messageTemplateFiring;
    delete (extra as any).messageTemplateResolved;
    form.setFieldsValue({
      name: row.name,
      type: row.type || "generic_webhook",
      url: row.url,
      secret: row.secret || "",
      headers_json: row.headers_json || "",
      extra_headers_json: stringifySettings(extra),
      at_mobiles: atMobiles,
      at_user_ids: atUserIds,
      is_at_all: !!settings.isAtAll,
      wecom_mode: typeof settings.wecomMode === "string" ? settings.wecomMode : "robot",
      corp_id: typeof settings.corpID === "string" ? settings.corpID : "",
      corp_secret: typeof settings.corpSecret === "string" ? settings.corpSecret : "",
      agent_id: typeof settings.agentId === "string" ? settings.agentId : "",
      ding_mode: typeof settings.dingMode === "string" ? settings.dingMode : "robot",
      ding_app_key: typeof settings.appKey === "string" ? settings.appKey : "",
      ding_app_secret: typeof settings.appSecret === "string" ? settings.appSecret : "",
      ding_chat_id: typeof settings.chatId === "string" ? settings.chatId : "",
      ding_sign_secret: typeof settings.signSecret === "string" ? settings.signSecret : "",
      email_to: emailTo,
      enabled: row.enabled,
      timeout_ms: row.timeout_ms || 5000,
    });
    setOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const extraSettings = parseChannelSettings(values.extra_headers_json || "{}");
      const settings: Record<string, unknown> = { ...extraSettings };
      const atMobiles = (values.at_mobiles ?? []).map((v: string) => v.trim()).filter(Boolean);
      const atUserIds = (values.at_user_ids ?? []).map((v: string) => v.trim()).filter(Boolean);
      const emailTo = (values.email_to ?? []).map((v: string) => v.trim()).filter(Boolean);
      if (atMobiles.length > 0) settings.atMobiles = atMobiles;
      if (atUserIds.length > 0) settings.atUserIds = atUserIds;
      if (values.is_at_all) settings.isAtAll = true;
      if ((values.wecom_mode || "").trim()) settings.wecomMode = values.wecom_mode.trim();
      if ((values.corp_id || "").trim()) settings.corpID = values.corp_id.trim();
      if ((values.corp_secret || "").trim()) settings.corpSecret = values.corp_secret.trim();
      if ((values.agent_id || "").trim()) settings.agentId = values.agent_id.trim();
      if ((values.ding_mode || "").trim()) settings.dingMode = values.ding_mode.trim();
      if ((values.ding_app_key || "").trim()) settings.appKey = values.ding_app_key.trim();
      if ((values.ding_app_secret || "").trim()) settings.appSecret = values.ding_app_secret.trim();
      if ((values.ding_chat_id || "").trim()) settings.chatId = values.ding_chat_id.trim();
      if ((values.ding_sign_secret || "").trim()) settings.signSecret = values.ding_sign_secret.trim();
      if (values.type === "email" && emailTo.length > 0) settings.to = emailTo;
      const payload = {
        name: values.name,
        type: values.type,
        url: values.url,
        secret: values.secret,
        headers_json: stringifySettings(settings),
        enabled: !!values.enabled,
        timeout_ms: Number(values.timeout_ms || 5000),
      };
      if (current) {
        await updateAlertChannel(current.id, payload);
        message.success("告警通道已更新");
      } else {
        await createAlertChannel(payload);
        message.success("告警通道已创建");
      }
      setOpen(false);
      await load();
    } finally {
      setSaving(false);
    }
  }

  function openTemplateConfig(channelID?: number) {
    const targetID = channelID ?? templateChannelID ?? list[0]?.id;
    if (!targetID) {
      message.warning("请先创建告警通道");
      return;
    }
    const row = list.find((it) => it.id === targetID);
    if (!row) {
      message.warning("未找到目标通道");
      return;
    }
    const settings = parseChannelSettings(row.headers_json || "");
    setTemplateChannelID(targetID);
    templateForm.setFieldsValue({
      template_firing: typeof settings.messageTemplateFiring === "string" ? settings.messageTemplateFiring : DEFAULT_FIRING_TEMPLATE,
      template_resolved: typeof settings.messageTemplateResolved === "string" ? settings.messageTemplateResolved : DEFAULT_RESOLVED_TEMPLATE,
      preview_status: "firing",
      preview_project_id: undefined,
      preview_raw_payload_json: "",
    });
    setFiringPreset("default");
    setResolvedPreset("default");
    setTemplateOpen(true);
  }

  async function submitTemplate() {
    if (!templateChannelID) {
      message.warning("请先选择要保存模板的告警通道");
      return;
    }
    const values = await templateForm.validateFields();
    const row = list.find((it) => it.id === templateChannelID);
    if (!row) {
      message.error("未找到目标通道");
      return;
    }
    const settings = parseChannelSettings(row.headers_json || "");
    const firing = String(values.template_firing || "").trim();
    const resolved = String(values.template_resolved || "").trim();
    if (firing) settings.messageTemplateFiring = firing;
    else delete (settings as any).messageTemplateFiring;
    if (resolved) settings.messageTemplateResolved = resolved;
    else delete (settings as any).messageTemplateResolved;
    setTemplateSaving(true);
    try {
      await updateAlertChannel(templateChannelID, {
        name: row.name,
        type: row.type,
        url: row.url,
        secret: row.secret,
        headers_json: stringifySettings(settings),
        enabled: row.enabled,
        timeout_ms: row.timeout_ms,
      });
      message.success("告警模板已保存");
      setTemplateOpen(false);
      await load();
    } finally {
      setTemplateSaving(false);
    }
  }

  const templateChannelRow = list.find((it) => it.id === templateChannelID);

  function openTest(row: AlertChannelItem) {
    setTestRow(row);
    testForm.setFieldsValue({
      status: "firing",
      severity: "info",
      title: "",
      content: "",
    });
    setTestOpen(true);
  }

  async function submitTest() {
    if (!testRow) return;
    const values = await testForm.validateFields();
    setTestSending(true);
    setTestReceipt(null);
    try {
      const receipt = await testAlertChannel(testRow.id, {
        status: values.status,
        severity: values.severity,
        title: values.title,
        content: values.content,
      });
      setTestReceipt(receipt);
      if (receipt.success) message.success("测试发送成功");
      else message.warning(receipt.error_message || "测试发送未成功");
    } finally {
      setTestSending(false);
    }
  }

  return (
    <div className="page-stack">
      <PageTelemetryHeader
        label="[ ALERT / CHANNEL ]"
        title="Webhook 告警通道"
        subtitle="配置 Webhook 投递端点、超时策略与告警模板绑定"
        meta={[
          `COUNT / ${list.length}`,
          loading ? "SYNC / PENDING" : "SYNC / OK",
        ]}
      />
    <Card className="table-card">
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 12 }}>
        <Space>
          <Select
            style={{ width: 260 }}
            allowClear
            placeholder="选择一个通道配置告警模板"
            value={templateChannelID}
            onChange={(v) => setTemplateChannelID(v)}
            options={list.map((it) => ({ label: `${it.name} (${it.type})`, value: it.id }))}
          />
          <Button type="primary" ghost onClick={() => openTemplateConfig()}>
            通知模板可视化
          </Button>
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建通道</Button>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
        columns={[
          { title: "名称", dataIndex: "name", width: 180 },
          { title: "类型", dataIndex: "type", width: 140 },
          { title: "Webhook URL", dataIndex: "url", ellipsis: true },
          { title: "超时(ms)", dataIndex: "timeout_ms", width: 110 },
          {
            title: "状态",
            dataIndex: "enabled",
            width: 90,
            render: (v: boolean) => (v ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>),
          },
          { title: "更新时间", dataIndex: "updated_at", width: 170, render: (v: string) => formatDateTime(v) },
          {
            title: "操作",
            key: "action",
            width: 220,
            render: (_: unknown, row: AlertChannelItem) => (
              <Space size={4} wrap>
                <Button type="link" icon={<SendOutlined />} onClick={() => openTest(row)}>
                  测试
                </Button>
                <Button type="link" onClick={() => openTemplateConfig(row.id)}>
                  模板
                </Button>
                <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(row)}>
                  编辑
                </Button>
                <Popconfirm
                  title="确认删除该通道吗？"
                  onConfirm={() =>
                    void (async () => {
                      await deleteAlertChannel(row.id);
                      message.success("已删除");
                      await load();
                    })()
                  }
                >
                  <Button type="link" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={current ? "编辑通道" : "新建告警通道"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void submit()}
        confirmLoading={saving}
        destroyOnClose
        width={760}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <Form.Item name="name" label="通道名称" rules={[{ required: true, message: "请输入通道名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="通道类型" rules={[{ required: true }]}>
            <Select options={channelTypeOptions} />
          </Form.Item>
          <Form.Item name="url" label="Webhook URL（email、钉钉app_chat 可留空）" rules={[{ type: "url", message: "URL 格式不正确" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="从字典填充 URL">
            <DictFillSelect
              form={form}
              fieldName="url"
              options={urlFillOptions}
              placeholder={
                channelType === "dingding"
                  ? "可选：选择钉钉 Webhook URL"
                  : channelType === "wechat_work" || channelType === "wechat"
                    ? "可选：选择企业微信 Webhook URL"
                    : "可选：选择后自动填入 Webhook URL"
              }
            />
          </Form.Item>
          <Form.Item name="secret" label="签名密钥（可选）">
            <Input.Password allowClear autoComplete="new-password" />
          </Form.Item>
          <Form.Item name="at_mobiles" label="@手机号（钉钉/企业微信）">
            <Select mode="tags" tokenSeparators={[",", " "]} placeholder="例如：13800000000" />
          </Form.Item>
          <Form.Item name="at_user_ids" label="@用户ID（钉钉/企业微信，可选）">
            <Select mode="tags" tokenSeparators={[",", " "]} placeholder="例如：user001" />
          </Form.Item>
          {channelType === "dingding" ? (
            <Form.Item name="is_at_all" label="@所有人（钉钉）" valuePropName="checked">
              <Switch />
            </Form.Item>
          ) : null}

          {channelType === "wechat_work" || channelType === "wechat" ? (
            <Form.Item name="wecom_mode" label="企业微信模式">
              <Select options={wecomModeOptions} />
            </Form.Item>
          ) : null}
          {channelType === "wechat_work" || channelType === "wechat" ? (
            <>
              <Form.Item
                name="corp_id"
                label="企业微信 corpID（app 模式：手机号查 userid 用）"
                hidden={wecomMode !== "app"}
              >
                <Input placeholder="wxcorp..." />
              </Form.Item>
              <Form.Item label="从字典填充 corpID" hidden={wecomMode !== "app"}>
                <DictFillSelect form={form} fieldName="corp_id" options={wecomCorpIDOptions} placeholder="可选：字典中的 corpID" />
              </Form.Item>
              <Form.Item
                name="corp_secret"
                label="企业微信 corpSecret（app 模式：手机号查 userid 用）"
                hidden={wecomMode !== "app"}
              >
                <Input.Password allowClear autoComplete="new-password" />
              </Form.Item>
              <Form.Item label="从字典填充 corpSecret" hidden={wecomMode !== "app"}>
                <DictFillSelect form={form} fieldName="corp_secret" options={wecomCorpSecretOptions} placeholder="可选：字典中的企业密钥" />
              </Form.Item>
              <Form.Item name="agent_id" label="企业微信 agentId（app 模式必填）" hidden={wecomMode !== "app"}>
                <Input placeholder="1000002" />
              </Form.Item>
              <Form.Item label="从字典填充 agentId" hidden={wecomMode !== "app"}>
                <DictFillSelect form={form} fieldName="agent_id" options={wecomAgentIDOptions} placeholder="可选：字典中的 agentId" />
              </Form.Item>
            </>
          ) : null}
          {channelType === "dingding" ? (
            <Form.Item name="ding_mode" label="钉钉模式">
              <Select options={dingModeOptions} />
            </Form.Item>
          ) : null}
          {channelType === "dingding" ? (
            <>
              <Form.Item name="ding_app_key" label="钉钉 appKey（app_chat 模式）" hidden={dingMode !== "app_chat"}>
                <Input />
              </Form.Item>
              <Form.Item label="从字典填充 appKey" hidden={dingMode !== "app_chat"}>
                <DictFillSelect form={form} fieldName="ding_app_key" options={dingAppKeyOptions} placeholder="可选：字典中的应用账号" />
              </Form.Item>
              <Form.Item name="ding_app_secret" label="钉钉 appSecret（app_chat 模式）" hidden={dingMode !== "app_chat"}>
                <Input.Password allowClear autoComplete="new-password" />
              </Form.Item>
              <Form.Item label="从字典填充 appSecret" hidden={dingMode !== "app_chat"}>
                <DictFillSelect form={form} fieldName="ding_app_secret" options={dingAppSecretOptions} placeholder="可选：字典中的应用密钥" />
              </Form.Item>
              <Form.Item name="ding_chat_id" label="钉钉 chatId（app_chat 模式必填 / robot 可不填）">
                <Input placeholder="chatxxxxx" />
              </Form.Item>
              <Form.Item label="从字典填充 chatId" hidden={dingMode !== "app_chat"}>
                <DictFillSelect form={form} fieldName="ding_chat_id" options={dingChatIDOptions} placeholder="可选：字典中的 chatId" />
              </Form.Item>
              <Form.Item name="ding_sign_secret" label="钉钉 signSecret（robot 加签可选）" hidden={dingMode !== "robot"}>
                <Input.Password allowClear autoComplete="new-password" />
              </Form.Item>
              <Form.Item
                label="从字典填充 signSecret"
                hidden={dingMode !== "robot"}
                extra="仅展示数据字典中「启用」且类型为 dingtalk_sign_secret 的条目（与 app_chat 的 dingtalk_app_secret 不是同一配置）。若暂无数据，请到数据字典启用或新增该类型。"
              >
                <DictFillSelect form={form} fieldName="ding_sign_secret" options={dingSignSecretOptions} placeholder="可选：字典中的 signSecret" />
              </Form.Item>
            </>
          ) : null}

          {channelType === "email" ? (
            <Form.Item
              name="email_to"
              label="邮件接收人（email 通道）"
              rules={[
                { required: true, message: "请填写至少一个收件邮箱" },
                {
                  validator: async (_, value) => {
                    const arr = (value ?? []).map((v: string) => String(v).trim()).filter(Boolean);
                    if (arr.length === 0) {
                      throw new Error("请填写至少一个收件邮箱");
                    }
                    const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                    for (const e of arr) {
                      if (!re.test(e)) {
                        throw new Error(`邮箱格式不正确：${e}`);
                      }
                    }
                  },
                },
              ]}
            >
              <Select mode="tags" tokenSeparators={[",", " ", ";"]} placeholder="输入后回车，或粘贴 a@xx.com,b@yy.com" />
            </Form.Item>
          ) : null}
          <Form.Item
            name="extra_headers_json"
            label="额外配置 JSON（可选）"
            extra={'用于自定义请求头。示例：{"headers":{"Authorization":"Bearer xxxxx"}}'}
            rules={[
              {
                validator: async (_, value) => {
                  const s = String(value || "").trim();
                  if (!s) return;
                  try {
                    const obj = JSON.parse(s);
                    if (!obj || typeof obj !== "object" || Array.isArray(obj)) {
                      throw new Error("JSON 必须是对象");
                    }
                  } catch {
                    throw new Error("JSON 格式不正确");
                  }
                },
              },
            ]}
          >
            <Input.TextArea rows={4} />
          </Form.Item>
          <Space style={{ width: "100%" }} size="large">
            <Form.Item name="timeout_ms" label="超时(ms)" style={{ width: 200 }}>
              <InputNumber min={1000} max={60000} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
      <Drawer
        title="通知模板可视化"
        open={templateOpen}
        onClose={() => setTemplateOpen(false)}
        width={1120}
        destroyOnClose
        extra={
          <Space>
            <Button onClick={() => setTemplateOpen(false)}>取消</Button>
            <Button type="primary" loading={templateSaving} onClick={() => void submitTemplate()}>
              保存模板
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
          左侧编辑 Go 模板，右侧模拟钉钉/企微通知气泡实时预览。变量说明与 sample_payload 见下方折叠区。
          <Link to="/alert-config-center" style={{ marginLeft: 8 }}>策略与联调</Link>
        </Typography.Paragraph>
        <Form form={templateForm} layout="vertical">
          <Form.Item label="当前通道">
            <Select
              value={templateChannelID}
              onChange={(v) => {
                setTemplateChannelID(v);
                openTemplateConfig(v);
              }}
              options={list.map((it) => ({ label: `${it.name} (${it.type})`, value: it.id }))}
            />
          </Form.Item>
          <AlertNotificationTemplateStudio
            form={templateForm}
            projects={projects}
            channelType={templateChannelRow?.type}
            editingTarget={editingTemplateTarget}
            onEditingTargetChange={setEditingTemplateTarget}
            firingPreset={firingPreset}
            resolvedPreset={resolvedPreset}
            onFiringPresetChange={setFiringPreset}
            onResolvedPresetChange={setResolvedPreset}
            firingTemplateRef={firingTemplateRef}
            resolvedTemplateRef={resolvedTemplateRef}
          />
        </Form>
      </Drawer>
      <Modal
        title={testRow ? `测试发送 #${testRow.id} ${testRow.name}` : "测试发送"}
        open={testOpen}
        onCancel={() => setTestOpen(false)}
        onOk={() => void submitTest()}
        confirmLoading={testSending}
        destroyOnClose
      >
        <Form form={testForm} layout="vertical">
          <Form.Item name="status" label="测试状态" rules={[{ required: true, message: "请选择测试状态" }]}>
            <Select
              options={[
                { label: "触发（firing）", value: "firing" },
                { label: "恢复（resolved）", value: "resolved" },
              ]}
            />
          </Form.Item>
          <Form.Item name="severity" label="级别（可选）">
            <Input placeholder="默认 info" />
          </Form.Item>
          <Form.Item name="title" label="标题（可选）">
            <Input placeholder="留空则按状态自动生成测试标题" />
          </Form.Item>
          <Form.Item name="content" label="内容（可选）">
            <Input.TextArea rows={3} placeholder="留空则自动生成测试内容" />
          </Form.Item>
          {testReceipt ? (
            <Form.Item label="投递回执">
              <Typography.Paragraph type={testReceipt.success ? undefined : "danger"}>
                HTTP {testReceipt.http_status_code ?? "-"} · {testReceipt.success ? "成功" : "失败"}
              </Typography.Paragraph>
              {testReceipt.error_message ? (
                <Typography.Text type="danger">{testReceipt.error_message}</Typography.Text>
              ) : null}
              {testReceipt.response_body ? (
                <Input.TextArea rows={4} readOnly value={testReceipt.response_body} />
              ) : null}
            </Form.Item>
          ) : null}
        </Form>
      </Modal>
    </Card>
    </div>
  );
}

