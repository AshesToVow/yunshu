import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Col, Drawer, Row, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { listAlertEvents, type AlertEventItem } from "../services/alerts";
import {
  getIncidentTimeline,
  listIncidents,
  openIncident,
  updateIncident,
  type IncidentItem,
  type IncidentTimeline,
} from "../services/incidents";
import { getIncidentContext, type IncidentContext } from "../services/service-portrait";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

const SEVERITY_OPTIONS = [
  { value: "critical,warning", label: "P1+P2 (critical/warning)" },
  { value: "critical", label: "仅 P1 (critical)" },
  { value: "warning", label: "仅 P2 (warning)" },
];

const WINDOW_OPTIONS = [
  { value: 30, label: "近 30 分钟" },
  { value: 60, label: "近 1 小时" },
  { value: 180, label: "近 3 小时" },
];

export function IncidentWorkbenchPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [severity, setSeverity] = useState("critical,warning");
  const [windowMinutes, setWindowMinutes] = useState(30);
  const [loading, setLoading] = useState(false);
  const [alerts, setAlerts] = useState<AlertEventItem[]>([]);
  const [ctx, setCtx] = useState<IncidentContext | null>(null);
  const [incidents, setIncidents] = useState<IncidentItem[]>([]);
  const [active, setActive] = useState<AlertEventItem | null>(null);
  const [timeline, setTimeline] = useState<IncidentTimeline | null>(null);

  const projectOptions = useMemo(
    () => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })),
    [projects],
  );

  useEffect(() => {
    void (async () => {
      const p = await getProjects({ page: 1, page_size: 1000 });
      setProjects(p.list);
      if (p.list[0]) setProjectId(p.list[0].id);
    })();
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, severity, windowMinutes]);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const [a, c, inc] = await Promise.all([
        listAlertEvents({
          page: 1,
          page_size: 50,
          status: "firing",
          projectId,
          severity,
        }),
        getIncidentContext(projectId, windowMinutes),
        listIncidents(projectId, { page: 1, page_size: 50 }),
      ]);
      setAlerts(a.list || []);
      setCtx(c);
      setIncidents(inc.list || []);
    } finally {
      setLoading(false);
    }
  }

  async function createFromAlert(row: AlertEventItem) {
    if (!projectId) return;
    const sev = String(row.severity || "").toLowerCase() === "critical" ? "p1" : "p2";
    const created = await openIncident(projectId, {
      title: row.title || `告警 #${row.id}`,
      severity: sev,
      summary: row.errorMessage || (row as any).error_message || "",
      alert_fingerprint: row.fingerprint || (row as any).Fingerprint,
    });
    message.success(`已开单 #${created.id}`);
    const tl = await getIncidentTimeline(projectId, created.id, windowMinutes);
    setTimeline(tl);
    void load();
  }

  async function setStatus(inc: IncidentItem, status: string) {
    if (!projectId) return;
    await updateIncident(projectId, inc.id, { status });
    message.success("状态已更新");
    void load();
  }

  return (
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
      <Card
        title="故障工作台"
        extra={
          <Space wrap>
            <Select style={{ width: 240 }} value={projectId} onChange={setProjectId} options={projectOptions} />
            <Select style={{ width: 220 }} value={severity} onChange={setSeverity} options={SEVERITY_OPTIONS} />
            <Select style={{ width: 140 }} value={windowMinutes} onChange={setWindowMinutes} options={WINDOW_OPTIONS} />
            <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
              刷新
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
          把「正在响的告警」升级成可跟踪的故障单：开单 → 接手处置 → 确认恢复 → 复盘；系统自动算 MTTA（开单到接手）和 MTTR（开单到恢复）。
        </Typography.Paragraph>
      </Card>

      <Row gutter={16}>
        <Col xs={24} lg={14}>
          <Card title="进行中告警" size="small" loading={loading}>
            <Table
              rowKey="id"
              size="small"
              dataSource={alerts}
              pagination={{ pageSize: 10 }}
              onRow={(row) => ({ onClick: () => setActive(row) })}
              columns={[
                {
                  title: "级别",
                  dataIndex: "severity",
                  width: 80,
                  render: (v: string) => {
                    const s = String(v || "").toLowerCase();
                    const label = s === "critical" ? "P1" : s === "warning" ? "P2" : v;
                    return <Tag color={s === "critical" ? "red" : "orange"}>{label}</Tag>;
                  },
                },
                { title: "标题", dataIndex: "title" },
                {
                  title: "操作",
                  width: 120,
                  render: (_: unknown, row: AlertEventItem) => (
                    <Button
                      type="link"
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        void createFromAlert(row);
                      }}
                    >
                      开单（建故障单）
                    </Button>
                  ),
                },
              ]}
            />
          </Card>
          <Card title="故障单" size="small" style={{ marginTop: 16 }} loading={loading}>
            <Table
              rowKey="id"
              size="small"
              dataSource={incidents}
              pagination={{ pageSize: 8 }}
              onRow={(row) => ({
                onClick: () => {
                  if (!projectId) return;
                  void getIncidentTimeline(projectId, row.id, windowMinutes).then(setTimeline);
                },
              })}
              columns={[
                { title: "ID", dataIndex: "id", width: 60 },
                { title: "标题", dataIndex: "title" },
                { title: "级别", dataIndex: "severity", width: 70, render: (v: string) => <Tag>{v}</Tag> },
                { title: "状态", dataIndex: "status", width: 100, render: (v: string) => <Tag>{v}</Tag> },
                {
                  title: "MTTA/MTTR",
                  width: 140,
                  render: (_: unknown, r: IncidentItem) =>
                    `${r.mtta_seconds ?? "-"}s / ${r.mttr_seconds ?? "-"}s`,
                },
                {
                  title: "处置",
                  width: 220,
                  render: (_: unknown, r: IncidentItem) => (
                    <Space size={0}>
                      {r.status === "open" ? (
                        <Button type="link" size="small" onClick={() => void setStatus(r, "mitigating")}>
                          接手（开始处置）
                        </Button>
                      ) : null}
                      {r.status === "mitigating" || r.status === "open" ? (
                        <Button type="link" size="small" onClick={() => void setStatus(r, "resolved")}>
                          恢复（故障已消除）
                        </Button>
                      ) : null}
                      {r.status === "resolved" ? (
                        <Button type="link" size="small" onClick={() => void setStatus(r, "postmortem")}>
                          复盘（归档）
                        </Button>
                      ) : null}
                    </Space>
                  ),
                },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          <Card
            title={`近 ${ctx?.window_minutes || windowMinutes} 分钟变更`}
            size="small"
            loading={loading}
            extra={<Link to="/change-center">变更中心</Link>}
          >
            <Table
              rowKey="id"
              size="small"
              pagination={{ pageSize: 8 }}
              dataSource={ctx?.changes || []}
              columns={[
                { title: "来源", dataIndex: "source", width: 70, render: (v: string) => <Tag>{v}</Tag> },
                { title: "摘要", dataIndex: "summary" },
              ]}
            />
          </Card>
          <Card title="同期发布" size="small" style={{ marginTop: 16 }} loading={loading}>
            <Table
              rowKey="id"
              size="small"
              pagination={{ pageSize: 5 }}
              dataSource={ctx?.releases || []}
              columns={[
                { title: "标题", dataIndex: "title" },
                { title: "状态", dataIndex: "status", width: 100, render: (v: string) => <Tag>{v}</Tag> },
              ]}
            />
          </Card>
        </Col>
      </Row>

      <Drawer title={active?.title || "告警详情"} open={!!active} onClose={() => setActive(null)} width={480}>
        {active ? (
          <Space direction="vertical" style={{ width: "100%" }}>
            <div>
              <Tag color={String(active.severity).toLowerCase() === "critical" ? "red" : "orange"}>
                {active.severity}
              </Tag>
              <Tag>{active.status}</Tag>
            </div>
            <Typography.Text type="secondary">集群：{active.cluster || "-"}</Typography.Text>
            <Typography.Text type="secondary">
              时间：{formatDateTime(active.createdAt || (active as any).created_at)}
            </Typography.Text>
            <Button type="primary" onClick={() => void createFromAlert(active)}>
              从该告警开故障单
            </Button>
          </Space>
        ) : null}
      </Drawer>

      <Drawer
        title={timeline ? `故障单 #${timeline.incident.id} 时间线` : "时间线"}
        open={!!timeline}
        onClose={() => setTimeline(null)}
        width={560}
      >
        {timeline ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            <Typography.Text>
              MTTA {timeline.mtta_seconds ?? timeline.incident.mtta_seconds ?? "-"}s · MTTR{" "}
              {timeline.mttr_seconds ?? timeline.incident.mttr_seconds ?? "-"}s
            </Typography.Text>
            <Card size="small" title="近窗变更">
              <Table
                rowKey="id"
                size="small"
                pagination={false}
                dataSource={timeline.changes}
                columns={[
                  { title: "来源", dataIndex: "source", width: 70 },
                  { title: "摘要", dataIndex: "summary" },
                ]}
              />
            </Card>
            <Card size="small" title="关联告警">
              <Table
                rowKey="id"
                size="small"
                pagination={false}
                dataSource={timeline.alerts}
                columns={[
                  { title: "级别", dataIndex: "severity", width: 90 },
                  { title: "标题", dataIndex: "title" },
                ]}
              />
            </Card>
          </Space>
        ) : null}
      </Drawer>
    </Space>
  );
}
