// @ts-nocheck
import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Typography, message } from "antd";
import { useCallback, useEffect, useState } from "react";
import { MonacoYamlEditor } from "../components/k8s/monaco-yaml-editor";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import {
  createK8sCrTemplate,
  deleteK8sCrTemplate,
  listK8sCrTemplates,
  updateK8sCrTemplate,
  type K8sCrTemplateItem,
} from "../services/k8s-cr-templates";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

export function K8sCrTemplatesPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>(0);
  const [kindFilter, setKindFilter] = useState<string>("");
  const [list, setList] = useState<K8sCrTemplateItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [current, setCurrent] = useState<K8sCrTemplateItem | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((r) => setProjects(r.list || []));
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setList(
        await listK8sCrTemplates({
          project_id: projectId > 0 ? projectId : undefined,
          kind: kindFilter || undefined,
        }),
      );
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [projectId, kindFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({
      project_id: projectId > 0 ? projectId : 0,
      gvk_version: "v1",
      sort_order: 0,
      body: `apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
  namespace: default
data:
  key: value
`,
    });
    setOpen(true);
  }

  function openEdit(row: K8sCrTemplateItem) {
    setCurrent(row);
    form.setFieldsValue({
      project_id: row.project_id,
      name: row.name,
      gvk_group: row.gvk_group,
      gvk_version: row.gvk_version || "v1",
      gvk_kind: row.gvk_kind,
      body: row.body,
      sort_order: row.sort_order,
    });
    setOpen(true);
  }

  async function onSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      const payload = {
        project_id: values.project_id ?? 0,
        name: values.name,
        gvk_group: values.gvk_group,
        gvk_version: values.gvk_version || "v1",
        gvk_kind: values.gvk_kind,
        body: values.body,
        sort_order: values.sort_order ?? 0,
      };
      if (current) {
        await updateK8sCrTemplate(current.id, payload);
        message.success("已更新模板");
      } else {
        await createK8sCrTemplate(payload);
        message.success("已创建模板");
      }
      setOpen(false);
      await load();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "保存失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
  <>
    <OpsPageHeader
      title="K8s CR/YAML 模板库"
      description="预置常用 CR 清单，可在自定义资源页复制 body 后应用；project_id=0 为全局模板。"
      extra={
        <Space>
          <Select
            allowClear
            placeholder="项目筛选"
            style={{ width: 200 }}
            value={projectId > 0 ? projectId : undefined}
            onChange={(v) => setProjectId(v ?? 0)}
            options={[
              { label: "全部（含全局）", value: 0 },
              ...projects.map((p) => ({ label: p.name, value: p.id })),
            ]}
          />
          <Input
            allowClear
            placeholder="Kind 筛选"
            style={{ width: 140 }}
            value={kindFilter}
            onChange={(e) => setKindFilter(e.target.value)}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} />
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建模板
          </Button>
        </Space>
      }
    />
    <Card className="table-card">
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{ pageSize: 15, showTotal: (t) => `共 ${t} 条` }}
        columns={[
          { title: "名称", dataIndex: "name", width: 180, ellipsis: true },
          {
            title: "项目",
            dataIndex: "project_id",
            width: 100,
            render: (v: number) => (v === 0 ? <Tag>全局</Tag> : v),
          },
          { title: "Kind", dataIndex: "gvk_kind", width: 120 },
          { title: "Group", dataIndex: "gvk_group", width: 160, ellipsis: true, render: (v?: string) => v || "—" },
          { title: "Version", dataIndex: "gvk_version", width: 80 },
          { title: "排序", dataIndex: "sort_order", width: 70 },
          { title: "创建时间", dataIndex: "created_at", width: 180, render: (v: string) => formatDateTime(v) },
          {
            title: "操作",
            width: 160,
            render: (_: unknown, row: K8sCrTemplateItem) => (
              <Space>
                <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(row)}>
                  编辑
                </Button>
                <Popconfirm title="删除该模板？" onConfirm={() => void deleteK8sCrTemplate(row.id).then(load)}>
                  <Button type="link" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
    </Card>

    <Modal
      title={current ? "编辑 CR 模板" : "新建 CR 模板"}
      open={open}
      onCancel={() => setOpen(false)}
      onOk={() => void onSubmit()}
      confirmLoading={submitting}
      width={880}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Space wrap style={{ width: "100%" }}>
          <Form.Item label="归属项目" name="project_id" style={{ width: 220 }}>
            <Select
              options={[
                { label: "全局 (0)", value: 0 },
                ...projects.map((p) => ({ label: p.name, value: p.id })),
              ]}
            />
          </Form.Item>
          <Form.Item label="名称" name="name" rules={[{ required: true }]} style={{ width: 220 }}>
            <Input />
          </Form.Item>
          <Form.Item label="Kind" name="gvk_kind" rules={[{ required: true }]} style={{ width: 160 }}>
            <Input placeholder="ConfigMap" />
          </Form.Item>
          <Form.Item label="Group" name="gvk_group" style={{ width: 200 }}>
            <Input placeholder="可选" allowClear />
          </Form.Item>
          <Form.Item label="Version" name="gvk_version" style={{ width: 100 }}>
            <Input />
          </Form.Item>
          <Form.Item label="排序" name="sort_order" style={{ width: 100 }}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Space>
        <Form.Item label="YAML 正文" name="body" rules={[{ required: true }]}>
          <MonacoYamlEditor height={360} />
        </Form.Item>
        <Typography.Text type="secondary">保存后可在「自定义资源」页复制应用；不会自动下发到集群。</Typography.Text>
      </Form>
    </Modal>
  </>
  );
}
