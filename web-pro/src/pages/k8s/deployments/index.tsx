// @ts-nocheck
import { LegacyShell } from "@/components/LegacyShell";
import {
  ApartmentOutlined,
  ColumnHeightOutlined,
  DownOutlined,
  EyeOutlined,
  FileTextOutlined,
  ReloadOutlined,
  RollbackOutlined,
  ScissorOutlined,
  TagsOutlined,
} from "@ant-design/icons";
import { Button, Card, Dropdown, Form, Input, InputNumber, Modal, Popconfirm, Progress, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useRef, useState } from "react";
import { useKeyValueViewer } from "@/components/k8s/key-value-viewer";
import { useRelatedPodsDrawer } from "@/components/k8s/related-pods-drawer";
import { RealtimeUsageText, WorkloadCpuUsageBars, WorkloadMemUsageBars } from "@/components/k8s/k8s-resource-usage-cells";
import { useWorkloadFormActions } from "@/components/k8s/workload-form-actions";
import { YamlCrudPage } from "@/components/k8s/yaml-crud-page";
import { InputNumber as AntdInputNumber } from "antd";
import { useWorkloadProgressOptional } from "@/contexts/workload-progress-context";
import { listNamespaces as listClusterNamespaces } from "@/services/clusters";
import { TopologyGraphView } from "@/components/k8s/topology-graph-view";
import { getWorkloadTopology, type TopologyGraph } from "@/services/k8s-topology";
import { formatDateTime } from "@/utils/format";
import { extractApiErrorMessage } from "@/services/http";
import {
  applyDeployment,
  buildCpuMemoryResourceMaps,
  deleteDeployment,
  getDeploymentDetail,
  listDeployments,
  getDeploymentRolloutStatus,
  listDeploymentPods,
  listDeploymentRevisions,
  patchDeploymentContainerResources,
  restartDeployment,
  rolloutUndoDeployment,
  scaleDeployment,
  type DeploymentRolloutStatus,
  type DeploymentRevisionItem,
  type WorkloadDetail,
  type WorkloadItem,
} from "@/services/workloads";
import {
  WorkloadFormModal,
  NameNamespaceItems,
  ContainerCommonItems,
  WorkloadAdvancedItems,
  WorkloadPolicyItems,
  DeploymentHealthAndImagePullSecretsItems,
  buildDeploymentYaml,
  deploymentObjToForm,
  deploymentYamlToForm,
  qosFromResources,
  type DeploymentFormValues,
} from "@/components/k8s/workload-forms";

function DeploymentDetailQuickEdit({
  detail,
  detailYaml,
  setDetailYaml,
}: {
  detail: WorkloadDetail;
  detailYaml: string;
  setDetailYaml: (next: string) => void;
}) {
  const [detailForm] = Form.useForm<DeploymentFormValues>();
  const values = deploymentYamlToForm(detailYaml || "") ?? deploymentObjToForm(detail.object) ?? deploymentYamlToForm(detail.yaml ?? "");
  const liveness = detail.object?.spec?.template?.spec?.containers?.[0]?.livenessProbe;
  const readiness = detail.object?.spec?.template?.spec?.containers?.[0]?.readinessProbe;
  const formatProbe = (probe: any): string => {
    if (!probe) return "-";
    if (probe.httpGet) {
      const hp = probe.httpGet;
      const port = typeof hp.port === "number" ? hp.port : String(hp.port || "");
      return `${hp.path || "/"} : ${port} ${hp.scheme || ""}`;
    }
    if (probe.tcpSocket) {
      const tp = probe.tcpSocket;
      const port = typeof tp.port === "number" ? tp.port : String(tp.port || "");
      return `TCP : ${port}`;
    }
    return "自定义探针";
  };
  return (
    <Form
      form={detailForm}
      layout="vertical"
      initialValues={values ?? undefined}
      onValuesChange={(_, allValues) => {
        try {
          setDetailYaml(buildDeploymentYaml(allValues as DeploymentFormValues));
        } catch {
          // ignore partial invalid values during typing
        }
      }}
    >
      <Card size="small" title="快速编辑">
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="name" label="名称" rules={[{ required: true }]} style={{ flex: 1 }}>
            <Input />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true }]} style={{ width: 220 }}>
            <Input />
          </Form.Item>
          <Form.Item name="replicas" label="副本数" style={{ width: 160 }}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Space>
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="container_name" label="容器名" style={{ width: 220 }}>
            <Input />
          </Form.Item>
          <Form.Item name="image" label="容器镜像" style={{ flex: 1 }}>
            <Input />
          </Form.Item>
          <Form.Item name="port" label="容器端口" style={{ width: 160 }}>
            <InputNumber min={1} max={65535} style={{ width: "100%" }} />
          </Form.Item>
        </Space>
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="requests_cpu" label="CPU Request" style={{ width: 180 }}>
            <Input />
          </Form.Item>
          <Form.Item name="limits_cpu" label="CPU Limit" style={{ width: 180 }}>
            <Input />
          </Form.Item>
          <Form.Item name="requests_memory" label="MEM Request" style={{ width: 180 }}>
            <Input />
          </Form.Item>
          <Form.Item name="limits_memory" label="MEM Limit" style={{ width: 180 }}>
            <Input />
          </Form.Item>
        </Space>
        <Typography.Text type="secondary">探针摘要：Liveness {formatProbe(liveness)}；Readiness {formatProbe(readiness)}</Typography.Text>
      </Card>
    </Form>
  );
}

