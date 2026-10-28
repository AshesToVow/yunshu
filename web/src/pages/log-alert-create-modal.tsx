import { Alert, Button, Form, Input, InputNumber, Modal, Segmented, Select, Space, Typography, message } from "antd";
import type { Dayjs } from "dayjs";
import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { createAlertMonitorRule } from "../services/alert-platform";
import { extractApiErrorMessage } from "../services/http";
import {
  LOG_ALERT_WINDOW_MAX_MINUTES,
  buildLogAlertRulePayload,
  inferLogAlertWindow,
  type LogAlertMode,
} from "../utils/log-alert-rule";

export type LogAlertCreateInitial = {
  projectId?: number;
  projectLabel?: string;
  level?: string;
  keyword?: string;
  serviceName?: string;
  namespace?: string;
  range?: [Dayjs, Dayjs] | null;
};

type FormValues = {
  name: string;
  mode: LogAlertMode;
  level?: string;
  keyword?: string;
  service_name?: string;
  namespace?: string;
  window_minutes: number;
  threshold: number;
  baseline_minutes?: number;
  spike_ratio?: number;
  severity: string;
  eval_interval_seconds: number;
};

export function LogAlertCreateModal({
  open,
  initial,
  onClose,
}: {
  open: boolean;
  initial: LogAlertCreateInitial;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const [form] = Form.useForm<FormValues>();
  const [submitting, setSubmitting] = useState(false);
  const [createdName, setCreatedName] = useState("");
  const mode = Form.useWatch("mode", form) || "error_count";

  useEffect(() => {
    if (!open) return;
    setCreatedName("");
    const windowMinutes = inferLogAlertWindow(initial.range);
    const level = String(initial.level || "").trim().toUpperCase();
    const serviceName = String(initial.serviceName || "").trim();
    form.setFieldsValue({
      name: `日志告警 · ${serviceName || "全部服务"} · ${level || "ERROR"}`,
      mode: "error_count",
      level,
      keyword: initial.keyword || "",
      service_name: serviceName,
      namespace: initial.namespace || "",
      window_minutes: windowMinutes,
      threshold: 10,
      baseline_minutes: Math.min(720, Math.max(windowMinutes, windowMinutes * 6)),
      spike_ratio: 3,
      severity: "warning",
      eval_interval_seconds: 60,
    });
  }, [open, initial, form]);

  async function submit() {
    if (!initial.projectId) {
      message.warning("请先在检索页选择项目");
      return;
    }
    let v: FormValues;
    try {
      v = await form.validateFields();
    } catch {
      return;
    }
    setSubmitting(true);
    try {
      const payload = buildLogAlertRulePayload({
        projectId: initial.projectId,
        name: v.name,
        mode: v.mode,
        level: v.level,
        keyword: v.keyword,
        serviceName: v.service_name,
        namespace: v.namespace,
        windowMinutes: v.window_minutes,
        threshold: v.threshold,
        baselineMinutes: v.baseline_minutes,
        spikeRatio: v.spike_ratio,
        severity: v.severity,
        evalIntervalSeconds: v.eval_interval_seconds,
      });
      await createAlertMonitorRule(payload);
      setCreatedName(String(v.name || "").trim());
      message.success("日志告警规则已创建并立即生效");
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e, "创建日志告警规则失败"));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      title="从当前日志筛选创建告警"
      open={open}
      onCancel={onClose}
      width={640}
      destroyOnClose
      footer={
        createdName
          ? [
              <Button key="close" onClick={onClose}>
                关闭
              </Button>,
              <Button
                key="go"
                type="primary"
                onClick={() =>
                  navigate(
                    initial.projectId
                      ? `/alert-monitor-platform/rules?project_id=${initial.projectId}`
                      : "/alert-monitor-platform/rules",
                  )
                }
              >
                打开规则中心
              </Button>,
            ]
          : [
              <Button key="cancel" onClick={onClose}>
                取消
              </Button>,
              <Button key="ok" type="primary" loading={submitting} disabled={!initial.projectId} onClick={() => void submit()}>
                创建并启用
              </Button>,
            ]
      }
    >
      {createdName ? (
        <Alert
          type="success"
          showIcon
          message={`已创建日志告警规则「${createdName}」`}
          description="规则由日志评估器按评估间隔扫描 ES 计数，命中后进入统一通知链路；可在规则中心继续配置处理人、值班与静默。"
        />
      ) : (
        <>
          {initial.projectId ? (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
              已带出当前检索条件：项目 {initial.projectLabel || initial.projectId}
              {initial.level ? ` · 级别 ${initial.level}` : ""}
              {initial.serviceName ? ` · 服务 ${initial.serviceName}` : ""}
              {initial.namespace ? ` · 命名空间 ${initial.namespace}` : ""}
              {initial.keyword ? ` · 关键词 ${initial.keyword}` : ""}
            </Typography.Paragraph>
          ) : (
            <Alert type="warning" showIcon message="请先在检索页选择项目，再创建日志告警" style={{ marginBottom: 12 }} />
          )}
          <Form form={form} layout="vertical" size="middle">
            <Form.Item name="name" label="规则名称" rules={[{ required: true, message: "请填写规则名称" }]}>
              <Input maxLength={128} placeholder="例如：订单服务 ERROR 突增" />
            </Form.Item>
            <Form.Item name="mode" label="判定方式">
              <Segmented
                options={[
                  { label: "错误数阈值", value: "error_count" },
                  { label: "错误量突增", value: "error_spike" },
                ]}
              />
            </Form.Item>
            <Space size="middle" style={{ display: "flex" }}>
              <Form.Item name="level" label="级别过滤" style={{ flex: 1 }}>
                <Input placeholder="ERROR（留空则不限级别）" />
              </Form.Item>
              <Form.Item name="window_minutes" label={`评估窗口（≤${LOG_ALERT_WINDOW_MAX_MINUTES} 分钟）`} style={{ flex: 1 }}>
                <InputNumber min={1} max={LOG_ALERT_WINDOW_MAX_MINUTES} style={{ width: "100%" }} addonAfter="分钟" />
              </Form.Item>
              <Form.Item name="threshold" label="触发阈值" style={{ flex: 1 }}>
                <InputNumber min={1} style={{ width: "100%" }} addonAfter="条" />
              </Form.Item>
            </Space>
            {mode === "error_spike" ? (
              <Space size="middle" style={{ display: "flex" }}>
                <Form.Item name="baseline_minutes" label="基线窗口" style={{ flex: 1 }}>
                  <InputNumber min={1} max={720} style={{ width: "100%" }} addonAfter="分钟" />
                </Form.Item>
                <Form.Item name="spike_ratio" label="突增倍率" style={{ flex: 1 }}>
                  <InputNumber min={1.1} step={0.5} style={{ width: "100%" }} addonAfter="倍" />
                </Form.Item>
              </Space>
            ) : null}
            <Space size="middle" style={{ display: "flex" }}>
              <Form.Item name="service_name" label="服务" style={{ flex: 1 }}>
                <Input placeholder="留空则不限服务" />
              </Form.Item>
              <Form.Item name="namespace" label="命名空间" style={{ flex: 1 }}>
                <Input placeholder="留空则不限命名空间" />
              </Form.Item>
            </Space>
            <Form.Item name="keyword" label="关键词 / DQL">
              <Input allowClear placeholder="例如：timeout 或 level:ERROR service:api" />
            </Form.Item>
            <Space size="middle" style={{ display: "flex" }}>
              <Form.Item name="severity" label="严重度" style={{ flex: 1 }}>
                <Select
                  options={[
                    { label: "critical", value: "critical" },
                    { label: "warning", value: "warning" },
                    { label: "info", value: "info" },
                  ]}
                />
              </Form.Item>
              <Form.Item name="eval_interval_seconds" label="评估间隔" style={{ flex: 1 }}>
                <Select
                  options={[
                    { label: "30 秒", value: 30 },
                    { label: "1 分钟", value: 60 },
                    { label: "5 分钟", value: 300 },
                  ]}
                />
              </Form.Item>
            </Space>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
              告警文案与标签会自动带上 <Typography.Text code>category=log</Typography.Text>，可在订阅树按 category
              路由到值班人；通知接收人仍由规则「处理人 / 值班」决定。
            </Typography.Paragraph>
          </Form>
        </>
      )}
    </Modal>
  );
}
