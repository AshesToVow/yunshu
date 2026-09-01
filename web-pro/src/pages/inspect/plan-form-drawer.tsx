// @ts-nocheck
// 巡检计划表单与巡检项/报告版式编辑弹窗（RF-10）。
// 由 web/src/pages/project-inspect-page.tsx 原样搬迁，页面只保留状态与编排。
import {
  Alert,
  Button,
  Col,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  message,
} from "antd";
import type { FormInstance } from "antd/es/form";
import { Link } from '@umijs/max';
import type { AlertDatasourceItem } from "../../services/alert-platform";
import {
  createInspectItem,
  previewInspectReportTemplate,
  updateInspectItem,
  updateInspectPlan,
  updateInspectReportTemplate,
  type InspectItem,
  type InspectReportTemplate,
} from "../../services/inspect";
import { extractApiErrorMessage } from "../../services/http";
import { CRON_PRESETS, THRESHOLD_TYPE_OPTIONS } from "./display";
import { toReportBlob } from "../../utils/inspect-report-download";

type InspectPlanFormSectionProps = {
  planForm: FormInstance;
  projectId: number;
  dsList: AlertDatasourceItem[];
  reportTemplates: InspectReportTemplate[];
  onSaved: () => void;
  onGoToItems: () => void;
  onGoToRuns: () => void;
};

export function InspectPlanFormSection({
  planForm,
  projectId,
  dsList,
  reportTemplates,
  onSaved,
  onGoToItems,
  onGoToRuns,
}: InspectPlanFormSectionProps) {
  return (
    <>
      <Alert
        type="info"
        showIcon
        className="project-inspect-page__guide"
        message="适配 Prometheus + Telegraf + Blackbox + Pushgateway"
        description={
          <ol className="project-inspect-page__guide-list">
            <li>在「告警监控平台」为本项目配置 Prometheus 数据源（指向你们的 Prometheus）。</li>
            <li>
              主机/中间件指标：在对应服务器 <code>telegraf.conf</code> 配置 <code>inputs.*</code>
              ，由 Prometheus 拉取后再启用巡检项。
            </li>
            <li>
              连通性/端口：使用 Blackbox 的 <code>probe_success</code>
              （ICMP/TCP/HTTP 的 job 名按 scrape 配置调整 PromQL）。
            </li>
            <li>批次任务：Pushgateway 推送后，按 job 名改「Pushgateway」相关巡检项。</li>
            <li>已有旧模板时，可在「巡检项」页签点击「重置为 Telegraf 模板」一键重建。</li>
          </ol>
        }
      />
      <Form
        form={planForm}
        layout="vertical"
        className="project-inspect-page__plan-form"
        onFinish={async (values) => {
          try {
            const list = String(values.recipients || "")
              .split(/[,;\s]+/)
              .map((s: string) => s.trim())
              .filter(Boolean);
            await updateInspectPlan(projectId, {
              enabled: values.enabled,
              cron_spec: values.cron_spec,
              datasource_id: values.datasource_id,
              report_list_mode: values.report_list_mode,
              report_template_id: values.report_template_id || 0,
              retain_days: values.retain_days,
              recipients: list,
            });
            message.success("计划已保存");
            onSaved();
          } catch (e) {
            message.error(extractApiErrorMessage(e, "保存失败"));
          }
        }}
      >
        <Row gutter={16}>
          <Col xs={24} md={12}>
            <Form.Item name="enabled" label="启用定时巡检" valuePropName="checked">
              <Switch checkedChildren="开" unCheckedChildren="关" />
            </Form.Item>
            <Form.Item
              name="datasource_id"
              label="Prometheus 数据源"
              rules={[{ required: true, message: "请选择数据源" }]}
              extra={<Link to={`/alert-monitor-platform/datasources?project_id=${projectId}`}>管理数据源</Link>}
            >
              <Select
                allowClear
                options={dsList.map((d) => ({
                  label: `${d.name} (#${d.id})`,
                  value: d.id,
                }))}
                placeholder="选择项目内数据源"
              />
            </Form.Item>
            <Form.Item
              name="cron_spec"
              label="Cron 表达式"
              extra="秒 分 时 日 月 周；可选手动输入自定义表达式"
            >
              <Select
                showSearch
                allowClear
                options={CRON_PRESETS}
                placeholder="选择或输入 Cron"
                dropdownRender={(menu) => (
                  <>
                    {menu}
                    <div style={{ padding: 8 }}>
                      <Input
                        placeholder="自定义 Cron，回车填入"
                        onPressEnter={(e) =>
                          planForm.setFieldValue("cron_spec", (e.target as HTMLInputElement).value)
                        }
                      />
                    </div>
                  </>
                )}
              />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item name="report_list_mode" label="报告明细模式">
              <Select
                options={[
                  { label: "仅异常（推荐日常）", value: "abnormal_only" },
                  { label: "摘要（按项汇总）", value: "summary" },
                  { label: "全部样本", value: "all" },
                ]}
              />
            </Form.Item>
            <Form.Item name="report_template_id" label="报告版式">
              <Select
                allowClear
                placeholder="默认标准版"
                options={reportTemplates.map((t) => ({
                  label: `${t.name}${t.project_id === 0 ? "（全局）" : ""}`,
                  value: t.id,
                }))}
              />
            </Form.Item>
            <Form.Item name="retain_days" label="报告保留天数（0=不清理）">
              <InputNumber min={0} max={3650} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item
              name="recipients"
              label="邮件收件人"
              extra="多个地址用逗号或空格分隔；需平台已配置发信"
            >
              <Input.TextArea rows={2} placeholder="ops@example.com, oncall@example.com" />
            </Form.Item>
          </Col>
        </Row>
        <Space wrap>
          <Button type="primary" htmlType="submit">
            保存计划
          </Button>
          <Button onClick={onGoToItems}>去配置巡检项</Button>
          <Button onClick={onGoToRuns}>查看历史</Button>
        </Space>
      </Form>
    </>
  );
}