export default function DeploymentsPage() {
  return (
    <LegacyShell>
      <DeploymentsPageInner />
    </LegacyShell>
  );
}

function DeploymentsPageInner() {
  const progress = useWorkloadProgressOptional();
  const listReloadRef = useRef<() => void>(() => {});
  const [form] = Form.useForm<DeploymentFormValues>();
  const formActions = useWorkloadFormActions<DeploymentFormValues>({
    form,
    mode: true,
    defaultMode: "create",
    getDetail: async (clusterId, namespace, name) => await getDeploymentDetail(clusterId, namespace, name),
    toFormValues: (d) => deploymentObjToForm(d.object) ?? deploymentYamlToForm(d.yaml ?? ""),
    buildFallbackValues: ({ recordName, namespace, record }) => ({
      name: recordName,
      namespace,
      replicas: Number(record?.replicas ?? 1) || 1,
      container_name: "",
      image: "",
      env_pairs: [{ key: "", value: "" }],
    }),
  });

  const [scaleOpen, setScaleOpen] = useState(false);
  const [scaleValue, setScaleValue] = useState<number>(1);
  const [scaleTarget, setScaleTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
  const [verticalOpen, setVerticalOpen] = useState(false);
  const [verticalTarget, setVerticalTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
  const [rolloutOpen, setRolloutOpen] = useState(false);
  const [rolloutTarget, setRolloutTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
  const [rolloutStatus, setRolloutStatus] = useState<DeploymentRolloutStatus | null>(null);
  const [revisionOpen, setRevisionOpen] = useState(false);
  const [revisionTarget, setRevisionTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
  const [revisionRows, setRevisionRows] = useState<DeploymentRevisionItem[]>([]);
  const [revisionLoading, setRevisionLoading] = useState(false);
  const [verticalForm] = Form.useForm<{
    container_name?: string;
    requests_cpu?: string;
    requests_memory?: string;
    limits_cpu?: string;
    limits_memory?: string;
  }>();
  const { openPods, viewer: podsViewer } = useRelatedPodsDrawer(async ({ clusterId, namespace, name }) => await listDeploymentPods(clusterId, namespace, name));
  const [topoOpen, setTopoOpen] = useState(false);
  const [topoLoading, setTopoLoading] = useState(false);
  const [topoGraph, setTopoGraph] = useState<TopologyGraph | null>(null);

  async function openTopology(clusterId: number, namespace: string, name: string) {
    setTopoOpen(true);
    setTopoLoading(true);
    setTopoGraph(null);
    try {
      setTopoGraph(await getWorkloadTopology({ cluster_id: clusterId, namespace, kind: "deployment", name }));
    } catch {
      setTopoOpen(false);
    } finally {
      setTopoLoading(false);
    }
  }

  async function openRevisionRollback(clusterId: number, namespace: string, name: string) {
    setRevisionTarget({ clusterId, namespace, name });
    setRevisionOpen(true);
    setRevisionLoading(true);
    setRevisionRows([]);
    try {
      const rows = await listDeploymentRevisions(clusterId, namespace, name);
      setRevisionRows(rows || []);
    } catch {
      message.error("加载历史版本失败");
      setRevisionOpen(false);
    } finally {
      setRevisionLoading(false);
    }
  }

  useEffect(() => {
    if (!rolloutOpen || !rolloutTarget) return;
    let cancelled = false;
    const poll = async () => {
      try {
        const st = await getDeploymentRolloutStatus(rolloutTarget.clusterId, rolloutTarget.namespace, rolloutTarget.name);
        if (!cancelled) setRolloutStatus(st);
      } catch {
        // ignore transient errors while rolling
      }
    };
    void poll();
    const timer = window.setInterval(() => void poll(), 2000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [rolloutOpen, rolloutTarget]);
  const { renderKVIcon, viewer } = useKeyValueViewer({
    width: 760,
    compact: true,
    pageSize: 10,
    destroyOnClose: true,
    emptyText: (title) => `暂无${title}`,
  });

  const columns: ColumnsType<WorkloadItem> = [
    { title: "命名空间", dataIndex: "namespace", width: 110 },
    { title: "名称", dataIndex: "name", width: 200 },
    { title: "副本", dataIndex: "replicas", width: 80 },
    {
      title: "资源汇总",
      dataIndex: "resource_text",
      width: 180,
      render: (v?: string) => <Typography.Text style={{ whiteSpace: "pre-wrap", fontSize: 12 }}>{v || "-"}</Typography.Text>,
    },
    { title: "标签", key: "labels", width: 70, align: "center", render: (_, r) => renderKVIcon("标签", <TagsOutlined />, r.labels) },
    { title: "注解", key: "annotations", width: 70, align: "center", render: (_, r) => renderKVIcon("注解", <FileTextOutlined />, r.annotations) },
    {
      title: "容器",
      dataIndex: "containers_text",
      width: 220,
      render: (v?: string) => <Typography.Text style={{ whiteSpace: "pre-wrap", fontSize: 12 }}>{v || "-"}</Typography.Text>,
    },
    {
      title: "实时用量",
      key: "usage_rt",
      width: 130,
      render: (_, r) => <RealtimeUsageText cpu={r.cpu_usage} mem={r.mem_usage} />,
    },
    {
      title: "CPU 资源",
      key: "cpu_bars",
      width: 152,
      render: (_, r) => <WorkloadCpuUsageBars row={r} />,
    },
    {
      title: "内存资源",
      key: "mem_bars",
      width: 152,
      render: (_, r) => <WorkloadMemUsageBars row={r} />,
    },
    {
      title: "当前状态",
      key: "status",
      width: 180,
      render: (_, r) => {
        const p = typeof r.ready_percent === "number" ? r.ready_percent : 0;
        return <Progress percent={Math.max(0, Math.min(100, p))} size="small" format={() => r.ready || `${p}%`} />;
      },
    },
    { title: "运行时长", dataIndex: "age", width: 90, fixed: "right" },
    { title: "条件", dataIndex: "conditions_text", width: 160, fixed: "right" },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<WorkloadItem, WorkloadDetail>
        title="Deployment 控制器管理"
        needNamespace
        watchResource="deployments"
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        columns={columns}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listDeployments(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getDeploymentDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyDeployment(clusterId, manifest),
          remove: async (args) => await deleteDeployment(args.clusterId, args.namespace ?? "default", args.name, args),
        }}
        onEdit={(record, ctx) => formActions.openEdit({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }, record)}
        onToolbarReady={(ctx) => {
          listReloadRef.current = ctx.reload;
        }}
        onCreateDrawerOpen={(ctx) => {
          if (!ctx.clusterId) return;
          formActions.prepareCreate(
            { clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default" },
            {
              namespace: ctx.namespace ?? "default",
              replicas: 1,
              env_pairs: [{ key: "", value: "" }],
              name: "",
              container_name: "",
              image: "nginx:latest",
              strategy_type: "RollingUpdate",
              rolling_update_max_surge: "1",
              rolling_update_max_unavailable: "0",
              min_ready_seconds: 5,
              progress_deadline_seconds: 600,
            } as Partial<DeploymentFormValues>,
          );
        }}
        renderCreateFormTab={(ctx) => (
          <WorkloadFormModal<DeploymentFormValues>
            embedded
            title="Deployment 表单创建"
            open={false}
            loading={formActions.loading}
            form={form}
            onCancel={ctx.closeCreateDrawer}
            onSubmit={(values) => {
              if (!formActions.ctx) return;
              const fctx = formActions.ctx;
              formActions.setLoading(true);
              void (async () => {
                try {
                  const manifest = buildDeploymentYaml(values);
                  await applyDeployment(fctx.clusterId, manifest);
                  message.success("已应用 Deployment");
                  ctx.closeCreateDrawer();
                  listReloadRef.current();
                } finally {
                  formActions.setLoading(false);
                }
              })();
            }}
          >
            <NameNamespaceItems />
            <Space style={{ width: "100%" }} align="start">
              <Form.Item name="replicas" label="副本数" rules={[{ required: true, message: "请输入副本数" }]} style={{ width: 240 }}>
                <AntdInputNumber min={0} />
              </Form.Item>
            </Space>
            <ContainerCommonItems showPort />
            <WorkloadAdvancedItems />
            <WorkloadPolicyItems showDeployStrategy />
            <DeploymentHealthAndImagePullSecretsItems />
            <Form.Item noStyle shouldUpdate>
              {() => {
                const v = form.getFieldsValue();
                const qos = qosFromResources(v);
                return (
                  <Typography.Text type="secondary">
                    QoS 说明：Deployment 不能直接设置 QoS，QoS 由 resources 推导，当前预估为：{qos}
                  </Typography.Text>
                );
              }}
            </Form.Item>
          </WorkloadFormModal>
        )}
        createTemplate={({ namespace }) => `apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo-deployment
  namespace: ${namespace ?? "default"}
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  minReadySeconds: 5
  selector:
    matchLabels:
      app: demo
  template:
    metadata:
      labels:
        app: demo
    spec:
      containers:
        - name: demo
          image: nginx:latest
          ports:
            - containerPort: 80
`}
        detailExtra={(detail, yamlCtx) =>
          yamlCtx ? (
            <DeploymentDetailQuickEdit detail={detail} detailYaml={yamlCtx.yaml} setDetailYaml={yamlCtx.setYaml} />
          ) : null
        }
        extraRowActions={(record, ctx) => (
          <Space>
            <Dropdown
              menu={{
                items: [
                  {
                    key: "pods",
                    label: "关联 Pods",
                    icon: <EyeOutlined />,
                    onClick: () => openPods({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }),
                  },
                  {
                    key: "topology",
                    label: "资源拓扑",
                    icon: <ApartmentOutlined />,
                    onClick: () => void openTopology(ctx.clusterId, ctx.namespace ?? "default", record.name),
                  },
                  {
                    key: "scale",
                    label: "水平扩缩（副本 · HPA / scale 类）",
                    icon: <ScissorOutlined />,
                    onClick: () => {
                      setScaleTarget({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      setScaleValue(Number(record.replicas ?? 1) || 1);
                      setScaleOpen(true);
                    },
                  },
                  {
                    key: "vertical",
                    label: "垂直扩缩（resources · VPA 类）",
                    icon: <ColumnHeightOutlined />,
                    onClick: () => {
                      setVerticalTarget({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      verticalForm.resetFields();
                      setVerticalOpen(true);
                    },
                  },
                  {
                    key: "rollout",
                    label: "发布进度",
                    icon: <EyeOutlined />,
                    onClick: () => {
                      setRolloutTarget({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      setRolloutStatus(null);
                      setRolloutOpen(true);
                    },
                  },
                  {
                    key: "restart",
                    label: "重启工作负载",
                    icon: <ReloadOutlined />,
                    onClick: () => {
                      void (async () => {
                        if (!ctx.clusterId) {
                          message.error("请先选择集群");
                          return;
                        }
                        const ns = ctx.namespace ?? "default";
                        try {
                          await restartDeployment(ctx.clusterId, ns, record.name);
                          message.success("已触发滚动重启");
                          progress?.track({
                            kind: "Deployment",
                            clusterId: ctx.clusterId,
                            namespace: ns,
                            name: record.name,
                            title: `重启 ${record.name}`,
                          });
                          ctx.reload();
                        } catch (e) {
                          message.error(extractApiErrorMessage(e, "重启失败"));
                        }
                      })();
                    },
                  },
                  {
                    key: "undo",
                    label: "回滚上一版本",
                    icon: <ReloadOutlined />,
                    onClick: () => {
                      void (async () => {
                        const ns = ctx.namespace ?? "default";
                        await rolloutUndoDeployment(ctx.clusterId, ns, record.name);
                        message.success("已触发平台回滚");
                        progress?.track({
                          kind: "Deployment",
                          clusterId: ctx.clusterId,
                          namespace: ns,
                          name: record.name,
                          title: `回滚 ${record.name}`,
                        });
                        ctx.reload();
                      })();
                    },
                  },
                  {
                    key: "undo-pick",
                    label: "选择版本回滚",
                    icon: <RollbackOutlined />,
                    onClick: () => {
                      const ns = ctx.namespace ?? "default";
                      void openRevisionRollback(ctx.clusterId, ns, record.name);
                    },
                  },
                ],
              }}
            >
              <Button type="link">
                更多 <DownOutlined />
              </Button>
            </Dropdown>
          </Space>
        )}
      />

      <Modal
        title={`发布进度${rolloutTarget ? `：${rolloutTarget.name}` : ""}`}
        open={rolloutOpen}
        onCancel={() => setRolloutOpen(false)}
        footer={null}
        width={560}
      >
        {rolloutStatus ? (
          <Space direction="vertical" style={{ width: "100%" }}>
            <Typography.Text>
              策略 {rolloutStatus.strategy_type}
              {rolloutStatus.max_surge != null ? ` · maxSurge ${rolloutStatus.max_surge}` : ""}
              {rolloutStatus.max_unavailable != null ? ` · maxUnavailable ${rolloutStatus.max_unavailable}` : ""}
            </Typography.Text>
            <Progress
              percent={
                rolloutStatus.replicas > 0
                  ? Math.round((rolloutStatus.ready_replicas / rolloutStatus.replicas) * 100)
                  : 0
              }
              status={rolloutStatus.complete ? "success" : "active"}
              format={() => `就绪 ${rolloutStatus.ready_replicas}/${rolloutStatus.replicas}`}
            />
            <Typography.Text type="secondary">
              已更新 {rolloutStatus.updated_replicas} · 可用 {rolloutStatus.available_replicas} · 不可用 {rolloutStatus.unavailable_replicas}
              {rolloutStatus.complete ? " · 发布完成" : rolloutStatus.progressing ? " · 滚动中" : ""}
            </Typography.Text>
            {rolloutStatus.conditions?.length ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {rolloutStatus.conditions.map((c) => `${c.type}=${c.status}`).join(" · ")}
              </Typography.Text>
            ) : null}
          </Space>
        ) : (
          <Typography.Text type="secondary">加载中…</Typography.Text>
        )}
      </Modal>

      <Modal
        title={`选择回滚版本${revisionTarget ? `：${revisionTarget.name}` : ""}`}
        open={revisionOpen}
        onCancel={() => setRevisionOpen(false)}
        footer={null}
        width={640}
      >
        <Table
          rowKey="revision"
          size="small"
          loading={revisionLoading}
          dataSource={revisionRows}
          pagination={false}
          columns={[
            { title: "Revision", dataIndex: "revision", width: 90 },
            {
              title: "状态",
              key: "current",
              width: 90,
              render: (_: unknown, r: DeploymentRevisionItem) => (r.current ? <Tag color="blue">当前</Tag> : null),
            },
            { title: "副本", key: "replicas", width: 100, render: (_: unknown, r: DeploymentRevisionItem) => `${r.ready}/${r.replicas}` },
            {
              title: "创建时间",
              dataIndex: "created_at",
              render: (v?: string) => (v ? formatDateTime(v) : "—"),
            },
            {
              title: "操作",
              width: 100,
              render: (_: unknown, r: DeploymentRevisionItem) =>
                r.current ? (
                  <Typography.Text type="secondary">—</Typography.Text>
                ) : (
                  <Popconfirm
                    title={`回滚到 revision ${r.revision}？`}
                    onConfirm={() => {
                      if (!revisionTarget) return;
                      void (async () => {
                        await rolloutUndoDeployment(
                          revisionTarget.clusterId,
                          revisionTarget.namespace,
                          revisionTarget.name,
                          r.revision,
                        );
                        message.success(`已回滚到 revision ${r.revision}`);
                        setRevisionOpen(false);
                        progress?.track({
                          kind: "Deployment",
                          clusterId: revisionTarget.clusterId,
                          namespace: revisionTarget.namespace,
                          name: revisionTarget.name,
                          title: `回滚 ${revisionTarget.name} → rev ${r.revision}`,
                        });
                        listReloadRef.current();
                      })();
                    }}
                  >
                    <Button size="small" icon={<RollbackOutlined />}>
                      回滚
                    </Button>
                  </Popconfirm>
                ),
            },
          ]}
        />
      </Modal>

      <Modal
        title={`Deployment 水平扩缩（HPA / scale 子资源类）${scaleTarget ? `：${scaleTarget.name}` : ""}`}
        open={scaleOpen}
        onCancel={() => setScaleOpen(false)}
        onOk={() => {
          if (!scaleTarget) return;
          void (async () => {
            await scaleDeployment(scaleTarget.clusterId, scaleTarget.namespace, scaleTarget.name, scaleValue);
            message.success("扩缩容成功");
            setScaleOpen(false);
          })();
        }}
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          与 Kubernetes HPA 使用的 scale 子资源同类：适用于 Deployment、StatefulSet、ReplicaSet、ReplicationController。
          DaemonSet、Job、CronJob 无此类「持续副本水平伸缩」，控制台不提供对应入口。
        </Typography.Paragraph>
        <Space>
          <span>副本数</span>
          <InputNumber min={0} value={scaleValue} onChange={(v) => setScaleValue(Number(v ?? 0))} />
        </Space>
      </Modal>

      <Modal
        title={`Deployment 垂直扩缩（Pod 模板 resources · VPA 类）${verticalTarget ? `：${verticalTarget.name}` : ""}`}
        open={verticalOpen}
        onCancel={() => setVerticalOpen(false)}
        destroyOnClose
        width={560}
        onOk={() => {
          if (!verticalTarget) return;
          void verticalForm.validateFields().then(async (values) => {
            const { requests, limits } = buildCpuMemoryResourceMaps(values);
            if (Object.keys(requests).length === 0 && Object.keys(limits).length === 0) {
              message.warning("请至少填写一项 requests 或 limits（如 cpu/memory）");
              return;
            }
            await patchDeploymentContainerResources(verticalTarget.clusterId, verticalTarget.namespace, verticalTarget.name, {
              container_name: values.container_name,
              requests,
              limits,
            });
            message.success("已更新容器资源");
            setVerticalOpen(false);
            listReloadRef.current();
          });
        }}
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          与 VPA 直接修改工作负载 Pod 模板的 resources 属于同一类能力；生产环境请结合集群 VPA 策略评估滚动与装箱影响。
          留空容器名则修改第一个容器。示例：CPU <Typography.Text code>100m</Typography.Text>，内存{" "}
          <Typography.Text code>256Mi</Typography.Text>。
        </Typography.Paragraph>
        <Form form={verticalForm} layout="vertical">
          <Form.Item label="容器名（可选）" name="container_name">
            <Input placeholder="默认第一个容器" allowClear />
          </Form.Item>
          <Form.Item label="requests.cpu" name="requests_cpu">
            <Input placeholder="如 100m" allowClear />
          </Form.Item>
          <Form.Item label="requests.memory" name="requests_memory">
            <Input placeholder="如 256Mi" allowClear />
          </Form.Item>
          <Form.Item label="limits.cpu" name="limits_cpu">
            <Input placeholder="如 500m" allowClear />
          </Form.Item>
          <Form.Item label="limits.memory" name="limits_memory">
            <Input placeholder="如 512Mi" allowClear />
          </Form.Item>
        </Form>
      </Modal>

      {podsViewer}

      <WorkloadFormModal<DeploymentFormValues>
        title="Deployment 表单编辑"
        open={formActions.open && formActions.mode === "edit"}
        loading={formActions.loading}
        form={form}
        onCancel={formActions.close}
        onSubmit={(values) => {
          if (!formActions.ctx) return;
          const wctx = formActions.ctx;
          formActions.setLoading(true);
          void (async () => {
            try {
              const manifest = buildDeploymentYaml(values);
              await applyDeployment(wctx.clusterId, manifest);
              message.success("已应用 Deployment");
              formActions.close();
              listReloadRef.current();
            } finally {
              formActions.setLoading(false);
            }
          })();
        }}
      >
        <NameNamespaceItems />
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="replicas" label="副本数" rules={[{ required: true, message: "请输入副本数" }]} style={{ width: 240 }}>
            <AntdInputNumber min={0} />
          </Form.Item>
        </Space>
        <ContainerCommonItems showPort />
        <WorkloadAdvancedItems />
        <WorkloadPolicyItems showDeployStrategy />
        <DeploymentHealthAndImagePullSecretsItems />
        <Form.Item noStyle shouldUpdate>
          {() => {
            const v = form.getFieldsValue();
            const qos = qosFromResources(v);
            return (
              <Typography.Text type="secondary">
                QoS 说明：Deployment 不能直接设置 QoS，QoS 由 resources 推导，当前预估为：{qos}
              </Typography.Text>
            );
          }}
        </Form.Item>
      </WorkloadFormModal>
      <Modal title="Deployment 资源拓扑（Ingress → Service → Workload → Pod）" open={topoOpen} onCancel={() => setTopoOpen(false)} footer={null} width={900}>
        {topoLoading ? <Typography.Text>加载中…</Typography.Text> : <TopologyGraphView graph={topoGraph} />}
      </Modal>
      {viewer}
    </>
  );
}

