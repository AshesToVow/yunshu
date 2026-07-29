import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Badge, Button, Calendar, Card, Col, DatePicker, Drawer, Form, Input, List, Modal, Row, Select, Space, Switch, Table, Tag, Typography, message } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import { listChangeEvents, type ChangeEventItem } from "../services/change-events";
import {
  conflictCheck,
  deleteFreezeWindow,
  listFreezeWindows,
  upsertFreezeWindow,
  type ConflictCheckResult,
  type FreezeWindowItem,
} from "../services/change-governance";
import { listMaintenanceWindows, type AlertMaintenanceWindowItem } from "../services/alert-maintenance";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

const SOURCE_OPTIONS = [
  { value: "", label: "全部来源" },
  { value: "cicd", label: "CI/CD" },
  { value: "k8s", label: "K8s" },
  { value: "dbmgmt", label: "数据库" },
  { value: "alert", label: "告警" },
];

const SCOPE_OPTIONS = [
  { value: "all", label: "全部" },
  { value: "cicd", label: "CI/CD" },
  { value: "k8s", label: "K8s" },
  { value: "dbmgmt", label: "数据库" },
];

export function ChangeCenterPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [source, setSource] = useState("");
  const [loading, setLoading] = useState(false);
  const [events, setEvents] = useState<ChangeEventItem[]>([]);
  const [windows, setWindows] = useState<AlertMaintenanceWindowItem[]>([]);
  const [freezes, setFreezes] = useState<FreezeWindowItem[]>([]);
  const [conflict, setConflict] = useState<ConflictCheckResult | null>(null);
  const [month, setMonth] = useState(() => dayjs());
  const [dayOpen, setDayOpen] = useState(false);
  const [dayItems, setDayItems] = useState<ChangeEventItem[]>([]);
  const [dayLabel, setDayLabel] = useState("");
  const [freezeOpen, setFreezeOpen] = useState(false);
  const [form] = Form.useForm();

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
  }, [projectId, source, month]);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const from = month.startOf("month").toISOString();
      const to = month.endOf("month").toISOString();
      const [ev, mw, fz, ck] = await Promise.all([
        listChangeEvents(projectId, { page: 1, page_size: 500, source: source || undefined, from, to }),
        listMaintenanceWindows({ projectId, page: 1, page_size: 200 }),
        listFreezeWindows(projectId, { page: 1, page_size: 200 }),
        conflictCheck(projectId, { source: source || "cicd" }),
      ]);
      setEvents(ev.list || []);
      setWindows((mw.list || []).filter((w) => w.enabled !== false));
      setFreezes(fz.list || []);
      setConflict(ck);
    } finally {
      setLoading(false);
    }
  }

  const eventsByDay = useMemo(() => {
    const map = new Map<string, ChangeEventItem[]>();
    for (const e of events) {
      const key = dayjs(e.started_at).format("YYYY-MM-DD");
      const arr = map.get(key) || [];
      arr.push(e);
      map.set(key, arr);
    }
    return map;
  }, [events]);

  const windowsByDay = useMemo(() => {
    const map = new Map<string, number>();
    for (const w of windows) {
      const start = dayjs(w.starts_at);
      const end = dayjs(w.ends_at);
      let cur = start.startOf("day");
      const last = end.startOf("day");
      while (cur.isBefore(last) || cur.isSame(last)) {
        const key = cur.format("YYYY-MM-DD");
        map.set(key, (map.get(key) || 0) + 1);
        cur = cur.add(1, "day");
      }
    }
    return map;
  }, [windows]);

  const freezesByDay = useMemo(() => {
    const map = new Map<string, number>();
    for (const w of freezes.filter((f) => f.enabled)) {
      let cur = dayjs(w.starts_at).startOf("day");
      const last = dayjs(w.ends_at).startOf("day");
      while (cur.isBefore(last) || cur.isSame(last)) {
        const key = cur.format("YYYY-MM-DD");
        map.set(key, (map.get(key) || 0) + 1);
        cur = cur.add(1, "day");
      }
    }
    return map;
  }, [freezes]);

  function dateCellRender(value: Dayjs) {
    const key = value.format("YYYY-MM-DD");
    const dayEvents = eventsByDay.get(key) || [];
    const mw = windowsByDay.get(key) || 0;
    const fz = freezesByDay.get(key) || 0;
    return (
      <ul style={{ listStyle: "none", margin: 0, padding: 0 }}>
        {dayEvents.length > 0 ? (
          <li>
            <Badge status="processing" text={`${dayEvents.length} 变更`} />
          </li>
        ) : null}
        {mw > 0 ? (
          <li>
            <Badge status="warning" text={`${mw} 维护窗`} />
          </li>
        ) : null}
        {fz > 0 ? (
          <li>
            <Badge status="error" text={`${fz} 冻结`} />
          </li>
        ) : null}
      </ul>
    );
  }

  async function submitFreeze() {
    if (!projectId) return;
    const v = await form.validateFields();
    await upsertFreezeWindow(projectId, {
      name: v.name,
      scope: v.scope,
      env: v.env || "",
      starts_at: (v.range[0] as Dayjs).toISOString(),
      ends_at: (v.range[1] as Dayjs).toISOString(),
      reason: v.reason,
      enabled: v.enabled !== false,
    });
    message.success("冻结窗口已保存");
    setFreezeOpen(false);
    form.resetFields();
    void load();
  }

  return (
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
      {conflict && (!conflict.allowed || conflict.conflict_warning || (conflict.active_freezes?.length ?? 0) > 0) ? (
        <Alert
          type={!conflict.allowed ? "error" : "warning"}
          showIcon
          message={!conflict.allowed ? "当前写操作将被拦截" : "变更治理提示"}
          description={
            conflict.message ||
            (conflict.active_freezes?.length
              ? `生效冻结窗 ${conflict.active_freezes.length} 个`
              : conflict.conflict_warning
                ? "近窗存在同类变更冲突"
                : undefined)
          }
        />
      ) : null}

      <Card
        title="变更中心"
        extra={
          <Space wrap>
            <Select style={{ width: 240 }} value={projectId} onChange={setProjectId} options={projectOptions} />
            <Select style={{ width: 140 }} value={source} onChange={setSource} options={SOURCE_OPTIONS} />
            <Button icon={<PlusOutlined />} onClick={() => setFreezeOpen(true)}>
              新建冻结窗
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
              刷新
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          日历叠加变更事件、告警维护窗与变更冻结窗；冻结窗内 CI/CD / 高危 SQL / K8s 写操作会被拒绝。
        </Typography.Paragraph>
        <Calendar
          value={month}
          onPanelChange={(v) => setMonth(v)}
          onSelect={(value) => {
            const key = value.format("YYYY-MM-DD");
            setDayLabel(key);
            setDayItems(eventsByDay.get(key) || []);
            setDayOpen(true);
          }}
          cellRender={(current, info) => {
            if (info.type === "date") return dateCellRender(current);
            return info.originNode;
          }}
        />
      </Card>

      <Row gutter={16}>
        <Col xs={24} lg={14}>
          <Card title="本月变更列表" size="small" loading={loading}>
            <Table
              rowKey="id"
              size="small"
              dataSource={events}
              pagination={{ pageSize: 10 }}
              columns={[
                { title: "时间", dataIndex: "started_at", width: 170, render: (v: string) => formatDateTime(v) },
                { title: "来源", dataIndex: "source", width: 80, render: (v: string) => <Tag>{v}</Tag> },
                { title: "动作", dataIndex: "action", width: 120 },
                { title: "风险", dataIndex: "risk_level", width: 80 },
                { title: "摘要", dataIndex: "summary" },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          <Card title="变更冻结窗口" size="small" loading={loading} style={{ marginBottom: 16 }}>
            <List
              dataSource={freezes}
              locale={{ emptyText: "暂无冻结窗口" }}
              renderItem={(w) => (
                <List.Item
                  actions={[
                    <Button
                      key="del"
                      type="link"
                      danger
                      size="small"
                      onClick={() => {
                        if (!projectId) return;
                        void deleteFreezeWindow(projectId, w.id).then(() => {
                          message.success("已删除");
                          void load();
                        });
                      }}
                    >
                      删除
                    </Button>,
                  ]}
                >
                  <List.Item.Meta
                    title={`${w.name} · ${w.scope}${w.env ? `/${w.env}` : ""}`}
                    description={`${formatDateTime(w.starts_at)} → ${formatDateTime(w.ends_at)}${w.reason ? ` · ${w.reason}` : ""}`}
                  />
                  <Tag color={w.enabled ? "red" : "default"}>{w.enabled ? "启用" : "停用"}</Tag>
                </List.Item>
              )}
            />
          </Card>
          <Card title="告警维护窗口" size="small" loading={loading}>
            <List
              dataSource={windows}
              locale={{ emptyText: "本项目暂无维护窗口" }}
              renderItem={(w) => (
                <List.Item>
                  <List.Item.Meta
                    title={w.name}
                    description={`${formatDateTime(w.starts_at)} → ${formatDateTime(w.ends_at)}`}
                  />
                  <Tag color={w.enabled ? "orange" : "default"}>{w.enabled ? "启用" : "停用"}</Tag>
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>

      <Drawer title={`${dayLabel} 变更`} open={dayOpen} onClose={() => setDayOpen(false)} width={520}>
        <List
          dataSource={dayItems}
          locale={{ emptyText: "当天无变更" }}
          renderItem={(e) => (
            <List.Item>
              <List.Item.Meta
                title={`${e.source}/${e.action}`}
                description={`${formatDateTime(e.started_at)} · ${e.summary}`}
              />
              <Tag>{e.risk_level}</Tag>
            </List.Item>
          )}
        />
      </Drawer>

      <Modal title="新建变更冻结窗口" open={freezeOpen} onCancel={() => setFreezeOpen(false)} onOk={() => void submitFreeze()} destroyOnClose>
        <Form form={form} layout="vertical" initialValues={{ scope: "all", enabled: true }}>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="scope" label="作用域" rules={[{ required: true }]}>
            <Select options={SCOPE_OPTIONS} />
          </Form.Item>
          <Form.Item name="env" label="环境（空=全部）">
            <Select allowClear options={[{ value: "prod" }, { value: "staging" }, { value: "test" }, { value: "dev" }]} />
          </Form.Item>
          <Form.Item name="range" label="时间范围" rules={[{ required: true }]}>
            <DatePicker.RangePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="reason" label="原因">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
