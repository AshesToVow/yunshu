import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Select, Space, message } from "antd";
import { useEffect, useState } from "react";
import { TopologyGraphView } from "../components/k8s/topology-graph-view";
import { getApplicationTopology, getProjects, type ApplicationTopologyGraph } from "../services/projects";

export function ApplicationTopologyPage() {
  const [projects, setProjects] = useState<{ label: string; value: number }[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [graph, setGraph] = useState<ApplicationTopologyGraph | null>(null);

  useEffect(() => {
    void (async () => {
      const res = await getProjects({ page: 1, page_size: 200 });
      const opts = (res.list ?? []).map((p) => ({ label: `${p.name} (${p.code})`, value: p.id }));
      setProjects(opts);
      if (opts[0]) setProjectId(opts[0].value);
    })();
  }, []);

  async function load(pid = projectId) {
    if (!pid) {
      message.warning("请选择项目");
      return;
    }
    setLoading(true);
    try {
      setGraph(await getApplicationTopology(pid));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (projectId) void load(projectId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  return (
    <Card title="应用拓扑图" extra={<span style={{ color: "#8c8c8c", fontSize: 13 }}>项目 → 分组 → 服务器 → 服务 → 日志源</span>}>
      <Space wrap style={{ marginBottom: 16 }}>
        <Select
          placeholder="选择项目"
          style={{ minWidth: 320 }}
          value={projectId}
          options={projects}
          onChange={setProjectId}
          showSearch
          optionFilterProp="label"
        />
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
          刷新
        </Button>
      </Space>
      <TopologyGraphView graph={graph} loading={loading} emptyText="该项目暂无服务/服务器数据" />
    </Card>
  );
}
