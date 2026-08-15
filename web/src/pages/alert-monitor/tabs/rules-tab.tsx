import { extractApiErrorMessage } from "../../../services/http";
import { Alert, Button, Form, Input, InputNumber, Modal, Segmented, Select, Space, Table, Typography, message } from "antd";
import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { useEffect, useMemo, useState } from "react";
import { useAlertMonitor } from "../context";
import {
  createAlertMonitorRuleFromTemplate,
  listAlertRuleTemplates,
  type AlertDatasourceItem,
  type AlertRuleTemplateItem,
} from "../../../services/alert-platform";

export function RulesTab() {
  const ctx = useAlertMonitor();
  const [tplOpen, setTplOpen] = useState(false);
  const [tplLoading, setTplLoading] = useState(false);
  const [templates, setTemplates] = useState<AlertRuleTemplateItem[]>([]);
  const [group, setGroup] = useState<string>("");
  const [selected, setSelected] = useState<AlertRuleTemplateItem | null>(null);
  const [form] = Form.useForm();

  const groups = useMemo(() => {
    const set = new Set(templates.map((t) => t.group));
    return Array.from(set);
  }, [templates]);

  const filtered = useMemo(
    () => (group ? templates.filter((t) => t.group === group) : templates),
    [templates, group],
  );

  async function openTemplates() {
    setTplOpen(true);
    setTplLoading(true);
    try {
      setTemplates(await listAlertRuleTemplates());
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载模板失败"));
    } finally {
      setTplLoading(false);
    }
  }

  useEffect(() => {
    if (!selected) return;
    form.setFieldsValue({
      datasource_id: ctx.datasources[0]?.id,
      name: selected.name,
      threshold: selected.default_params?.threshold,
    });
  }, [selected, ctx.datasources, form]);

  async function submitFromTemplate() {
    if (!selected) return;
    try {
      const v = await form.validateFields();
      const params: Record<string, string> = {};
      if (v.threshold != null && String(v.threshold).trim() !== "") {
        params.threshold = String(v.threshold);
      }
      await createAlertMonitorRuleFromTemplate({
        template_id: selected.id,
        datasource_id: Number(v.datasource_id),
        name: v.name,
        params,
      });
      message.success("已从模板创建规则（含 category label，可在订阅树按 category 路由）");
      setTplOpen(false);
      setSelected(null);
      await ctx.loadRules(ctx.projectContextId);
    } catch (e) {
      if (e && typeof e === "object" && "errorFields" in e) return;
      message.error(extractApiErrorMessage(e, "从模板创建失败"));
    }
  }

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Space wrap align="center">
        <Button type="primary" icon={<PlusOutlined />} onClick={ctx.openRuleCreate}>
          新建规则
        </Button>
        <Button onClick={() => void openTemplates()}>从模板创建</Button>
        <Button
          icon={<ReloadOutlined />}
          onClick={() => void Promise.all([ctx.loadRules(ctx.projectContextId), ctx.loadDatasources(ctx.projectContextId)])}
        >
          刷新
        </Button>
        <Segmented
          value={ctx.ruleEnabledFilter}
          options={[
            { label: `全部 (${ctx.ruleEnabledStats.total})`, value: "all" },
            { label: `启用 (${ctx.ruleEnabledStats.enabled})`, value: "enabled" },
            { label: `停用 (${ctx.ruleEnabledStats.disabled})`, value: "disabled" },
          ]}
          onChange={(v) => ctx.setRuleEnabledFilter(v as "all" | "enabled" | "disabled")}
        />
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          当前显示 {ctx.ruleDisplayList.length} 条
          {ctx.ruleEnabledFilter !== "all" ? `（共 ${ctx.ruleEnabledStats.total} 条）` : ""}
          · 生产核心告警建议走 Prometheus+Alertmanager → Webhook；平台规则为轻量补充
        </Typography.Text>
      </Space>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
        这里配置的是平台内监控规则：平台会定时对数据源执行 PromQL，命中后走与 Webhook 入站相同的通知与历史记录链路。规则级「处理人」与所选「值班表」当前班次通知邮箱会在告警
        outgoing 中合并去重；部门选择为根部门子树全员。
      </Typography.Paragraph>
      <Alert
        type="info"
        showIcon
        message="规则配置建议"
        description={
          <Space direction="vertical" size={8} style={{ width: "100%" }}>
            <span>
              建议先确认四项：1) <Typography.Text code>datasource</Typography.Text> 选正确集群；2){" "}
              <Typography.Text code>severity</Typography.Text> 与策略匹配（critical/warning/info）；3){" "}
              <Typography.Text code>for_seconds</Typography.Text> 防抖时长；4){" "}
              <Typography.Text code>eval_interval_seconds</Typography.Text> 评估频率（常用 30s/60s）。
            </span>
            <Space wrap>
              <Button size="small" onClick={ctx.openHistoryTab}>
                查看规则触发历史
              </Button>
            </Space>
          </Space>
        }
      />
      <Table rowKey="id" columns={ctx.ruleColumns} dataSource={ctx.ruleDisplayList} pagination={{ pageSize: 20, showSizeChanger: true }} scroll={{ x: 1100 }} />

      <Modal
        title="从规则模板创建"
        open={tplOpen}
        onCancel={() => {
          setTplOpen(false);
          setSelected(null);
        }}
        onOk={() => void submitFromTemplate()}
        okButtonProps={{ disabled: !selected }}
        width={820}
        destroyOnClose
      >
        <Space style={{ marginBottom: 12 }} wrap>
          <Select
            allowClear
            placeholder="分组筛选"
            style={{ width: 160 }}
            value={group || undefined}
            onChange={(v) => setGroup(v || "")}
            options={groups.map((g) => ({ value: g, label: g }))}
          />
        </Space>
        <Table
          rowKey="id"
          size="small"
          loading={tplLoading}
          dataSource={filtered}
          pagination={{ pageSize: 20, showSizeChanger: true }}
          rowSelection={{
            type: "radio",
            selectedRowKeys: selected ? [selected.id] : [],
            onChange: (_keys, rows) => setSelected(rows[0] || null),
          }}
          columns={[
            { title: "分组", dataIndex: "group", width: 100 },
            { title: "名称", dataIndex: "name", width: 180 },
            { title: "说明", dataIndex: "description" },
            { title: "级别", dataIndex: "severity", width: 90 },
          ]}
        />
        {selected ? (
          <Form form={form} layout="vertical" style={{ marginTop: 16 }}>
            <Form.Item name="datasource_id" label="数据源" rules={[{ required: true, message: "请选择数据源" }]}>
              <Select
                options={ctx.datasources.map((d: AlertDatasourceItem) => ({
                  value: d.id,
                  label: `${d.name}${d.project_id ? ` (项目 ${d.project_id})` : ""}`,
                }))}
              />
            </Form.Item>
            <Form.Item name="name" label="规则名称">
              <Input />
            </Form.Item>
            {selected.default_params?.threshold != null ? (
              <Form.Item name="threshold" label="阈值 threshold">
                <InputNumber style={{ width: "100%" }} />
              </Form.Item>
            ) : null}
            <Typography.Text type="secondary" style={{ fontSize: 12 }}>
              将写入 labels.category={selected.labels?.category || selected.group}，可在订阅树按 category 匹配路由。
            </Typography.Text>
          </Form>
        ) : null}
      </Modal>
    </Space>
  );
}
