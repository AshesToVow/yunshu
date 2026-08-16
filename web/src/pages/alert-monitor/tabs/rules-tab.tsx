import { extractApiErrorMessage } from "../../../services/http";
import { Alert, Button, Form, Input, InputNumber, Modal, Segmented, Select, Space, Switch, Table, Typography, message } from "antd";
import { PlusOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import { useEffect, useMemo, useState } from "react";
import { useAlertMonitor } from "../context";
import {
  createAlertMonitorRuleFromTemplate,
  importPrometheusYAML,
  listAlertRuleTemplates,
  type AlertDatasourceItem,
  type AlertRuleTemplateItem,
} from "../../../services/alert-platform";

export function RulesTab() {
  const ctx = useAlertMonitor();
  const [tplOpen, setTplOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [tplLoading, setTplLoading] = useState(false);
  const [templates, setTemplates] = useState<AlertRuleTemplateItem[]>([]);
  const [group, setGroup] = useState<string>("");
  const [selected, setSelected] = useState<AlertRuleTemplateItem | null>(null);
  const [form] = Form.useForm();
  const [importForm] = Form.useForm();
  const [importPreview, setImportPreview] = useState<
    Array<{ group_name: string; name: string; expr: string; for_seconds: number; severity: string }>
  >([]);

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

  const datasources = ctx.dsList ?? [];

  useEffect(() => {
    if (!selected) return;
    form.setFieldsValue({
      datasource_id: datasources[0]?.id,
      name: selected.name,
      threshold: selected.default_params?.threshold,
    });
  }, [selected, datasources, form]);

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
          icon={<UploadOutlined />}
          onClick={() => {
            importForm.resetFields();
            importForm.setFieldsValue({
              datasource_id: datasources[0]?.id,
              enabled: true,
              dry_run: true,
            });
            setImportPreview([]);
            setImportOpen(true);
          }}
        >
          导入 Prometheus YAML
        </Button>
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
          · 告警由本平台规则中心评测数据源产生（Telegraf / blackbox / Pushgateway → Prom/VM）；在此新建、调整、启停规则
        </Typography.Text>
      </Space>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
        规则中心是唯一告警产生入口：平台定时对数据源执行 PromQL，命中后经屏蔽 / 抑制 / 订阅树投递到钉钉、企微、邮件等。可「新建规则」或「从模板创建」（含 Telegraf、blackbox、Pushgateway）。规则级「处理人」与值班表当前班次邮箱会在 outgoing 中合并去重。
      </Typography.Paragraph>
      <Alert
        type="info"
        showIcon
        message="规则配置建议"
        description={
          <Space direction="vertical" size={8} style={{ width: "100%" }}>
            <span>
              建议先确认：1) <Typography.Text code>datasource</Typography.Text> 指向正确 Prom/VM；2){" "}
              <Typography.Text code>severity</Typography.Text> 与通知路由匹配；3){" "}
              <Typography.Text code>for_seconds</Typography.Text> 防抖；4){" "}
              <Typography.Text code>eval_interval_seconds</Typography.Text>（常用 30s/60s）。采集侧见{" "}
              <Typography.Text code>deploy/monitoring/</Typography.Text>（Consul SD + Telegraf 样例）。
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
                options={datasources.map((d: AlertDatasourceItem) => ({
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
      <Modal
        title="导入 Prometheus 规则 YAML"
        open={importOpen}
        onCancel={() => setImportOpen(false)}
        width={860}
        destroyOnClose
        footer={[
          <Button key="cancel" onClick={() => setImportOpen(false)}>
            取消
          </Button>,
          <Button
            key="dry"
            onClick={async () => {
              try {
                const v = await importForm.validateFields();
                const res = await importPrometheusYAML({
                  datasource_id: Number(v.datasource_id),
                  yaml: v.yaml,
                  enabled: !!v.enabled,
                  dry_run: true,
                });
                setImportPreview(res.preview || []);
                message.success(`预览 ${res.preview?.length || 0} 条告警规则（跳过 recording ${res.skipped}）`);
                if (res.errors?.length) message.warning(res.errors.slice(0, 3).join("; "));
              } catch (e) {
                if (e && typeof e === "object" && "errorFields" in e) return;
                message.error(extractApiErrorMessage(e, "预览失败"));
              }
            }}
          >
            预览
          </Button>,
          <Button
            key="ok"
            type="primary"
            onClick={async () => {
              try {
                const v = await importForm.validateFields();
                const res = await importPrometheusYAML({
                  datasource_id: Number(v.datasource_id),
                  yaml: v.yaml,
                  enabled: !!v.enabled,
                  dry_run: false,
                });
                setImportPreview(res.preview || []);
                message.success(`已创建 ${res.created} 条，跳过 ${res.skipped}`);
                if (res.errors?.length) message.warning(res.errors.slice(0, 3).join("; "));
                setImportOpen(false);
                await ctx.loadRules(ctx.projectContextId);
              } catch (e) {
                if (e && typeof e === "object" && "errorFields" in e) return;
                message.error(extractApiErrorMessage(e, "导入失败"));
              }
            }}
          >
            确认导入
          </Button>,
        ]}
      >
        <Form form={importForm} layout="vertical">
          <Form.Item name="datasource_id" label="目标数据源" rules={[{ required: true }]}>
            <Select
              options={datasources.map((d: AlertDatasourceItem) => ({
                value: d.id,
                label: d.name,
              }))}
            />
          </Form.Item>
          <Form.Item name="enabled" label="导入后启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="yaml"
            label="Prometheus rules YAML"
            rules={[{ required: true, message: "请粘贴 YAML" }]}
            extra="支持标准 groups[].rules[].alert；recording 规则自动跳过。"
          >
            <Input.TextArea rows={12} placeholder={"groups:\n  - name: example\n    rules:\n      - alert: InstanceDown\n        expr: up == 0\n        for: 2m\n        labels:\n          severity: critical"} />
          </Form.Item>
        </Form>
        {importPreview.length > 0 ? (
          <Table
            size="small"
            rowKey={(r) => `${r.group_name}-${r.name}`}
            dataSource={importPreview}
            pagination={false}
            columns={[
              { title: "组", dataIndex: "group_name", width: 120 },
              { title: "告警名", dataIndex: "name", width: 160 },
              { title: "级别", dataIndex: "severity", width: 90 },
              { title: "for(s)", dataIndex: "for_seconds", width: 80 },
              { title: "expr", dataIndex: "expr", ellipsis: true },
            ]}
          />
        ) : null}
      </Modal>
    </Space>
  );
}
