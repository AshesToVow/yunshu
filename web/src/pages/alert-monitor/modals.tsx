// @ts-nocheck
import {
  AutoComplete,
  Button,
  Card,
  DatePicker,
  Drawer,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  TreeSelect,
  Typography,
  message,
} from "antd";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import type { MetricLabelFilter } from "./platform-provider-types";
import { useAlertMonitor } from "./context";

type QuickSilenceTarget = {
  key: string;
  name: string;
  labels: Record<string, string>;
  startsAt: import("dayjs").Dayjs;
  endsAt: import("dayjs").Dayjs;
};

export function AlertMonitorModals() {
  const c = useAlertMonitor();
  return (
    <>
      <Drawer
        title={c.dsCurrent ? "编辑数据源" : "新建数据源"}
        placement="right"
        width={640}
        open={c.dsModalOpen}
        onClose={() => c.setDsModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => c.setDsModalOpen(false)}>取消</Button>
            <Button type="primary" loading={c.dsSubmitting} onClick={() => void c.submitDs()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={c.dsForm} layout="vertical" autoComplete="off">
          <Form.Item name="project_id" label="所属项目" rules={[{ required: true, message: "请选择项目" }]}>
            <Select options={c.projectOptions} placeholder="请选择项目" />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="type"
            label="类型"
            rules={[{ required: true, message: "请选择数据源类型" }]}
            extra={
              <Typography.Text type="secondary">
                VictoriaMetrics 使用与 Prometheus 兼容的 HTTP API（/api/v1/query）。
              </Typography.Text>
            }
          >
            <Select
              options={[
                { value: "prometheus", label: "Prometheus" },
                { value: "victoria", label: "VictoriaMetrics" },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="base_url"
            label="Base URL"
            rules={[{ required: true, message: "请输入或从下拉选择 Base URL" }]}
            extra={
              <Typography.Text type="secondary">
                可直接输入；亦可从下拉选字典项（类型 <Typography.Text code>alert_datasource_base_url</Typography.Text>，「值」存完整 URL）。
              </Typography.Text>
            }
          >
            <AutoComplete
              style={{ width: "100%" }}
              allowClear
              placeholder="输入 URL 或点击选择字典项"
              options={c.dsUrlAutoOpts}
              filterOption={(input, option) =>
                (option?.label ?? "").toString().toLowerCase().includes(input.toLowerCase()) ||
                (option?.value ?? "").toString().toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="bearer_token" label="Bearer Token">
            <Input.Password placeholder="留空表示不改" autoComplete="new-password" />
          </Form.Item>
          <Form.Item
            name="basic_user"
            label="Basic 用户"
            extra={
              <Typography.Text type="secondary">
                可直接输入；亦可从下拉选字典项（<Typography.Text code>alert_datasource_basic_user</Typography.Text>）；密码勿入字典。
              </Typography.Text>
            }
          >
            <AutoComplete
              style={{ width: "100%" }}
              allowClear
              placeholder="输入用户名或从字典选择"
              options={c.dsBasicUserAutoOpts}
              filterOption={(input, option) =>
                (option?.label ?? "").toString().toLowerCase().includes(input.toLowerCase()) ||
                (option?.value ?? "").toString().toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="basic_password" label="Basic 密码">
            <Input.Password placeholder="留空表示不改" autoComplete="new-password" />
          </Form.Item>
          <Form.Item name="skip_tls_verify" label="跳过 TLS 校验" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title={c.silCurrent ? "编辑静默" : "新建静默"}
        placement="right"
        width={720}
        open={c.silModalOpen}
        onClose={() => c.setSilModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => c.setSilModalOpen(false)}>取消</Button>
            <Button type="primary" loading={c.silSubmitting} onClick={() => void c.submitSil()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={c.silForm} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
            匹配器：名称通常选 <Typography.Text code>alertname</Typography.Text> / <Typography.Text code>cluster</Typography.Text> 等；值支持精确匹配；勾选「正则」时按 Alertmanager matcher 语义使用正则。
          </Typography.Paragraph>
          <Form.List name="matchers">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field) => (
                  <Space key={field.key} align="baseline" style={{ display: "flex", marginBottom: 8 }} wrap>
                    <Form.Item
                      name={[field.name, "name"]}
                      rules={[{ required: true, message: "填写 label 名称" }]}
                      style={{ marginBottom: 0, minWidth: 200 }}
                    >
                      <AutoComplete
                        allowClear
                        placeholder="label 名（可输入或选字典）"
                        options={c.silenceMatcherNameOptions}
                        filterOption={(input, option) =>
                          (option?.label ?? "").toString().toLowerCase().includes(input.toLowerCase()) ||
                          (option?.value ?? "").toString().toLowerCase().includes(input.toLowerCase())
                        }
                      />
                    </Form.Item>
                    <Form.Item name={[field.name, "value"]} style={{ marginBottom: 0, flex: 1, minWidth: 160 }}>
                      <Input placeholder="匹配值" />
                    </Form.Item>
                    <Form.Item
                      name={[field.name, "is_regex"]}
                      valuePropName="checked"
                      initialValue={false}
                      style={{ marginBottom: 0 }}
                    >
                      <Switch checkedChildren="正则" unCheckedChildren="精确" />
                    </Form.Item>
                    <MinusCircleOutlined onClick={() => remove(field.name)} />
                  </Space>
                ))}
                <Form.Item>
                  <Button type="dashed" onClick={() => add({ name: "alertname", value: "", is_regex: false })} block icon={<PlusOutlined />}>
                    添加匹配条件
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>
          <Form.Item name="starts_at" label="开始时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="ends_at" label="结束时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="comment" label="说明">
            <Input />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title="批量静默（可分别设置起止时间）"
        placement="right"
        width={1000}
        open={c.quickSilenceOpen}
        onClose={() => c.setQuickSilenceOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => c.setQuickSilenceOpen(false)}>取消</Button>
            <Button type="primary" loading={c.quickSilenceSubmitting} onClick={() => void c.submitQuickSilence()}>
              确定
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
          静默说明（可选）：将写入每条静默记录的 <Typography.Text code>comment</Typography.Text> 字段，便于审计。
        </Typography.Paragraph>
        <Input.TextArea
          rows={2}
          value={c.quickSilenceComment}
          onChange={(e) => c.setQuickSilenceComment(e.target.value)}
          placeholder="例如：发布窗口临时静默、误报告警排查中…"
          maxLength={512}
          showCount
          style={{ marginBottom: 12 }}
        />
        <Table
          rowKey="key"
          size="small"
          pagination={false}
          dataSource={c.quickSilenceTargets}
          scroll={{ x: 920 }}
          columns={[
            { title: "名称", dataIndex: "name", width: 200 },
            {
              title: "匹配器摘要",
              width: 360,
              ellipsis: true,
              render: (_: unknown, r: QuickSilenceTarget) =>
                Object.entries(r.labels || {})
                  .map(([k, v]) => `${k}=${v}`)
                  .join(", "),
            },
            {
              title: "开始",
              width: 170,
              render: (_: unknown, r: QuickSilenceTarget) => (
                <DatePicker
                  showTime
                  value={r.startsAt}
                  onChange={(v) =>
                    c.setQuickSilenceTargets((prev) => prev.map((it) => (it.key === r.key ? { ...it, startsAt: v ?? it.startsAt } : it)))
                  }
                />
              ),
            },
            {
              title: "结束",
              width: 170,
              render: (_: unknown, r: QuickSilenceTarget) => (
                <DatePicker
                  showTime
                  value={r.endsAt}
                  onChange={(v) =>
                    c.setQuickSilenceTargets((prev) => prev.map((it) => (it.key === r.key ? { ...it, endsAt: v ?? it.endsAt } : it)))
                  }
                />
              ),
            },
          ]}
        />
      </Drawer>

      <Drawer
        title={c.cloudExpiryCurrent ? "编辑云到期规则" : "新建云到期规则"}
        placement="right"
        width={700}
        open={c.cloudExpiryModalOpen}
        onClose={() => c.setCloudExpiryModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => c.setCloudExpiryModalOpen(false)}>取消</Button>
            <Button type="primary" loading={c.cloudExpirySubmitting} onClick={() => void c.submitCloudExpiryRule()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={c.cloudExpiryForm} layout="vertical">
          <Form.Item name="project_id" label="项目" rules={[{ required: true, message: "请选择项目" }]}>
            <Select options={c.projectOptions} placeholder="选择项目" />
          </Form.Item>
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: "请输入规则名称" }]}>
            <Input placeholder="例如：核心生产云资源到期提醒" />
          </Form.Item>
          <Form.Item name="provider" label="云厂商">
            <Select
              allowClear
              placeholder="全部厂商"
              options={[
                { label: "全部", value: "" },
                { label: "阿里云", value: "alibaba" },
                { label: "腾讯云", value: "tencent" },
                { label: "京东云", value: "jd" },
              ]}
            />
          </Form.Item>
          <Form.Item
            name="region_scope"
            label="地域范围"
            extra="多个地域用英文逗号分隔；留空表示全部。腾讯云请填 API 地域（如 ap-guangzhou），也可填「广州」等常见中文名。"
          >
            <Input placeholder="多个地域用英文逗号分隔；留空表示全部" />
          </Form.Item>
          <Form.Item name="advance_days" label="提前告警天数" rules={[{ required: true, message: "请输入提前天数" }]}>
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="severity" label="告警级别" rules={[{ required: true, message: "请选择级别" }]}>
            <Select options={c.alertSeverityOpts} />
          </Form.Item>
          <Form.Item
            name="eval_cron_spec"
            dependencies={["schedule_enabled"]}
            label="Cron 表达式"
            rules={[
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (getFieldValue("schedule_enabled") !== false && !String(value ?? "").trim()) {
                    return Promise.reject(new Error("已启用定时评估时必须填写 Cron"));
                  }
                  return Promise.resolve();
                },
              }),
            ]}
            extra={
              <span>
                启用「定时自动评估」时<strong>必填</strong>。robfig/cron：<strong>五段</strong>为「分 时 日 月 周」；亦支持<strong>六段</strong>「秒 分 时 日 月 周」及 <code>@every 1m</code> 等描述符。服务约每 5 秒检查一次是否到点，故 Cron 粒度不宜低于该量级。
                <br />
                示例：<code>*/1 * * * *</code> 每分钟；<code>0 * * * *</code> 每小时整点；<code>0 */2 * * *</code> 每 2 小时整点（注意五段里首位是<strong>分</strong>，勿写成 <code>*/2 * * * *</code>，否则表示「每 2 分钟」）；<code>0 9 * * *</code> 每天 9:00。
              </span>
            }
          >
            <Input allowClear placeholder="例如 0 9 * * * 或 @every 1h" />
          </Form.Item>
          <Form.Item
            name="schedule_enabled"
            label="启用定时自动评估"
            valuePropName="checked"
            extra="开启后由告警服务按上方 Cron 拉云 API；有 Redis 时写入上次评估时间；关闭则仅「立即执行一次评估」会拉云。"
          >
            <Switch />
          </Form.Item>
          <Form.Item
            name="labels_json"
            label="附加 Labels(JSON)"
            rules={[
              {
                validator: async (_, value) => {
                  const normalized = c.normalizeCloudExpiryLabelsJSON(String(value || ""));
                  if (normalized === null) {
                    throw new Error("labels_json 必须是合法 JSON 对象，例如 {\"biz\":\"payments\"}");
                  }
                },
              },
            ]}
            extra="支持实时 JSON 语法校验；仅允许 JSON 对象。"
          >
            <Input.TextArea rows={4} placeholder='例如：{"biz":"payments","env":"prod"}' />
          </Form.Item>
          <Form.Item>
            <Button
              onClick={() => {
                const raw = String(c.cloudExpiryForm.getFieldValue("labels_json") || "");
                const normalized = c.normalizeCloudExpiryLabelsJSON(raw);
                if (normalized === null) {
                  message.error("JSON 格式错误，无法格式化");
                  return;
                }
                c.cloudExpiryForm.setFieldValue("labels_json", normalized);
                message.success("已格式化 JSON");
              }}
            >
              格式化 JSON
            </Button>
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title={c.ruleCurrent ? "编辑监控规则" : "新建监控规则"}
        placement="right"
        width={920}
        open={c.ruleModalOpen}
        onClose={() => c.setRuleModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => c.setRuleModalOpen(false)}>取消</Button>
            <Button type="primary" loading={c.ruleSubmitting} onClick={() => void c.submitRule()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={c.ruleForm} layout="vertical">
          <Form.Item name="datasource_id" label="数据源" rules={[{ required: true }]}>
            <Select
              options={(c.projectContextId ? c.dsList.filter((d) => d.project_id === c.projectContextId) : c.dsList).map((d) => ({
                label: d.project_name ? `${d.project_name} / ${d.name}` : d.name,
                value: d.id,
              }))}
            />
          </Form.Item>
          <Form.Item name="name" label="规则名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Card size="small" title="PromQL 辅助生成（推荐）" style={{ marginBottom: 12 }}>
            <Space direction="vertical" style={{ width: "100%" }} size={8}>
              <Space wrap>
                <Input
                  style={{ width: 260 }}
                  value={c.metricKeyword}
                  onChange={(e) => c.setMetricKeyword(e.target.value)}
                  placeholder="按指标名关键字检索，如 cpu/memory/http"
                />
                <Button loading={c.metricLoading} onClick={() => void c.loadMetricOptionsForRule()}>
                  拉取指标
                </Button>
                <Select
                  showSearch
                  style={{ minWidth: 320 }}
                  placeholder="选择指标名"
                  value={c.selectedMetric || undefined}
                  options={c.metricOptions.map((m) => ({ label: m, value: m }))}
                  onChange={(v) => c.setSelectedMetric(String(v || ""))}
                  filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
                />
              </Space>
              {c.metricLabelFilters.map((f, idx) => (
                <Space key={`metric-filter-${idx}`} wrap>
                  <Select
                    mode="tags"
                    style={{ width: 180 }}
                    value={f.key ? [f.key] : []}
                    placeholder="标签名"
                    options={c.commonLabelKeyOptions}
                    onChange={(v) =>
                      c.setMetricLabelFilters((prev) =>
                        prev.map((it, i) => {
                          if (i !== idx) return it;
                          const val = Array.isArray(v) ? String(v[0] || "") : "";
                          return { ...it, key: val };
                        }),
                      )
                    }
                  />
                  <Select
                    style={{ width: 110 }}
                    value={f.op}
                    options={[
                      { label: "等于 (=)", value: "=" },
                      { label: "不等于 (!=)", value: "!=" },
                      { label: "正则 (=~)", value: "=~" },
                      { label: "反正则 (!~)", value: "!~" },
                    ]}
                    onChange={(v) =>
                      c.setMetricLabelFilters((prev) => prev.map((it, i) => (i === idx ? { ...it, op: v as MetricLabelFilter["op"] } : it)))
                    }
                  />
                  <AutoComplete
                    style={{ width: 260 }}
                    value={f.value}
                    options={c.labelValueOptions.map((v) => ({ value: v }))}
                    onChange={(v) =>
                      c.setMetricLabelFilters((prev) => prev.map((it, i) => (i === idx ? { ...it, value: String(v || "") } : it)))
                    }
                    placeholder="标签值，可手填或拉取候选"
                  />
                  <Button loading={c.labelValueLoading} onClick={() => void c.loadLabelValuesForRule(idx)}>
                    拉取值
                  </Button>
                  <Button
                    danger
                    disabled={c.metricLabelFilters.length <= 1}
                    onClick={() => c.setMetricLabelFilters((prev) => prev.filter((_, i) => i !== idx))}
                  >
                    删除
                  </Button>
                </Space>
              ))}
              <Space wrap>
                <Button onClick={() => c.setMetricLabelFilters((prev) => [...prev, { key: "", op: "=", value: "" }])}>新增标签过滤</Button>
                <Button type="primary" onClick={c.applyMetricSelectorToRuleExpr}>
                  生成并带入 PromQL
                </Button>
              </Space>
              <Typography.Text type="secondary">
                先选指标，再按标签过滤，最后一键带入到上方 PromQL；不会覆盖你后续用“条件构建器”生成的比较表达式。
              </Typography.Text>
              <Card size="small" title="Prometheus 函数助手（内置）">
                <Space direction="vertical" style={{ width: "100%" }} size={8}>
                  <Space wrap>
                    <Select
                      style={{ minWidth: 280 }}
                      value={c.selectedPromFunc}
                      options={c.promFunctionTemplates.map((it) => ({ label: it.label, value: it.key }))}
                      onChange={(v) => c.setSelectedPromFunc(String(v || "rate"))}
                    />
                    <Button onClick={c.insertPromFunctionToExpr}>插入到 PromQL</Button>
                    <Button onClick={c.usePromFunctionAsConditionMetric}>带入条件构造器首条指标</Button>
                  </Space>
                  <Typography.Text type="secondary">
                    {c.selectedPromFuncMeta.desc}
                    <br />
                    模板：<Typography.Text code>{c.selectedPromFuncMeta.template}</Typography.Text>
                    <br />
                    推荐顺序：第1步标签过滤 {"->"} 第2步函数（可选） {"->"} 第3步阈值比较。
                  </Typography.Text>
                </Space>
              </Card>
            </Space>
          </Card>
          <Form.Item name="expr" label="PromQL" rules={[{ required: true }]}>
            <Input.TextArea rows={4} />
          </Form.Item>
          <Card size="small" title="条件构建器（可选）" style={{ marginBottom: 12 }}>
            <Space direction="vertical" style={{ width: "100%" }} size={8}>
              <Space wrap>
                <Typography.Text type="secondary">组合逻辑</Typography.Text>
                <Select style={{ width: 180 }} value={c.ruleLogic} options={c.ruleLogicOptions} onChange={(v) => c.setRuleLogic(v as RuleBuilderLogic)} />
              </Space>
              {c.ruleConditions.map((cond, idx) => (
                <Space key={`rule-cond-${idx}`} wrap style={{ width: "100%" }}>
                  <Input
                    style={{ minWidth: 320 }}
                    value={cond.metric}
                    onChange={(e) =>
                      c.setRuleConditions((prev) => prev.map((it, i) => (i === idx ? { ...it, metric: e.target.value } : it)))
                    }
                    placeholder="指标表达式，如 rate(http_requests_total[5m])"
                  />
                  <Select
                    style={{ width: 160 }}
                    value={cond.comparator}
                    options={c.ruleComparatorOptions}
                    onChange={(v) =>
                      c.setRuleConditions((prev) => prev.map((it, i) => (i === idx ? { ...it, comparator: v as RuleComparator } : it)))
                    }
                  />
                  <InputNumber
                    style={{ width: 160 }}
                    value={cond.threshold}
                    onChange={(v) =>
                      c.setRuleConditions((prev) => prev.map((it, i) => (i === idx ? { ...it, threshold: v ?? null } : it)))
                    }
                    placeholder="阈值"
                  />
                  <Tag>{c.thresholdUnit || "raw"}</Tag>
                  <Button
                    danger
                    disabled={c.ruleConditions.length <= 1}
                    onClick={() => c.setRuleConditions((prev) => prev.filter((_, i) => i !== idx))}
                  >
                    删除条件
                  </Button>
                </Space>
              ))}
              <Space wrap>
                <Button onClick={() => c.setRuleConditions((prev) => [...prev, { metric: "", comparator: ">", threshold: null }])}>新增条件</Button>
                <Button type="primary" onClick={c.applyRuleBuilderToExpr}>
                  生成 PromQL
                </Button>
                <Button onClick={c.applyStepwisePromQL}>按步骤一键生成（推荐）</Button>
              </Space>
            </Space>
          </Card>
          <Form.Item name="for_seconds" label="持续满足秒数 (for)">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="threshold_unit" label="阈值单位">
            <Select options={c.thresholdUnitOptions} />
          </Form.Item>
          <Form.Item name="eval_interval_seconds" label="评估间隔秒">
            <InputNumber min={5} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="severity" label="级别" rules={[{ required: true, message: "请选择级别" }]}>
            <Select placeholder="选择级别" options={c.ruleSeverityOptions} />
          </Form.Item>
          <Form.Item
            name="labels_json"
            label="附加 Labels（JSON）"
            rules={[
              {
                validator: async (_, value) => {
                  const normalized = c.normalizeCloudExpiryLabelsJSON(String(value ?? ""));
                  if (normalized === null) {
                    throw new Error("须为合法 JSON 对象，例如 {\"route\":\"prod-warning-email\",\"biz\":\"core\"}");
                  }
                },
              },
            ]}
            extra={
              "写入后会与 PromQL 样本标签等合并到告警 labels；订阅树节点可按 match_labels_json 分流。可与数据源侧 Prometheus 规则告警共用同一套标签维度（如 severity、cluster、route）。勿在此处填写 alertname / datasource_id，规则会自动填充。"
            }
          >
            <Input.TextArea rows={4} placeholder='{"route":"prod-critical-all"}' />
          </Form.Item>
          <Form.Item>
            <Button
              type="link"
              style={{ paddingLeft: 0 }}
              onClick={() => {
                const raw = String(c.ruleForm.getFieldValue("labels_json") || "");
                const normalized = c.normalizeCloudExpiryLabelsJSON(raw);
                if (normalized === null) {
                  message.error("JSON 格式错误，无法格式化");
                  return;
                }
                c.ruleForm.setFieldValue("labels_json", normalized);
                message.success("已格式化 JSON");
              }}
            >
              格式化 labels JSON
            </Button>
          </Form.Item>
          <Form.Item
            label="告警文案预设（新手推荐）"
            name="rule_template_preset"
            extra="选择后自动填充 summary/description 模板，可继续手工修改；编辑规则时会自动回显匹配到的预设。"
          >
            <Select
              placeholder="选择一个预设模板"
              options={c.ruleTemplatePresetOptions}
              onChange={(v) => c.applyRuleAnnotationPreset(String(v || "generic"))}
            />
          </Form.Item>
          <Form.Item
            name="summary_template"
            label="告警摘要模板（summary）"
            extra='支持占位符：{{$labels.xxx}}、{{$value}}、{{.RuleName}}、{{.Expr}}'
            rules={[{ required: true, message: "请填写 summary 模板" }]}
          >
            <Input.TextArea rows={2} placeholder="{{$labels.instance}}: {{.RuleName}} 告警触发，当前值 {{$value}}" />
          </Form.Item>
          <Form.Item
            name="description_template"
            label="告警描述模板（description）"
            extra='支持占位符：{{$labels.xxx}}、{{$value}}、{{.RuleName}}、{{.Expr}}'
            rules={[{ required: true, message: "请填写 description 模板" }]}
          >
            <Input.TextArea rows={3} placeholder="规则 {{.RuleName}} 触发，PromQL={{.Expr}}，实例={{$labels.instance}}，当前值={{$value}}" />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title="规则处理人"
        placement="right"
        width={640}
        open={c.assignOpen}
        onClose={() => c.setAssignOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => c.setAssignOpen(false)}>取消</Button>
            <Button type="primary" loading={c.assignSubmitting} onClick={() => void c.submitAssign()}>
              保存
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary">
          邮件仅发往「用户」中勾选的人员与下方「邮箱」字段（不含部门子树展开）。部门用于钉钉/企微 @（含子树 ∩ 项目成员）。钉钉/企微手机号无法在企业内解析时会补发邮件；wechat 等也会补发邮件。
        </Typography.Paragraph>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 12 }}>
          部门需<strong>手动选择</strong>，保存后按库中配置加载，不会随「用户」自动回填。部门仅用于钉钉/企微 @（项目成员 ∩ 子树 + 部门负责人）；邮件只发上方勾选用户。
        </Typography.Paragraph>
        <Form form={c.assignForm} layout="vertical">
          <Form.Item name="user_ids" label="用户">
            <Select mode="multiple" options={c.users} optionFilterProp="label" placeholder="选择用户" />
          </Form.Item>
          {c.assignUsersHint ? (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              用户资料邮箱：{c.assignUsersHint}
            </Typography.Paragraph>
          ) : null}
          <Form.Item
            name="department_ids"
            label="部门（子树）"
            extra="可选。用于 IM @，不参与邮件收件人。清空后保存即可生效。"
          >
            <TreeSelect treeData={c.deptTree} treeCheckable showSearch allowClear treeDefaultExpandAll style={{ width: "100%" }} placeholder="可选，用于钉钉/企微 @" />
          </Form.Item>
          {c.assignUserIds?.length === 1 ? (
            <Form.Item name="profile_email" label="邮箱（可改，保存时写回该用户资料）">
              <Input placeholder="无邮箱时请填写，保存后写入用户表" />
            </Form.Item>
          ) : (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              多人时邮件按各用户资料邮箱合并；仅选择一名用户时可在此编辑邮箱并写回用户资料。
            </Typography.Paragraph>
          )}
          <Form.Item name="notify_on_resolved" label="恢复时通知" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title="规则值班（按时间段生效）"
        placement="right"
        width={720}
        open={c.dutyModalOpen}
        onClose={() => c.setDutyModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Button type="primary" onClick={() => c.setDutyModalOpen(false)}>
            关闭
          </Button>
        }
      >
        <Space direction="vertical" style={{ width: "100%" }} size="small">
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            当前规则 ID：{c.dutyRuleId ?? "-"}。班次命中时会与“处理人”邮箱合并去重后写入 <Typography.Text code>assignee_emails</Typography.Text>，并优先于邮件通道固定收件人。
          </Typography.Paragraph>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
            若其他规则上已配好相同时间段与值班人，可从该规则「复制班次」到本规则（会新增独立记录，两条规则各自生效、互不影响）。
          </Typography.Paragraph>
          <Space wrap align="start">
            <Select
              allowClear
              showSearch
              placeholder="选择已有班次的来源规则"
              style={{ minWidth: 280 }}
              options={c.copyDutyRuleOptions}
              value={c.copySourceRuleId}
              onChange={(v) => c.setCopySourceRuleId(v)}
              optionFilterProp="label"
              disabled={!c.dutyRuleId || c.copyDutyRuleOptions.length === 0}
            />
            <Button
              loading={c.copyDutyLoading}
              disabled={!c.dutyRuleId || !c.copySourceRuleId}
              onClick={() => void c.copyDutyBlocksFromSelectedRule()}
            >
              复制班次到当前规则
            </Button>
          </Space>
          <Button type="primary" icon={<PlusOutlined />} disabled={!c.dutyRuleId} onClick={c.openBlkCreate}>
            新建班次
          </Button>
          <Table rowKey="id" columns={c.blkColumns} dataSource={c.blockList} pagination={false} size="small" scroll={{ x: 800 }} />
        </Space>
      </Drawer>

      <Drawer
        title={c.blkCurrent ? "编辑班次" : "新建班次"}
        placement="right"
        width={640}
        open={c.blkModalOpen}
        onClose={() => c.setBlkModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => c.setBlkModalOpen(false)}>取消</Button>
            <Button type="primary" loading={c.blkSubmitting} onClick={() => void c.submitBlk()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={c.blkForm} layout="vertical">
          <Form.Item name="monitor_rule_id" hidden>
            <InputNumber />
          </Form.Item>
          <Form.Item name="range" label="起止时间" rules={[{ required: true }]}>
            <DatePicker.RangePicker showTime={{ format: "HH:mm" }} format="YYYY-MM-DD HH:mm" style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="title" label="标题">
            <Input />
          </Form.Item>
          <Form.Item name="user_ids" label="用户">
            <Select mode="multiple" options={c.users} optionFilterProp="label" placeholder="选择值班人员" />
          </Form.Item>
          {c.dutyUsersHint ? (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              用户资料邮箱：{c.dutyUsersHint}
            </Typography.Paragraph>
          ) : null}
          <Form.Item
            name="department_ids"
            label="部门（子树）"
            extra="当前班次解析仍为部门子树内用户（未按项目成员过滤）；与显式「用户」合并。若需项目交集与负责人，请用规则「处理人」侧配置。"
          >
            <TreeSelect treeData={c.deptTree} treeCheckable showSearch allowClear treeDefaultExpandAll style={{ width: "100%" }} placeholder="随用户带出，可改" />
          </Form.Item>
          {c.blkUserIds?.length === 1 ? (
            <Form.Item name="profile_email" label="邮箱（可改，保存班次时写回该用户资料）">
              <Input placeholder="无邮箱时请填写，保存后写入用户表" />
            </Form.Item>
          ) : (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              多人值班时通知仍按各用户资料邮箱合并；仅选择一名用户时可在此编辑邮箱并写回用户资料。
            </Typography.Paragraph>
          )}
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Drawer>
    </>
  );
}
