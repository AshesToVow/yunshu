// @ts-nocheck
import { PageContainer } from "@ant-design/pro-components";
import { DeleteOutlined, EditOutlined, EyeOutlined, FileAddOutlined, ReloadOutlined, SnippetsOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Empty, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, TreeSelect, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import { K8sDeleteDialog } from "@/components/k8s/k8s-delete-dialog";
import { getClusters, listNamespaces as listClusterNamespaces, type ClusterItem } from "@/services/clusters";
import type { K8sDeleteOptions } from "@/services/service-factory";
import { applyCr, deleteCr, getCrDetail, listCrResources, listCrs, type CrDetail, type CrItem, type CrResourceItem } from "@/services/crs";
import { listK8sCrTemplates, type K8sCrTemplateItem } from "@/services/k8s-cr-templates";
import { Link } from '@umijs/max';

function materializeCrTemplateBody(
  body: string,
  opts: {
    namespace?: string;
    apiVersion?: string;
    kind?: string;
  },
): string {
  let out = body.trim();
  if (opts.apiVersion) {
    out = out.replace(/^apiVersion:\s*.*$/m, `apiVersion: ${opts.apiVersion}`);
  }
  if (opts.kind) {
    out = out.replace(/^kind:\s*.*$/m, `kind: ${opts.kind}`);
  }
  if (opts.namespace) {
    if (/^\s*namespace:\s*.+$/m.test(out)) {
      out = out.replace(/^\s*namespace:\s*.*$/m, `  namespace: ${opts.namespace}`);
    } else if (/^metadata:\s*$/m.test(out)) {
      out = out.replace(/^metadata:\s*$/m, `metadata:\n  namespace: ${opts.namespace}`);
    } else if (/^metadata:\s*\n/m.test(out)) {
      out = out.replace(/^(metadata:\s*\n)/m, `$1  namespace: ${opts.namespace}\n`);
    }
  }
  return out;
}

function templateMatchesResource(tpl: K8sCrTemplateItem, res?: CrResourceItem): boolean {
  if (!res) return true;
  if (!tpl.gvk_kind || !res.kind) return false;
  if (tpl.gvk_kind.toLowerCase() !== res.kind.toLowerCase()) return false;
  const g = (tpl.gvk_group || "").trim();
  const rg = (res.group || "").trim();
  if (g && rg && g.toLowerCase() !== rg.toLowerCase()) return false;
  return true;
}

export default function CrsPage() {
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number>();
  const [resources, setResources] = useState<CrResourceItem[]>([]);
  const [selectedResourceName, setSelectedResourceName] = useState<string>();
  const [namespaces, setNamespaces] = useState<Array<{ label: string; value: string }>>([]);
  const [namespace, setNamespace] = useState<string>("default");
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<CrItem[]>([]);

  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailName, setDetailName] = useState("");
  const [detail, setDetail] = useState<CrDetail | null>(null);
  const [detailYaml, setDetailYaml] = useState("");
  const [detailSubmitting, setDetailSubmitting] = useState(false);

  const [applyOpen, setApplyOpen] = useState(false);
  const [applyLoading, setApplyLoading] = useState(false);
  const [manifest, setManifest] = useState("");
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteTargetName, setDeleteTargetName] = useState("");
  const [deleteLoading, setDeleteLoading] = useState(false);

  const [templateOpen, setTemplateOpen] = useState(false);
  const [templateLoading, setTemplateLoading] = useState(false);
  const [templates, setTemplates] = useState<K8sCrTemplateItem[]>([]);
  const [templateKeyword, setTemplateKeyword] = useState("");

  const currentCluster = useMemo(() => clusters.find((c) => c.id === clusterId), [clusters, clusterId]);

  const selectedResource = useMemo(
    () => resources.find((r) => r.name === selectedResourceName),
    [resources, selectedResourceName],
  );
  const resourceTree = useMemo(() => {
    const groups = new Map<string, CrResourceItem[]>();
    for (const r of resources) {
      const k = r.group || "core";
      if (!groups.has(k)) groups.set(k, []);
      groups.get(k)?.push(r);
    }
    return Array.from(groups.entries()).map(([group, list]) => ({
      title: group,
      value: `group:${group}`,
      selectable: false,
      children: list.map((r) => ({
        title: `${r.kind} (${r.version}) - ${r.resource}`,
        value: r.name,
      })),
    }));
  }, [resources]);

  const filteredTemplates = useMemo(() => {
    const kw = templateKeyword.trim().toLowerCase();
    return templates
      .filter((t) => templateMatchesResource(t, selectedResource))
      .filter((t) => {
        if (!kw) return true;
        return (
          t.name.toLowerCase().includes(kw) ||
          t.gvk_kind.toLowerCase().includes(kw) ||
          (t.gvk_group || "").toLowerCase().includes(kw)
        );
      });
  }, [templates, selectedResource, templateKeyword]);

  const columns: ColumnsType<CrItem> = [
    { title: "名称", dataIndex: "name", width: 220 },
    { title: "命名空间", dataIndex: "namespace", width: 140, render: (v?: string) => v || "-" },
    { title: "APIVersion", dataIndex: "api_version", width: 220 },
    { title: "Kind", dataIndex: "kind", width: 180 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
    {
      title: "操作",
      key: "action",
      width: 240,
      fixed: "right",
      render: (_: unknown, record: CrItem) => (
        <Space>
          <Button type="link" icon={<EyeOutlined />} onClick={() => void openDetail(record.name)}>
            详情
          </Button>
          <Button type="link" icon={<EditOutlined />} onClick={() => void openEdit(record.name)}>
            编辑
          </Button>
          <Button
            danger
            type="link"
            icon={<DeleteOutlined />}
            onClick={() => {
              setDeleteTargetName(record.name);
              setDeleteDialogOpen(true);
            }}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  async function loadClusters() {
    const res = await getClusters({ page: 1, page_size: 200 });
    const list = res.list ?? [];
    setClusters(list);
    if (!clusterId) {
      const first = list.find((c) => c.status === 1);
      if (first) setClusterId(first.id);
    }
  }

  async function loadResources(cid: number) {
    const list = await listCrResources(cid);
    setResources(list);
    if (!list.some((x) => x.name === selectedResourceName)) {
      setSelectedResourceName(list[0]?.name);
    }
  }

  async function loadNamespaces(cid: number) {
    const res = await listClusterNamespaces(cid);
    const opts = (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
    setNamespaces(opts);
    if (!opts.some((o) => o.value === namespace)) {
      setNamespace(opts[0]?.value ?? "default");
    }
  }

  async function reload() {
    if (!clusterId || !selectedResource) return;
    if (selectedResource.namespaced && !namespace) return;
    setLoading(true);
    try {
      const list = await listCrs({
        clusterId,
        group: selectedResource.group,
        version: selectedResource.version,
        resource: selectedResource.resource,
        namespace: selectedResource.namespaced ? namespace : undefined,
        keyword: keyword.trim() || undefined,
      });
      setData(list ?? []);
    } finally {
      setLoading(false);
    }
  }

  async function openDetail(name: string) {
    if (!clusterId || !selectedResource) return;
    setDetailOpen(true);
    setDetailLoading(true);
    setDetailName(name);
    setDetail(null);
    try {
      const d = await getCrDetail({
        clusterId,
        group: selectedResource.group,
        version: selectedResource.version,
        resource: selectedResource.resource,
        namespace: selectedResource.namespaced ? namespace : undefined,
        name,
      });
      setDetail(d);
      setDetailYaml(d.yaml ?? "");
    } finally {
      setDetailLoading(false);
    }
  }

  async function openTemplatePicker() {
    setTemplateOpen(true);
    setTemplateLoading(true);
    setTemplateKeyword("");
    try {
      const pid = currentCluster?.owning_project_id;
      const list = await listK8sCrTemplates({
        project_id: pid && pid > 0 ? pid : undefined,
        kind: selectedResource?.kind || undefined,
      });
      setTemplates(list);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载模板失败");
      setTemplates([]);
    } finally {
      setTemplateLoading(false);
    }
  }

  function useTemplate(tpl: K8sCrTemplateItem) {
    const apiVersion = selectedResource
      ? `${selectedResource.group}/${selectedResource.version}`
      : tpl.gvk_group
        ? `${tpl.gvk_group}/${tpl.gvk_version || "v1"}`
        : tpl.gvk_version || "v1";
    const kind = selectedResource?.kind || tpl.gvk_kind;
    const ns = selectedResource?.namespaced ? namespace || "default" : undefined;
    setManifest(
      materializeCrTemplateBody(tpl.body, {
        namespace: ns,
        apiVersion,
        kind,
      }),
    );
    setTemplateOpen(false);
    setApplyOpen(true);
    message.success(`已载入模板「${tpl.name}」，请确认 YAML 后应用`);
  }

  async function openEdit(name: string) {
    if (!clusterId || !selectedResource) return;
    setApplyOpen(true);
    setApplyLoading(true);
    try {
      const d = await getCrDetail({
        clusterId,
        group: selectedResource.group,
        version: selectedResource.version,
        resource: selectedResource.resource,
        namespace: selectedResource.namespaced ? namespace : undefined,
        name,
      });
      setManifest(d.yaml ?? "");
    } finally {
      setApplyLoading(false);
    }
  }

  async function doDelete(name: string, deleteOpts?: K8sDeleteOptions) {
    if (!clusterId || !selectedResource) return;
    await deleteCr({
      clusterId,
      group: selectedResource.group,
      version: selectedResource.version,
      resource: selectedResource.resource,
      namespace: selectedResource.namespaced ? namespace : undefined,
      name,
      ...deleteOpts,
    });
    message.success("删除成功");
    await reload();
  }

  useEffect(() => {
    void loadClusters();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!clusterId) return;
    void (async () => {
      await Promise.all([loadResources(clusterId), loadNamespaces(clusterId)]);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId]);

  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, selectedResourceName, namespace]);

  return (
    <PageContainer header={{ title: "CR 实例管理", subTitle: "自定义资源实例列表与 YAML 应用" }}>
    <Card className="table-card" bordered={false}>
      <div className="ops-filter-bar" style={{ display: "flex", justifyContent: "space-between", marginBottom: 12, gap: 12 }}>
        <Space wrap>
          <Select
            placeholder="选择集群"
            style={{ minWidth: 220 }}
            value={clusterId}
            onChange={setClusterId}
            options={clusters.map((c) => ({
              label: c.status === 1 ? c.name : `${c.name}（已停用）`,
              value: c.id,
              disabled: c.status !== 1,
            }))}
          />
          <TreeSelect
            placeholder="选择 CR 类型（按 Group/Kind）"
            style={{ minWidth: 200, maxWidth: "min(420px, 100%)" }}
            value={selectedResourceName}
            onChange={(v) => setSelectedResourceName(String(v || ""))}
            treeData={resourceTree}
            showSearch
            treeDefaultExpandAll
            allowClear
          />
          {selectedResource?.namespaced ? (
            <Select
              placeholder="命名空间"
              style={{ minWidth: 180 }}
              value={namespace}
              onChange={setNamespace}
              options={namespaces}
              showSearch
              optionFilterProp="label"
            />
          ) : null}
          <Input.Search
            allowClear
            placeholder="搜索名称"
            style={{ width: 240 }}
            onSearch={(v) => {
              setKeyword(v);
              void reload();
            }}
          />
        </Space>
        <Space>
          <Button icon={<SnippetsOutlined />} onClick={() => void openTemplatePicker()}>
            从模板创建
          </Button>
          <Button
            icon={<FileAddOutlined />}
            onClick={() => {
              const defaultNs = selectedResource?.namespaced ? `  namespace: ${namespace || "default"}\n` : "";
              const apiVersion = selectedResource ? `${selectedResource.group}/${selectedResource.version}` : "example.com/v1";
              const kind = selectedResource?.kind || "Example";
              setManifest(`apiVersion: ${apiVersion}
kind: ${kind}
metadata:
  name: demo-${kind.toLowerCase()}
${defaultNs}spec: {}
`);
              setApplyOpen(true);
            }}
          >
            快捷创建
          </Button>
          <Button
            type="primary"
            icon={<FileAddOutlined />}
            onClick={() => {
              setManifest("");
              setApplyOpen(true);
            }}
          >
            应用 YAML
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void reload()}>
            刷新
          </Button>
        </Space>
      </div>

      <div style={{ marginBottom: 8 }}>
        {selectedResource ? (
          <Space>
            <Tag color="blue">{selectedResource.kind}</Tag>
            <Tag>{selectedResource.group}</Tag>
            <Tag>{selectedResource.version}</Tag>
            <Tag>{selectedResource.resource}</Tag>
            <Tag color={selectedResource.namespaced ? "green" : "purple"}>
              {selectedResource.namespaced ? "Namespaced" : "Cluster"}
            </Tag>
          </Space>
        ) : null}
      </div>

      <Table<CrItem>
        rowKey={(r) => `${r.namespace || "_cluster_"}-${r.name}`}
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
        scroll={{ x: "max-content" }}
      />

      <Drawer
        title={`详情 - ${detailName}`}
        open={detailOpen}
        width={980}
        onClose={() => setDetailOpen(false)}
        className="detail-edit-drawer"
        extra={
          <Space>
            <Button
              icon={<EditOutlined />}
              onClick={() => {
                setManifest(detail?.yaml ?? "");
                setApplyOpen(true);
              }}
            >
              基于详情编辑
            </Button>
            <Button
              type="primary"
              loading={detailSubmitting}
              onClick={() => {
                if (!clusterId) {
                  message.warning("请先选择集群");
                  return;
                }
                setDetailSubmitting(true);
                void (async () => {
                  try {
                    await applyCr(clusterId, detailYaml);
                    message.success("详情修改已保存");
                    const latest = await getCrDetail({
                      clusterId,
                      group: selectedResource?.group || "",
                      version: selectedResource?.version || "",
                      resource: selectedResource?.resource || "",
                      namespace: selectedResource?.namespaced ? namespace : undefined,
                      name: detailName,
                    });
                    setDetail(latest);
                    setDetailYaml(latest.yaml ?? "");
                    await reload();
                  } finally {
                    setDetailSubmitting(false);
                  }
                })();
              }}
            >
              保存修改
            </Button>
          </Space>
        }
      >
        {detailLoading ? (
          <Typography.Text type="secondary">加载中...</Typography.Text>
        ) : (
          <Form layout="vertical">
            <Form.Item label="资源名称">
              <Input value={detailName} readOnly />
            </Form.Item>
            <Form.Item label="资源类型">
              <Input value={selectedResource ? `${selectedResource.kind} (${selectedResource.group}/${selectedResource.version})` : "-"} readOnly />
            </Form.Item>
            <Form.Item label="命名空间">
              <Input value={selectedResource?.namespaced ? namespace : "Cluster Scope"} readOnly />
            </Form.Item>
            <Form.Item label="YAML">
              <Input.TextArea value={detailYaml} onChange={(e) => setDetailYaml(e.target.value)} autoSize={{ minRows: 20, maxRows: 28 }} />
            </Form.Item>
          </Form>
        )}
      </Drawer>

      <Modal
        title="应用 YAML"
        open={applyOpen}
        width={980}
        confirmLoading={applyLoading}
        onCancel={() => setApplyOpen(false)}
        onOk={() => {
          if (!clusterId) {
            message.warning("请先选择集群");
            return;
          }
          setApplyLoading(true);
          void (async () => {
            try {
              await applyCr(clusterId, manifest);
              message.success("应用成功");
              setApplyOpen(false);
              await reload();
            } finally {
              setApplyLoading(false);
            }
          })();
        }}
      >
        <Input.TextArea value={manifest} onChange={(e) => setManifest(e.target.value)} autoSize={{ minRows: 20, maxRows: 28 }} />
      </Modal>

      <K8sDeleteDialog
        open={deleteDialogOpen}
        resourceName={deleteTargetName}
        loading={deleteLoading}
        onCancel={() => {
          setDeleteDialogOpen(false);
          setDeleteTargetName("");
        }}
        onConfirm={async (deleteOpts) => {
          if (!deleteTargetName) return;
          setDeleteLoading(true);
          try {
            await doDelete(deleteTargetName, deleteOpts);
            setDeleteDialogOpen(false);
            setDeleteTargetName("");
          } finally {
            setDeleteLoading(false);
          }
        }}
      />

      <Modal
        title="从模板创建 CR"
        open={templateOpen}
        onCancel={() => setTemplateOpen(false)}
        footer={null}
        width={880}
        destroyOnClose
      >
        <Space direction="vertical" style={{ width: "100%" }} size="middle">
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {selectedResource
              ? `已按当前类型筛选：${selectedResource.kind}（${selectedResource.group}/${selectedResource.version}）`
              : "未选择 CR 类型时将展示全部模板；建议先选择左侧 CR 类型"}
            {currentCluster?.owning_project_id ? (
              <>
                {" "}
                · 含项目 #{currentCluster.owning_project_id} 与全局模板
              </>
            ) : (
              " · 含全局模板"
            )}
            <Link to="/k8s-cr-templates" style={{ marginLeft: 8 }}>管理模板库</Link>
          </Typography.Paragraph>
          <Input.Search
            allowClear
            placeholder="搜索模板名称 / Kind / Group"
            onSearch={(v) => setTemplateKeyword(v)}
            onChange={(e) => setTemplateKeyword(e.target.value)}
          />
          <Table<K8sCrTemplateItem>
            rowKey="id"
            size="small"
            loading={templateLoading}
            dataSource={filteredTemplates}
            pagination={{ pageSize: 8, showSizeChanger: false }}
            locale={{
              emptyText: (
                <Empty description="无匹配模板">
                  <Link to="/k8s-cr-templates">去模板库新建</Link>
                </Empty>
              ),
            }}
            columns={[
              { title: "名称", dataIndex: "name", width: 160, ellipsis: true },
              { title: "Kind", dataIndex: "gvk_kind", width: 120 },
              { title: "Group", dataIndex: "gvk_group", width: 160, ellipsis: true, render: (v?: string) => v || "—" },
              {
                title: "归属",
                dataIndex: "project_id",
                width: 80,
                render: (v: number) => (v === 0 ? <Tag>全局</Tag> : `项目 ${v}`),
              },
              {
                title: "摘要",
                key: "body",
                ellipsis: true,
                render: (_: unknown, r: K8sCrTemplateItem) => (
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {r.body.split("\n").slice(0, 2).join(" · ")}
                  </Typography.Text>
                ),
              },
              {
                title: "操作",
                width: 90,
                render: (_: unknown, r: K8sCrTemplateItem) => (
                  <Button type="link" size="small" onClick={() => useTemplate(r)}>
                    使用
                  </Button>
                ),
              },
            ]}
          />
        </Space>
      </Modal>
    </Card>
    </PageContainer>
  );
}