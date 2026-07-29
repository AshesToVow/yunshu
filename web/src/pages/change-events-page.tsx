import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Select, Space, Table, Tag } from "antd";
import { useEffect, useMemo, useState } from "react";
import { getProjects, type ProjectItem } from "../services/projects";
import { listChangeEvents, type ChangeEventItem } from "../services/change-events";
import { formatDateTime } from "../utils/format";

const SOURCE_OPTIONS = [
  { value: "", label: "全部来源" },
  { value: "cicd", label: "CI/CD" },
  { value: "k8s", label: "K8s" },
  { value: "dbmgmt", label: "数据库" },
  { value: "alert", label: "告警" },
  { value: "cmdb", label: "CMDB" },
];

export function ChangeEventsPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [source, setSource] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ChangeEventItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);

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
  }, [projectId, source, page]);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const res = await listChangeEvents(projectId, {
        page,
        page_size: 20,
        source: source || undefined,
      });
      setList(res.list);
      setTotal(res.total);
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card
      title="变更事件"
      extra={
        <Space wrap>
          <Select style={{ width: 260 }} value={projectId} onChange={(v) => { setPage(1); setProjectId(v); }} options={projectOptions} />
          <Select style={{ width: 140 }} value={source} onChange={(v) => { setPage(1); setSource(v); }} options={SOURCE_OPTIONS} />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{ current: page, total, pageSize: 20, onChange: setPage }}
        columns={[
          { title: "时间", dataIndex: "started_at", width: 180, render: (v: string) => formatDateTime(v) },
          { title: "来源", dataIndex: "source", width: 90, render: (v: string) => <Tag>{v}</Tag> },
          { title: "动作", dataIndex: "action", width: 140 },
          { title: "风险", dataIndex: "risk_level", width: 90 },
          { title: "状态", dataIndex: "status", width: 100 },
          { title: "摘要", dataIndex: "summary" },
        ]}
      />
    </Card>
  );
}
