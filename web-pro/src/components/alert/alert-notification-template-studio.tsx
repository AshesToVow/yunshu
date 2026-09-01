// @ts-nocheck
import { Collapse, Col, Form, Input, Row, Segmented, Select, Space, Table, Tag, Typography, type FormInstance } from "antd";
import { useEffect, useRef, useState } from "react";
import {
  previewAlertChannelTemplate,
  type AlertTemplatePreviewResult,
  type AlertTemplateVariableDoc,
} from "../../services/alerts";
import type { ProjectItem } from "../../services/projects";
import {
  CHANNEL_PRESET_OPTIONS,
  type TemplatePreviewStatus,
  presetTemplateByMode,
} from "./notification-template-presets";
import { NotificationPreviewBubble } from "./notification-preview-bubble";

export type AlertNotificationTemplateStudioProps = {
  form: FormInstance;
  projects: ProjectItem[];
  channelType?: string;
  editingTarget: TemplatePreviewStatus;
  onEditingTargetChange: (t: TemplatePreviewStatus) => void;
  firingPreset: string;
  resolvedPreset: string;
  onFiringPresetChange: (v: string) => void;
  onResolvedPresetChange: (v: string) => void;
  firingTemplateRef?: React.RefObject<unknown>;
  resolvedTemplateRef?: React.RefObject<unknown>;
};

