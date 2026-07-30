import { DeleteOutlined, LinkOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, AutoComplete, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { listAlertMonitorRules } from "../services/alert-platform";
import { listCicdServices } from "../services/cicd";
import { getClusters } from "../services/clusters";
import { listDbInstances } from "../services/dbmgmt";
import { listNamespaces } from "../services/namespaces";
import {
  getProjectLogSources,
  getProjectServices,
  getProjects,
  type ProjectItem,
} from "../services/projects";
import {
  addServiceCatalogLink,
  deleteServiceCatalog,
  deleteServiceCatalogLink,
  listServiceCatalog,
  upsertServiceCatalog,
  type ServiceCatalogItem,
  type ServiceLinkItem,
} from "../services/service-catalog";
import { getUsers } from "../services/users";
import { listDaemonSets, listDeployments, listStatefulSets } from "../services/workloads";
import type { UserItem } from "../types/api";

const LINK_TYPE_OPTIONS = [
  { value: "cicd_service", label: "CI/CD 服务" },
  { value: "cmdb_service", label: "CMDB 服务" },
  { value: "log_source", label: "日志源" },
  { value: "k8s_workload", label: "K8s Workload" },
  { value: "alert_monitor_rule", label: "告警规则" },
  { value: "db_instance", label: "数据库实例" },
];

const K8S_KIND_OPTIONS = [
  { value: "Deployment", label: "Deployment" },
  { value: "StatefulSet", label: "StatefulSet" },
  { value: "DaemonSet", label: "DaemonSet" },
];

type RefOption = { value: number; label: string };

export function ServiceCatalogPage() {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ServiceCatalogItem[]>([]);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [productLineOptions, setProductLineOptions] = useState<string[]>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [linkOpen, setLinkOpen] = useState(false);
  const [current, setCurrent] = useState<ServiceCatalogItem | null>(null);
  const [form] = Form.useForm();
  const [linkForm] = Form.useForm();

  const linkType = Form.useWatch("link_type", linkForm) as string | undefined;
  const k8sClusterId = Form.useWatch("k8s_cluster_id", linkForm) as number | undefined;
  const k8sNamespace = Form.useWatch("k8s_namespace", linkForm) as string | undefined;
  const k8sKind = Form.useWatch("k8s_kind", linkForm) as string | undefined;

  const [refOptions, setRefOptions] = useState<RefOption[]>([]);
  const [refLoading, setRefLoading] = useState(false);
  const [clusterOptions, setClusterOptions] = useState<{ value: number; label: string }[]>([]);
  const [nsOptions, setNsOptions] = useState<{ value: string; label: string }[]>([]);
  const [workloadOptions, setWorkloadOptions] = useState<{ value: string; label: string }[]>([]);

  const projectOptions = useMemo(
    () => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })),
    [projects],
  );
  const userOptions = useMemo(
    () =>
      users.map((u) => ({
        value: u.username,
        label: `${u.nickname || u.username} (${u.username})`,
      })),
    [users],
  );

  useEffect(() => {
    void (async () => {
      const [p, u] = await Promise.all([
        getProjects({ page: 1, page_size: 1000 }),
        getUsers({ page: 1, page_size: 1000 }),
      ]);
      setProjects(p.list);
      setUsers(u.list || []);
      if (p.list[0]) setProjectId(p.list[0].id);
    })();
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  useEffect(() => {
    if (!linkOpen || !projectId || !linkType || linkType === "k8s_workload") {
      setRefOptions([]);
      return;
    }
    void loadRefOptions(linkType);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [linkOpen, projectId, linkType]);

  useEffect(() => {
    if (!linkOpen || linkType !== "k8s_workload") return;
    void (async () => {
      const res = await getClusters({ page: 1, page_size: 200 });
      setClusterOptions((res.list || []).map((c: { id: number; name: string }) => ({ value: c.id, label: c.name })));
    })();
  }, [linkOpen, linkType]);

  useEffect(() => {
    if (!k8sClusterId || linkType !== "k8s_workload") {
      setNsOptions([]);
      return;
    }
    void (async () => {
      try {
        const listNs = await listNamespaces(k8sClusterId);
        const arr = Array.isArray(listNs) ? listNs : [];
        setNsOptions(arr.map((n) => ({ value: n.name, label: n.name })).filter((o) => o.value));
      } catch {
        setNsOptions([]);
      }
    })();
  }, [k8sClusterId, linkType]);

  useEffect(() => {
    if (!k8sClusterId || !k8sNamespace || !k8sKind || linkType !== "k8s_workload") {
      setWorkloadOptions([]);
      return;
    }
    void (async () => {
      try {
        let items: { name: string }[] = [];
        if (k8sKind === "Deployment") items = await listDeployments(k8sClusterId, k8sNamespace);
        else if (k8sKind === "StatefulSet") items = await listStatefulSets(k8sClusterId, k8sNamespace);
        else if (k8sKind === "DaemonSet") items = await listDaemonSets(k8sClusterId, k8sNamespace);
        setWorkloadOptions((items || []).map((w) => ({ value: w.name, label: w.name })));
      } catch {
        setWorkloadOptions([]);
      }
    })();
  }, [k8sClusterId, k8sNamespace, k8sKind, linkType]);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const [res, cicd] = await Promise.all([
        listServiceCatalog(projectId, { page: 1, page_size: 200 }),
        listCicdServices(projectId, { page: 1, page_size: 500 }).catch(() => ({ list: [] as { product_line?: string }[] })),
      ]);
      setList(res.list);
      const lines = new Set<string>();
      for (const row of res.list || []) {
        if (row.product_line?.trim()) lines.add(row.product_line.trim());
      }
      for (const row of cicd.list || []) {
        if (row.product_line?.trim()) lines.add(row.product_line.trim());
      }
      setProductLineOptions([...lines].sort());
    } finally {
      setLoading(false);
    }
  }

  async function loadRefOptions(type: string) {
    if (!projectId) return;
    setRefLoading(true);
    try {
      let opts: RefOption[] = [];
      if (type === "cicd_service") {
        const res = await listCicdServices(projectId, { page: 1, page_size: 500 });
        opts = (res.list || []).map((s: { id: number; name: string; identifier: string }) => ({
          value: s.id,
          label: `${s.name} (${s.identifier}) #${s.id}`,
        }));
      } else if (type === "cmdb_service") {
        const res = await getProjectServices(projectId, { page: 1, page_size: 500 });
        opts = (res.list || []).map((s) => ({
          value: s.id,
          label: `${s.name}${s.env ? ` (${s.env})` : ""} #${s.id}`,
        }));
      } else if (type === "log_source") {
        const res = await getProjectLogSources(projectId, { page: 1, page_size: 500 });
        opts = (res.list || []).map((s) => ({
          value: s.id,
          label: `${s.path || s.log_type} (service#${s.service_id}) #${s.id}`,
        }));
      } else if (type === "db_instance") {
        const res = await listDbInstances(projectId, { page: 1, page_size: 500 });
        opts = (res.list || []).map((s) => ({
          value: s.id,
          label: `${s.name} (${s.env}) #${s.id}`,
        }));
      } else if (type === "alert_monitor_rule") {
        const res = await listAlertMonitorRules({ page: 1, page_size: 500, project_id: projectId });
        opts = (res.list || []).map((s) => ({
          value: s.id,
          label: `${s.name || s.id} #${s.id}`,
        }));
      }
      setRefOptions(opts);
    } catch {
      setRefOptions([]);
      message.warning("加载引用列表失败，请检查对应模块权限");
    } finally {
      setRefLoading(false);
    }
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ status: 1, criticality: "normal" });
    setEditorOpen(true);
  }

  function openEdit(row: ServiceCatalogItem) {
    setCurrent(row);
    form.setFieldsValue(row);
    setEditorOpen(true);
  }

  async function onSubmit() {
    if (!projectId) return;
    const v = await form.validateFields();
    await upsertServiceCatalog(projectId, { ...v, id: current?.id });
    message.success(current ? "已更新" : "已创建");
    setEditorOpen(false);
    void load();
  }

  function openLink(row: ServiceCatalogItem) {
    setCurrent(row);
    linkForm.resetFields();
    linkForm.setFieldsValue({ link_type: "cicd_service" });
    setLinkOpen(true);
  }

  async function onLinkSubmit() {
    if (!projectId || !current) return;
    const v = await linkForm.validateFields();
    let refId = v.ref_id != null && v.ref_id !== "" ? Number(v.ref_id) : undefined;
    let refKey = (v.ref_key as string | undefined)?.trim() || undefined;

    if (v.link_type === "k8s_workload") {
      const key = `${v.k8s_cluster_id}/${v.k8s_namespace}/${v.k8s_kind}/${v.k8s_name}`;
      refKey = key;
      refId = undefined;
    }

    await addServiceCatalogLink(projectId, current.id, {
      link_type: v.link_type,
      ref_id: Number.isFinite(refId) ? refId : undefined,
      ref_key: refKey,
    });
    message.success("已绑定");
    setLinkOpen(false);
    void load();
  }

  async function onDeleteLink(row: ServiceCatalogItem, link: ServiceLinkItem) {
    if (!projectId) return;
    await deleteServiceCatalogLink(projectId, row.id, link.id);
    message.success("已解绑");
    void load();
  }

  const needsRefId = linkType && linkType !== "k8s_workload";
  const needsK8sKey = linkType === "k8s_workload";

  return (
    <Card
      title="服务目录"
      extra={
        <Space wrap>
          <Select style={{ width: 260 }} value={projectId} onChange={setProjectId} options={projectOptions} placeholder="选择项目" />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建
          </Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={false}
        columns={[
          { title: "标识", dataIndex: "identifier", width: 160 },
          { title: "名称", dataIndex: "name", width: 160 },
          { title: "负责人", dataIndex: "owner", width: 120 },
          { title: "产品线", dataIndex: "product_line", width: 120 },
          {
            title: "等级",
            dataIndex: "criticality",
            width: 100,
            render: (v: string) => <Tag>{v || "normal"}</Tag>,
          },
          {
            title: "绑定",
            dataIndex: "links",
            render: (links: ServiceLinkItem[] = [], row: ServiceCatalogItem) => (
              <Space wrap size={[4, 4]}>
                {(links || []).map((l) => (
                  <Tag
                    key={l.id}
                    closable
                    onClose={(e) => {
                      e.preventDefault();
                      void onDeleteLink(row, l);
                    }}
                  >
                    {l.link_type}:{l.ref_id ?? l.ref_key}
                  </Tag>
                ))}
              </Space>
            ),
          },
          {
            title: "操作",
            width: 220,
            render: (_: unknown, row: ServiceCatalogItem) => (
              <Space>
                <Button type="link" onClick={() => openEdit(row)}>
                  编辑
                </Button>
                <Button type="link" onClick={() => navigate(`/service-portrait?project_id=${projectId}&catalog_id=${row.id}`)}>
                  画像
                </Button>
                <Button type="link" icon={<LinkOutlined />} onClick={() => openLink(row)}>
                  绑定
                </Button>
                <Popconfirm
                  title="确认删除？"
                  onConfirm={() =>
                    projectId &&
                    deleteServiceCatalog(projectId, row.id).then(() => {
                      message.success("已删除");
                      void load();
                    })
                  }
                >
                  <Button type="link" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal title={current ? "编辑服务" : "新建服务"} open={editorOpen} onOk={() => void onSubmit()} onCancel={() => setEditorOpen(false)} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="identifier" label="标识" rules={[{ required: true }]} extra="建议与 CI/CD 应用 identifier 一致，作为跨模块统一服务名">
            <Input disabled={!!current} placeholder="如 springbootDemo" />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="owner" label="负责人" extra="从平台用户选择，保存为用户名">
            <Select showSearch allowClear placeholder="选择负责人" optionFilterProp="label" options={userOptions} />
          </Form.Item>
          <Form.Item name="product_line" label="产品线" extra="可从已有产品线选择，也可直接输入新名称">
            <AutoComplete
              allowClear
              placeholder="选择或输入产品线"
              options={productLineOptions.map((v) => ({ value: v, label: v }))}
              filterOption={(input, option) =>
                String(option?.value || "")
                  .toLowerCase()
                  .includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="criticality" label="关键等级">
            <Select options={[{ value: "critical" }, { value: "high" }, { value: "normal" }, { value: "low" }]} />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={`绑定关联：${current?.name || ""}`} open={linkOpen} onOk={() => void onLinkSubmit()} onCancel={() => setLinkOpen(false)} destroyOnClose width={560}>
        <Form form={linkForm} layout="vertical">
          <Form.Item name="link_type" label="类型" rules={[{ required: true }]} extra="把该业务服务挂到具体发布/日志/K8s/库表等实体上">
            <Select
              options={LINK_TYPE_OPTIONS}
              onChange={() => {
                linkForm.setFieldsValue({
                  ref_id: undefined,
                  ref_key: undefined,
                  k8s_cluster_id: undefined,
                  k8s_namespace: undefined,
                  k8s_kind: undefined,
                  k8s_name: undefined,
                });
              }}
            />
          </Form.Item>

          {needsRefId ? (
            <Form.Item name="ref_id" label="引用对象" rules={[{ required: true, message: "请选择要绑定的对象" }]}>
              <Select
                showSearch
                allowClear
                loading={refLoading}
                placeholder="从列表选择"
                optionFilterProp="label"
                options={refOptions}
              />
            </Form.Item>
          ) : null}

          {needsK8sKey ? (
            <>
              <Form.Item name="k8s_cluster_id" label="集群" rules={[{ required: true }]}>
                <Select
                  showSearch
                  options={clusterOptions}
                  optionFilterProp="label"
                  onChange={() =>
                    linkForm.setFieldsValue({ k8s_namespace: undefined, k8s_kind: undefined, k8s_name: undefined })
                  }
                />
              </Form.Item>
              <Form.Item name="k8s_namespace" label="命名空间" rules={[{ required: true }]}>
                <Select
                  showSearch
                  options={nsOptions}
                  optionFilterProp="label"
                  onChange={() => linkForm.setFieldsValue({ k8s_name: undefined })}
                />
              </Form.Item>
              <Form.Item name="k8s_kind" label="工作负载类型" rules={[{ required: true }]}>
                <Select options={K8S_KIND_OPTIONS} onChange={() => linkForm.setFieldsValue({ k8s_name: undefined })} />
              </Form.Item>
              <Form.Item name="k8s_name" label="工作负载名称" rules={[{ required: true }]} extra="将保存为 clusterId/ns/kind/name">
                <Select showSearch options={workloadOptions} optionFilterProp="label" />
              </Form.Item>
            </>
          ) : null}
        </Form>
      </Modal>
    </Card>
  );
}
