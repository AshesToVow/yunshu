import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Input, Select, Space, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { TopologyGraphView } from "../components/k8s/topology-graph-view";
import { getClusters, listNamespaces, type ClusterItem } from "../services/clusters";
import { getWorkloadTopology } from "../services/k8s-topology";

const KIND_OPTIONS = [
  { label: "Deployment", value: "deployment" },
  { label: "StatefulSet", value: "statefulset" },
  { label: "DaemonSet", value: "daemonset" },
  { label: "Service", value: "service" },
  { label: "Ingress", value: "ingress" },
];

export function K8sResourceTopologyPage() {
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number>();
  const [namespace, setNamespace] = useState("default");
  const [namespaceOptions, setNamespaceOptions] = useState<{ label: string; value: string }[]>([]);
  const [kind, setKind] = useState("deployment");
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [graph, setGraph] = useState<Awaited<ReturnType<typeof getWorkloadTopology>> | null>(null);

  const clusterOptions = useMemo(
    () => clusters.map((c) => ({ label: c.status === 1 ? c.name : `${c.name}（已停用）`, value: c.id, disabled: c.status !== 1 })),
    [clusters],
  );

  useEffect(() => {
    void (async () => {
      const res = await getClusters({ page: 1, page_size: 200 });
      const list = res.list ?? [];
      setClusters(list);
      const first = list.find((c) => c.status === 1);
      if (first) setClusterId(first.id);
    })();
  }, []);

  useEffect(() => {
    if (!clusterId) return;
    void (async () => {
      try {
        const ns = await listNamespaces(clusterId);
        const opts = (ns.list ?? []).map((n: { name: string }) => ({ label: n.name, value: n.name }));
        setNamespaceOptions(opts);
        if (!opts.some((o) => o.value === namespace)) {
          setNamespace(opts[0]?.value ?? "default");
        }
      } catch (e) {
        message.error(e instanceof Error ? e.message : "加载命名空间失败");
      }
    })();
  }, [clusterId]);

  async function load() {
    if (!clusterId || !namespace.trim() || !name.trim()) {
      message.warning("请选择集群、命名空间并填写资源名称");
      return;
    }
    setLoading(true);
    try {
      setGraph(await getWorkloadTopology({ cluster_id: clusterId, namespace, kind, name: name.trim() }));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Card
      title="资源拓扑图"
      extra={
        <TypographyHint />
      }
    >
      <Space wrap style={{ marginBottom: 16 }}>
        <Select placeholder="集群" style={{ minWidth: 220 }} value={clusterId} options={clusterOptions} onChange={setClusterId} />
        <Select placeholder="命名空间" style={{ minWidth: 180 }} value={namespace} options={namespaceOptions} onChange={setNamespace} showSearch optionFilterProp="label" />
        <Select style={{ width: 160 }} value={kind} options={KIND_OPTIONS} onChange={setKind} />
        <Input placeholder="资源名称" style={{ width: 220 }} value={name} onChange={(e) => setName(e.target.value)} onPressEnter={() => void load()} />
        <Button type="primary" icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
          查询拓扑
        </Button>
      </Space>
      <TopologyGraphView graph={graph} loading={loading} emptyText="选择资源后点击「查询拓扑」" />
    </Card>
  );
}

function TypographyHint() {
  return <span style={{ color: "#8c8c8c", fontSize: 13 }}>入口到 Pod：Ingress → Service → Workload → ReplicaSet → Pod</span>;
}