export function AlertNotificationTemplateStudio({
  form,
  projects,
  channelType,
  editingTarget,
  onEditingTargetChange,
  firingPreset,
  resolvedPreset,
  onFiringPresetChange,
  onResolvedPresetChange,
  firingTemplateRef,
  resolvedTemplateRef,
}: AlertNotificationTemplateStudioProps) {
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState("");
  const [previewResult, setPreviewResult] = useState<AlertTemplatePreviewResult | null>(null);
  const previewSeqRef = useRef(0);

  const templateFiring = Form.useWatch("template_firing", form);
  const templateResolved = Form.useWatch("template_resolved", form);
  const previewStatus = (Form.useWatch("preview_status", form) || "firing") as TemplatePreviewStatus;
  const previewProjectID = Form.useWatch("preview_project_id", form) as number | undefined;
  const previewRawPayloadJSON = Form.useWatch("preview_raw_payload_json", form) as string | undefined;

  useEffect(() => {
    const timer = window.setTimeout(() => {
      const seq = ++previewSeqRef.current;
      setPreviewLoading(true);
      setPreviewError("");
      void previewAlertChannelTemplate({
        template_firing: String(templateFiring || ""),
        template_resolved: String(templateResolved || ""),
        status: previewStatus,
        project_id: previewProjectID,
        raw_payload_json: String(previewRawPayloadJSON || ""),
      })
        .then((res) => {
          if (seq !== previewSeqRef.current) return;
          setPreviewResult(res);
        })
        .catch((err: unknown) => {
          if (seq !== previewSeqRef.current) return;
          setPreviewResult(null);
          setPreviewError(err instanceof Error ? err.message : "模板预览失败");
        })
        .finally(() => {
          if (seq === previewSeqRef.current) setPreviewLoading(false);
        });
    }, 300);
    return () => window.clearTimeout(timer);
  }, [templateFiring, templateResolved, previewStatus, previewProjectID, previewRawPayloadJSON]);

  function insertTemplateToken(token: string) {
    const fieldName = editingTarget === "resolved" ? "template_resolved" : "template_firing";
    const currentValue = String(form.getFieldValue(fieldName) || "");
    const inputRef = editingTarget === "resolved" ? resolvedTemplateRef?.current : firingTemplateRef?.current;
    const textarea = (inputRef as { resizableTextArea?: { textArea?: HTMLTextAreaElement } })?.resizableTextArea?.textArea;
    if (textarea && Number.isFinite(textarea.selectionStart) && Number.isFinite(textarea.selectionEnd)) {
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      const next = `${currentValue.slice(0, start)}${token}${currentValue.slice(end)}`;
      form.setFieldValue(fieldName, next);
      window.setTimeout(() => {
        textarea.focus();
        const pos = start + token.length;
        textarea.setSelectionRange(pos, pos);
      }, 0);
      return;
    }
    const sep = currentValue.trim() ? "\n" : "";
    form.setFieldValue(fieldName, `${currentValue}${sep}${token}`);
  }

  function applyTemplatePreset(target: TemplatePreviewStatus, presetMode: string) {
    const tpl = presetTemplateByMode(presetMode);
    if (target === "firing") {
      onFiringPresetChange(presetMode);
      form.setFieldValue("template_firing", tpl.firing);
      return;
    }
    onResolvedPresetChange(presetMode);
    form.setFieldValue("template_resolved", tpl.resolved);
  }

  const projectOptions = projects.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id }));
  const availableFields = previewResult?.combined_fields ?? [];
  const fixedFields = previewResult?.available_fields ?? [];
  const rawPayloadFields = previewResult?.raw_payload_fields ?? [];
  const suggestedLabelKeys = previewResult?.suggested_label_keys ?? [];
  const templateVariables: AlertTemplateVariableDoc[] = previewResult?.template_variables ?? [];

  return (
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
      <Row gutter={16}>
        <Col xs={24} lg={12}>
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Space wrap>
              <Form.Item name="preview_status" label="预览类型" initialValue="firing" style={{ marginBottom: 0 }}>
                <Segmented
                  options={[
                    { label: "触发预览", value: "firing" },
                    { label: "恢复预览", value: "resolved" },
                  ]}
                />
              </Form.Item>
              <Form.Item name="preview_project_id" label="项目上下文" style={{ marginBottom: 0, minWidth: 200 }}>
                <Select allowClear options={projectOptions} placeholder="默认示例项目" style={{ width: 220 }} />
              </Form.Item>
            </Space>

            <Form.Item label="编辑目标">
              <Select
                value={editingTarget}
                onChange={(v: TemplatePreviewStatus) => onEditingTargetChange(v)}
                options={[
                  { label: "触发模板", value: "firing" },
                  { label: "恢复模板", value: "resolved" },
                ]}
              />
            </Form.Item>

            <Form.Item
              name="template_firing"
              label="告警触发模板"
              extra="Go text/template；留空使用系统默认。事件台深链 {{.EventPath}}"
            >
              <Input.TextArea ref={firingTemplateRef as never} rows={8} placeholder="{{.Title}}" />
            </Form.Item>
            <Form.Item label="触发模板预设">
              <Select value={firingPreset} options={[...CHANNEL_PRESET_OPTIONS]} onChange={(v) => applyTemplatePreset("firing", v)} />
            </Form.Item>

            <Form.Item name="template_resolved" label="告警恢复模板">
              <Input.TextArea ref={resolvedTemplateRef as never} rows={8} placeholder="{{.StartsAt}} ~ {{.EndsAt}}" />
            </Form.Item>
            <Form.Item label="恢复模板预设">
              <Select value={resolvedPreset} options={[...CHANNEL_PRESET_OPTIONS]} onChange={(v) => applyTemplatePreset("resolved", v)} />
            </Form.Item>

            <Form.Item
              name="preview_raw_payload_json"
              label="预览原始 JSON（可选）"
              extra="与示例 payload 合并；同名字段以 JSON 为准"
              rules={[
                {
                  validator: async (_, value) => {
                    const s = String(value || "").trim();
                    if (!s) return;
                    try {
                      const obj = JSON.parse(s);
                      if (!obj || typeof obj !== "object" || Array.isArray(obj)) throw new Error("必须是对象");
                    } catch {
                      throw new Error("JSON 格式不正确");
                    }
                  },
                },
              ]}
            >
              <Input.TextArea rows={4} placeholder='{"labels":{"namespace":"prod"},"current":"9"}' />
            </Form.Item>
          </Space>
        </Col>

        <Col xs={24} lg={12}>
          <NotificationPreviewBubble
            rendered={previewResult?.rendered || ""}
            status={previewStatus}
            loading={previewLoading}
            error={previewError}
            channelType={channelType}
          />
          <Form.Item label="纯文本输出" style={{ marginTop: 12 }}>
            <Input.TextArea rows={6} value={previewResult?.rendered || ""} readOnly />
          </Form.Item>
        </Col>
      </Row>

      <Collapse
        size="small"
        items={[
          {
            key: "fields",
            label: "变量与字段（点击插入）",
            children: (
              <Space direction="vertical" style={{ width: "100%" }} size="middle">
                <div>
                  <Typography.Text type="secondary">组合字段</Typography.Text>
                  <div style={{ marginTop: 8 }}>
                    <Space wrap>
                      {availableFields.map((v) => (
                        <Tag key={v} color="blue" style={{ cursor: "pointer" }} onClick={() => insertTemplateToken(`{{.${v}}}`)}>
                          {v}
                        </Tag>
                      ))}
                    </Space>
                  </div>
                </div>
                <div>
                  <Typography.Text type="secondary">固定字段</Typography.Text>
                  <div style={{ marginTop: 8 }}>
                    <Space wrap>
                      {fixedFields.map((v) => (
                        <Tag key={v}>{v}</Tag>
                      ))}
                    </Space>
                  </div>
                </div>
                <div>
                  <Typography.Text type="secondary">标签键（近期告警）</Typography.Text>
                  <div style={{ marginTop: 8 }}>
                    <Space wrap>
                      {suggestedLabelKeys.map((v) => (
                        <Tag
                          key={v}
                          color="purple"
                          style={{ cursor: "pointer" }}
                          onClick={() => insertTemplateToken(`{{index .Labels "${v}"}}`)}
                        >
                          {v}
                        </Tag>
                      ))}
                    </Space>
                  </div>
                </div>
                <div>
                  <Typography.Text type="secondary">原始 JSON 顶层字段</Typography.Text>
                  <div style={{ marginTop: 8 }}>
                    <Space wrap>
                      {rawPayloadFields.length === 0 ? <Tag>（暂无）</Tag> : rawPayloadFields.map((v) => <Tag key={v}>{v}</Tag>)}
                    </Space>
                  </div>
                </div>
                <Table
                  size="small"
                  pagination={false}
                  rowKey="name"
                  dataSource={templateVariables}
                  columns={[
                    {
                      title: "变量",
                      dataIndex: "name",
                      width: 200,
                      render: (v: string) => (
                        <Typography.Text code style={{ cursor: "pointer" }} onClick={() => insertTemplateToken(`{{.${v}}}`)}>
                          {"{{." + v + "}}"}
                        </Typography.Text>
                      ),
                    },
                    { title: "说明", dataIndex: "description" },
                  ]}
                />
              </Space>
            ),
          },
          {
            key: "payload",
            label: "渲染上下文 sample_payload",
            children: (
              <Input.TextArea rows={10} readOnly value={JSON.stringify(previewResult?.sample_payload || {}, null, 2)} />
            ),
          },
        ]}
      />
    </Space>
  );
}
