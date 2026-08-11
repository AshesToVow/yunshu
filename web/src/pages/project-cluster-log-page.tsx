import { CloudUploadOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import { getClusters, type ClusterItem } from "../services/clusters";
import {
  createClusterLogRule,
  deleteClusterLogRule,
  deployClusterLog,
  getProjects,
  listClusterLogAgents,
  listClusterLogRules,
  refreshClusterLogStatus,
  updateClusterLogRule,
  type ClusterLogAgent,
  type ClusterLogRule,
  type ProjectItem,
} from "../services/log-platform";
import { extractApiErrorMessage } from "../services/http";

function parseJSONList(raw?: string): string {
  if (!raw) return "";
  try {
    const arr = JSON.parse(raw) as string[];
    return Array.isArray(arr) ? arr.join(", ") : raw;
  } catch {
    return raw;
  }
}

function splitCSV(v?: string): string[] {
  if (!v) return [];
  return v
    .split(/[,;\n]/)
    .map((s) => s.trim())
    .filter(Boolean);
}

const MIN_INHERITED_QPS = 50;

/** 与后端 allocateRuleRateLimits 对齐：固定占用 + 均分剩余。 */
function allocateRuleQPS(rules: ClusterLogRule[], projectQps: number): Map<number, number> {
  const budget = projectQps > 0 ? projectQps : 2000;
  const out = new Map<number, number>();
  const inherit: number[] = [];
  let fixedSum = 0;
  for (const r of rules) {
    if (!r.enabled) continue;
    const q = r.rate_limit_qps ?? 0;
    if (q > 0) {
      out.set(r.id, q);
      fixedSum += q;
    } else {
      inherit.push(r.id);
    }
  }
  if (inherit.length === 0) return out;
  let remain = budget - fixedSum;
  if (remain < 0) remain = 0;
  const base = Math.floor(remain / inherit.length);
  let extra = remain % inherit.length;
  for (const id of inherit) {
    let q = base + (extra > 0 ? 1 : 0);
    if (extra > 0) extra -= 1;
    if (q < MIN_INHERITED_QPS) q = MIN_INHERITED_QPS;
    out.set(id, q);
  }
  return out;
}

/** 集群采集：规则 + DaemonSet 下发（与主机 Loggie 并列）。 */
export function ProjectClusterLogPage({ embedded }: { embedded?: boolean }) {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number>();
  const [rules, setRules] = useState<ClusterLogRule[]>([]);
  const [agents, setAgents] = useState<ClusterLogAgent[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [rateLimitQps, setRateLimitQps] = useState<number>(2000);
  const [form] = Form.useForm();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 1000 })
      .then((p) => {
        const list = p?.list || [];
        setProjects(list);
        if (list[0]) setProjectId(list[0].id);
      })
      .catch(() => undefined);
    void getClusters({ page: 1, page_size: 1000 })
      .then((res) => {
        const list = res?.list || [];
        setClusters(list);
        if (list[0]) setClusterId(list[0].id);
      })
      .catch(() => undefined);
  }, []);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const [r, a] = await Promise.all([
        listClusterLogRules(projectId, clusterId),
        listClusterLogAgents(projectId),
      ]);
      setRules(r?.list || []);
      setAgents(a?.list || []);
      const cur = (a?.list || []).find((x) => x.cluster_id === clusterId);
      if (cur?.rate_limit_qps && cur.rate_limit_qps > 0) {
        setRateLimitQps(cur.rate_limit_qps);
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载集群采集失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [projectId, clusterId]);

  async function onCreate() {
    if (!projectId || !clusterId) return;
    const values = await form.validateFields();
    try {
      await createClusterLogRule(projectId, {
        cluster_id: clusterId,
        name: values.name,
        match_namespaces: splitCSV(values.match_namespaces),
        match_workloads: splitCSV(values.match_workloads),
        parse_profile: values.parse_profile || "cri",
        rate_limit_qps: values.rate_limit_qps || undefined,
        enabled: values.enabled ?? true,
        remark: values.remark,
      });
      message.success("规则已创建");
      setOpen(false);
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建失败"));
    }
  }

  async function onDeploy() {
    if (!projectId || !clusterId) return;
    try {
      const agent = await deployClusterLog(projectId, {
        cluster_id: clusterId,
        namespace: "yunshu-logging",
        rate_limit_qps: rateLimitQps,
      });
      message.success(
        `已下发 DaemonSet（ready ${agent.ready_replicas}/${agent.desired_replicas}，QPS ${agent.rate_limit_qps ?? rateLimitQps}）`,
      );
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "部署失败"));
    }
  }

  const body = (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Space wrap>
        <Select
          style={{ minWidth: 200 }}
          placeholder="项目"
          value={projectId}
          options={projects.map((p) => ({ value: p.id, label: p.name }))}
          onChange={setProjectId}
        />
        <Select
          style={{ minWidth: 220 }}
          placeholder="集群"
          value={clusterId}
          options={clusters.map((c) => ({ value: c.id, label: c.name }))}
          onChange={setClusterId}
        />
        <span>
          项目限流 QPS{" "}
          <InputNumber min={100} max={100000} step={100} value={rateLimitQps} onChange={(v) => setRateLimitQps(Number(v) || 2000)} />
        </span>
        <Button icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
          刷新
        </Button>
        <Button type="primary" icon={<CloudUploadOutlined />} disabled={!clusterId || !projectId} onClick={() => void onDeploy()}>
          部署/同步 DaemonSet
        </Button>
        <Button
          icon={<PlusOutlined />}
          disabled={!clusterId || !projectId}
          onClick={() => {
            form.resetFields();
            form.setFieldsValue({ parse_profile: "cri", enabled: true, rate_limit_qps: 0 });
            setOpen(true);
          }}
        >
          新建规则
        </Button>
      </Space>
      <Card
        size="small"
        title="采集规则（按 Namespace / Workload；宽采排除系统 ns；未填 QPS 的规则均分项目配额）"
      >
        <Table
          size="small"
          rowKey="id"
          loading={loading}
          dataSource={rules}
          pagination={{ pageSize: 8 }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "名称", dataIndex: "name" },
            { title: "Namespaces", dataIndex: "match_namespaces", render: parseJSONList, ellipsis: true },
            { title: "Workloads", dataIndex: "match_workloads", render: parseJSONList, ellipsis: true },
            { title: "解析", dataIndex: "parse_profile", width: 90 },
            {
              title: "配置 QPS",
              dataIndex: "rate_limit_qps",
              width: 90,
              render: (v?: number) => (v && v > 0 ? v : "均分"),
            },
            {
              title: "生效 QPS",
              width: 90,
              render: (_: unknown, row: ClusterLogRule) => {
                const alloc = allocateRuleQPS(rules, rateLimitQps);
                const v = alloc.get(row.id);
                return row.enabled ? (v ?? "—") : "—";
              },
            },
            {
              title: "启用",
              dataIndex: "enabled",
              width: 80,
              render: (v: boolean, row: ClusterLogRule) => (
                <Switch
                  size="small"
                  checked={v}
                  onChange={(checked) =>
                    projectId
                      ? void updateClusterLogRule(projectId, row.id, { enabled: checked }).then(load)
                      : undefined
                  }
                />
              ),
            },
            {
              title: "操作",
              width: 80,
              render: (_: unknown, row: ClusterLogRule) => (
                <Popconfirm
                  title="删除规则？"
                  onConfirm={() => (projectId ? void deleteClusterLogRule(projectId, row.id).then(load) : undefined)}
                >
                  <Button type="link" size="small" danger>
                    删除
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
        />
      </Card>
      <Card size="small" title="DaemonSet 状态">
        <Table
          size="small"
          rowKey="id"
          dataSource={agents}
          pagination={false}
          columns={[
            { title: "集群ID", dataIndex: "cluster_id", width: 90 },
            { title: "命名空间", dataIndex: "namespace", width: 140 },
            { title: "状态", dataIndex: "status", width: 110, render: (v: string) => <Tag>{v}</Tag> },
            {
              title: "副本",
              width: 100,
              render: (_: unknown, r: ClusterLogAgent) => `${r.ready_replicas ?? 0}/${r.desired_replicas ?? 0}`,
            },
            { title: "限流 QPS", dataIndex: "rate_limit_qps", width: 100, render: (v?: number) => v ?? "—" },
            { title: "错误", dataIndex: "last_error", ellipsis: true, render: (v: string) => v || "—" },
            {
              title: "操作",
              width: 90,
              render: (_: unknown, r: ClusterLogAgent) => (
                <Button
                  type="link"
                  size="small"
                  onClick={() =>
                    projectId
                      ? void refreshClusterLogStatus(projectId, r.cluster_id).then(load).catch((e) => {
                          message.error(extractApiErrorMessage(e, "刷新失败"));
                        })
                      : undefined
                  }
                >
                  刷新
                </Button>
              ),
            },
          ]}
        />
      </Card>
      <Modal title="新建集群采集规则" open={open} onCancel={() => setOpen(false)} onOk={() => void onCreate()} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="match_namespaces" label="Namespaces（逗号分隔，空=宽采并排除系统 ns）">
            <Input placeholder="default, prod" />
          </Form.Item>
          <Form.Item name="match_workloads" label="Workload 名前缀（逗号分隔，可选）">
            <Input placeholder="my-app" />
          </Form.Item>
          <Form.Item
            name="rate_limit_qps"
            label="固定限流 QPS"
            extra="0 或空=从项目配额中与其他「均分」规则平分剩余；高流量可给核心规则设固定值"
          >
            <InputNumber min={0} max={100000} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="parse_profile" label="解析档">
            <Select options={[{ value: "cri", label: "CRI /var/log/pods" }]} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );

  if (embedded) return body;
  return <Card className="table-card">{body}</Card>;
}

export default ProjectClusterLogPage;
