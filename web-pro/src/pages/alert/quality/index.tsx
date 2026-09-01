// @ts-nocheck
import { PageContainer } from "@ant-design/pro-components";
import { ReloadOutlined } from "@ant-design/icons";
import { Card, Col, Progress, Row, Select, Space, Table, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { getAlertQualityReport, type AlertQualityReport } from "@/services/alert-quality";
import { getProjects, type ProjectItem } from "@/services/projects";
import { formatDateTime } from "@/utils/format";

const WINDOW_OPTIONS = [
  { value: 24, label: "近 24 小时" },
  { value: 72, label: "近 3 天" },
  { value: 168, label: "近 7 天" },
];

type Props = {
  /** 嵌入告警平台时隐藏独立项目选择，跟随顶栏上下文 */
  embedded?: boolean;
  projectContextId?: number;
};

export default function AlertQualityPage({ embedded, projectContextId }: Props) {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number | undefined>(projectContextId);
  const [windowHours, setWindowHours] = useState(24);
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState<AlertQualityReport | null>(null);

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

  return (
    <PageContainer header={{ title: "告警质量治理", subTitle: "噪声、覆盖与响应质量概览" }}>
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
              rowKey={(r) => `${r.title}-${r.severity}`}
              size="small"
              pagination={false}
              dataSource={report?.noise_top || []}
              columns={[
                { title: "标题", dataIndex: "title" },
                { title: "级别", dataIndex: "severity", width: 90, render: (v: string) => <Tag>{v}</Tag> },
                { title: "次数", dataIndex: "count", width: 80 },
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
              ]}
            />
          </Card>
        </Col>
      </Row>
    </Space>
    </PageContainer>
  );
}
