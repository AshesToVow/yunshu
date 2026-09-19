import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Col, Popconfirm, Progress, Row, Select, Space, Table, Tag, Typography, message } from "antd";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import { createInhibitionRule } from "../services/alert-inhibition";
import { createAlertSilence } from "../services/alert-platform";
import { getAlertQualityReport, type AlertQualityReport } from "../services/alert-quality";
import { extractApiErrorMessage } from "../services/http";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

const WINDOW_OPTIONS = [
  { value: 24, label: "近 24 小时" },
  { value: 72, label: "近 3 天" },
  { value: 168, label: "近 7 天" },
];

type NoiseRow = AlertQualityReport["noise_top"][number];
type RepeatRow = AlertQualityReport["repeat_fingerprints"][number];

type Props = {
  /** 嵌入告警平台时隐藏独立项目选择，跟随顶栏上下文 */
  embedded?: boolean;
  projectContextId?: number;
};

export function AlertQualityPage({ embedded, projectContextId }: Props) {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number | undefined>(projectContextId);
  const [windowHours, setWindowHours] = useState(24);
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<AlertQualityReport | null>(null);
  const [actionKey, setActionKey] = useState("");

  const effectiveProjectId = embedded ? projectContextId : projectId;

  const projectOptions = useMemo(
    () => [{ value: 0, label: "全部项目" }, ...projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` }))],
    [projects],
  );

  useEffect(() => {
    if (embedded) return;
    void (async () => {
      const p = await getProjects({ page: 1, page_size: 1000 });
      setProjects(p.list);
    })();
  }, [embedded]);

  useEffect(() => {
    if (embedded) {
      setProjectId(projectContextId);
    }
  }, [embedded, projectContextId]);

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [effectiveProjectId, windowHours]);

  async function load() {
    setLoading(true);
    try {
      const r = await getAlertQualityReport({
        window_hours: windowHours,
        project_id: effectiveProjectId || undefined,
      });
      setReport(r);
    } finally {
      setLoading(false);
    }
  }

  async function silence2h(row: { fingerprint?: string; alertname?: string; title?: string }, key: string) {
    if (!row.fingerprint) {
      message.warning("缺少 fingerprint，无法静默");
      return;
    }
    const matchers: Array<{ name: string; value: string; is_regex: boolean }> = [
      { name: "fingerprint", value: row.fingerprint, is_regex: false },
    ];
    const alertname = row.alertname || row.title;
    if (alertname) matchers.push({ name: "alertname", value: alertname, is_regex: false });
    setActionKey(key);
    try {
      await createAlertSilence({
        name: `静默 ${alertname || row.fingerprint}（2 小时）`,
        matchers_json: JSON.stringify(matchers),
        comment: "质量治理一键静默 2h",
        enabled: true,
        starts_at: dayjs().toISOString(),
        ends_at: dayjs().add(2, "hour").toISOString(),
        project_id: effectiveProjectId && effectiveProjectId > 0 ? effectiveProjectId : undefined,
      });
      message.success("已创建 2 小时静默");
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建静默失败"));
    } finally {
      setActionKey("");
    }
  }

  async function createInhibitionDraft(row: { fingerprint?: string; alertname?: string; title?: string }, key: string) {
    const title = row.title || row.alertname || row.fingerprint || "unknown";
    const labels: Record<string, string> = {};
    if (row.alertname) labels.alertname = row.alertname;
    else if (row.fingerprint) labels.fingerprint = row.fingerprint;
    else if (row.title) labels.alertname = row.title;
    if (!Object.keys(labels).length) {
      message.warning("缺少匹配标签，无法创建抑制草稿");
      return;
    }
    const labelsJson = JSON.stringify(labels);
    setActionKey(key);
    try {
      await createInhibitionRule({
        name: `抑制草稿-${title}`,
        source_match_labels_json: labelsJson,
        target_match_labels_json: labelsJson,
        duration_seconds: 3600,
        enabled: true,
        project_id: effectiveProjectId && effectiveProjectId > 0 ? effectiveProjectId : undefined,
      });
      message.success("已创建抑制草稿");
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建抑制草稿失败"));
    } finally {
      setActionKey("");
    }
  }

  function renderActions(row: NoiseRow | RepeatRow, prefix: string) {
    const fp = "fingerprint" in row ? row.fingerprint : undefined;
    const silenceKey = `${prefix}-silence-${fp || row.title}`;
    const inhibitKey = `${prefix}-inhibit-${fp || row.title}`;
    return (
      <Space size={4} wrap>
        {fp ? (
          <Popconfirm title="确认静默该告警 2 小时？" onConfirm={() => void silence2h(row, silenceKey)}>
            <Button size="small" type="link" loading={actionKey === silenceKey}>
              静默 2h
            </Button>
          </Popconfirm>
        ) : null}
        <Popconfirm title="创建抑制规则草稿？" onConfirm={() => void createInhibitionDraft(row, inhibitKey)}>
          <Button size="small" type="link" loading={actionKey === inhibitKey}>
            抑制草稿
          </Button>
        </Popconfirm>
      </Space>
    );
  }

  return (
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
      <Card
        title={embedded ? undefined : "告警质量治理"}
        size={embedded ? "small" : "default"}
        extra={
          <Space wrap>
            {!embedded ? (
              <Select
                style={{ width: 220 }}
                value={projectId ?? 0}
                onChange={(v) => setProjectId(v || undefined)}
                options={projectOptions}
              />
            ) : null}
            <Select style={{ width: 140 }} value={windowHours} onChange={setWindowHours} options={WINDOW_OPTIONS} />
            <a onClick={() => void load()}>
              <ReloadOutlined /> 刷新
            </a>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: embedded ? 0 : undefined }}>
          噪音 Top、重复 fingerprint、通知失败率、当前 firing 与质量分
          {embedded && effectiveProjectId ? "（跟随顶栏项目）" : ""}。
        </Typography.Paragraph>
      </Card>

      <Row gutter={16}>
        <Col xs={24} md={6}>
          <Card loading={loading} size="small" title="质量分">
            <Progress
              type="dashboard"
              percent={report?.quality_score ?? 0}
              format={(p) => `${p}`}
              status={(report?.quality_score ?? 0) >= 80 ? "success" : (report?.quality_score ?? 0) >= 60 ? "normal" : "exception"}
            />
            <Typography.Text type="secondary">
              {report ? `${formatDateTime(report.from)} → ${formatDateTime(report.to)}` : "-"}
            </Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card loading={loading} size="small" title="投递事件">
            <Typography.Title level={3} style={{ margin: 0 }}>
              {report?.total_events ?? 0}
            </Typography.Title>
            <Typography.Text type="secondary">窗口内通知流水</Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card loading={loading} size="small" title="当前 firing">
            <Typography.Title level={3} style={{ margin: 0 }}>
              {report?.cur_firing_count ?? 0}
            </Typography.Title>
            <Typography.Text type="secondary">未恢复告警实例</Typography.Text>
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card loading={loading} size="small" title="通知失败率">
            <Typography.Title level={3} style={{ margin: 0 }}>
              {((report?.notify_fail_rate ?? 0) * 100).toFixed(1)}%
            </Typography.Title>
            <Typography.Text type="secondary">失败 {report?.notify_failed ?? 0} 次</Typography.Text>
          </Card>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col xs={24} lg={12}>
          <Card title="噪音 Top" size="small" loading={loading}>
            <Table
              rowKey={(r) => `${r.title}-${r.severity}-${r.fingerprint || ""}`}
              size="small"
              pagination={false}
              dataSource={report?.noise_top || []}
              columns={[
                { title: "标题", dataIndex: "title", ellipsis: true },
                { title: "级别", dataIndex: "severity", width: 90, render: (v: string) => <Tag>{v}</Tag> },
                { title: "次数", dataIndex: "count", width: 80 },
                {
                  title: "操作",
                  key: "actions",
                  width: 160,
                  render: (_: unknown, row: NoiseRow) => renderActions(row, "noise"),
                },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="重复 Fingerprint (≥3)" size="small" loading={loading}>
            <Table
              rowKey="fingerprint"
              size="small"
              pagination={false}
              dataSource={report?.repeat_fingerprints || []}
              columns={[
                { title: "Fingerprint", dataIndex: "fingerprint", ellipsis: true },
                { title: "标题", dataIndex: "title", ellipsis: true },
                { title: "次数", dataIndex: "count", width: 80 },
                {
                  title: "操作",
                  key: "actions",
                  width: 160,
                  render: (_: unknown, row: RepeatRow) => renderActions(row, "repeat"),
                },
              ]}
            />
          </Card>
        </Col>
      </Row>
    </Space>
  );
}