type InspectReportTemplateFormModalProps = {
  open: boolean;
  onClose: () => void;
  tplForm: FormInstance;
  editingTpl: InspectReportTemplate | null;
  projectId: number;
  onSaved: () => void;
};

export function InspectReportTemplateFormModal({
  open,
  onClose,
  tplForm,
  editingTpl,
  projectId,
  onSaved,
}: InspectReportTemplateFormModalProps) {
  return (
    <Modal
      title="编辑项目报告版式"
      open={open}
      onCancel={onClose}
      width={880}
      destroyOnClose
      footer={[
        <Button
          key="preview"
          onClick={async () => {
            try {
              const values = await tplForm.validateFields();
              const resp = await previewInspectReportTemplate(projectId, {
                code: editingTpl?.code,
                body: values.body,
              });
              const blob = toReportBlob(resp, "text/html;charset=utf-8");
              const url = URL.createObjectURL(blob);
              window.open(url, "_blank", "noopener,noreferrer");
              setTimeout(() => URL.revokeObjectURL(url), 60_000);
            } catch (e) {
              message.error(extractApiErrorMessage(e, "预览失败"));
            }
          }}
        >
          预览
        </Button>,
        <Button key="cancel" onClick={onClose}>
          取消
        </Button>,
        <Button
          key="ok"
          type="primary"
          onClick={async () => {
            if (!editingTpl) return;
            try {
              const values = await tplForm.validateFields();
              await updateInspectReportTemplate(projectId, editingTpl.id, values);
              message.success("已保存");
              onClose();
              onSaved();
            } catch (e) {
              message.error(extractApiErrorMessage(e, "保存失败"));
            }
          }}
        >
          保存
        </Button>,
      ]}
    >
      <Form form={tplForm} layout="vertical">
        <Form.Item name="name" label="名称" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="remark" label="说明">
          <Input />
        </Form.Item>
        <Form.Item
          name="body"
          label="HTML 模板"
          rules={[{ required: true, message: "请填写模板正文" }]}
          extra="Go html/template 语法；可用字段：Project、Score、Grade、Summary、Groups、Findings 等。"
        >
          <Input.TextArea
            rows={18}
            style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" }}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}

type InspectItemFormModalProps = {
  open: boolean;
  onClose: () => void;
  itemForm: FormInstance;
  editingItem: InspectItem | null;
  projectId: number;
  onSaved: () => void;
};

export function InspectItemFormModal({
  open,
  onClose,
  itemForm,
  editingItem,
  projectId,
  onSaved,
}: InspectItemFormModalProps) {
  return (
    <Modal
      title={editingItem ? "编辑巡检项" : "新增巡检项"}
      open={open}
      onCancel={onClose}
      onOk={async () => {
        try {
          const values = await itemForm.validateFields();
          if (editingItem) {
            await updateInspectItem(projectId, editingItem.id, values);
          } else {
            await createInspectItem(projectId, values);
          }
          message.success("已保存");
          onClose();
          onSaved();
        } catch (e) {
          if (e && typeof e === "object" && "errorFields" in e) return;
          message.error(extractApiErrorMessage(e, "保存失败"));
        }
      }}
      destroyOnClose
      width={720}
    >
      <Form form={itemForm} layout="vertical">
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="type" label="分类" rules={[{ required: true }]}>
              <Input placeholder="如：基础设施层 / 数据库监控" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="name" label="名称" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="description" label="说明">
          <Input.TextArea rows={2} placeholder="指标来源、标签约定、何时启用等" />
        </Form.Item>
        <Form.Item
          name="query"
          label="PromQL"
          rules={[{ required: true }]}
          extra="即时向量查询；无数据时该项会记为异常。"
        >
          <Input.TextArea rows={3} style={{ fontFamily: "ui-monospace, Menlo, Consolas, monospace" }} />
        </Form.Item>
        <Row gutter={16}>
          <Col xs={24} sm={8}>
            <Form.Item name="threshold_type" label="比较方式" rules={[{ required: true }]}>
              <Select options={THRESHOLD_TYPE_OPTIONS} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={6}>
            <Form.Item name="threshold" label="阈值" rules={[{ required: true }]}>
              <InputNumber style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col xs={12} sm={4}>
            <Form.Item name="unit" label="单位">
              <Input placeholder="%" />
            </Form.Item>
          </Col>
          <Col xs={24} sm={6}>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
}
