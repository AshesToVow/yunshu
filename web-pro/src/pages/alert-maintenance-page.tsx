// @ts-nocheck
import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, DatePicker, Form, Input, Modal, Popconfirm, Select, Space, Switch, Table, Tag, message } from "antd";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import {
  createMaintenanceWindow,
  deleteMaintenanceWindow,
  listMaintenanceWindows,
  updateMaintenanceWindow,
  type AlertMaintenanceWindowItem,
} from "../services/alert-maintenance";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

type MatcherForm = { name: string; value: string; is_regex: boolean };

function matchersToJSON(rows: MatcherForm[]) {
  const items = rows
    .map((r) => ({
      name: String(r.name || "").trim(),
      value: String(r.value || "").trim(),
      is_regex: Boolean(r.is_regex),
    }))
    .filter((r) => r.name !== "");
  return JSON.stringify(items);
}

export function AlertMaintenancePage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [keyword, setKeyword] = useState("");
  const [list, setList] = useState<AlertMaintenanceWindowItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [current, setCurrent] = useState<AlertMaintenanceWindowItem | null>(null);
  const [form] = Form.useForm();

  const projectOptions = useMemo(() => projects.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id })), [projects]);

  async function loadProjects() {
    const res = await getProjects({ page: 1, page_size: 500 });
    setProjects(res.list ?? []);
  }

  async function load(nextPage = page, nextSize = pageSize) {
    setLoading(true);
    try {
      const res = await listMaintenanceWindows({
        projectId: projectId,
        keyword: keyword.trim() || undefined,
        page: nextPage,
        page_size: nextSize,
      });
      setList(res.list ?? []);
      setTotal(res.total ?? 0);
      setPage(res.page ?? nextPage);
      setPageSize(res.page_size ?? nextSize);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadProjects();
  }, []);

  useEffect(() => {
    void load(1, pageSize);
  }, [projectId, keyword]);

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({
      name: "",
      project_id: projectId,
      matchers: [{ name: "alertname", value: "", is_regex: false }],
      comment: "",
      enabled: true,
      starts_at: dayjs(),
      ends_at: dayjs().add(4, "hour"),
    });
    setOpen(true);
  }

  function openEdit(row: AlertMaintenanceWindowItem) {
    setCurrent(row);
    form.setFieldsValue({
      name: row.name,
      project_id: row.project_id || undefined,
      matchers: row.matchers?.length ? row.matchers : [{ name: "alertname", value: "", is_regex: false }],
      comment: row.comment || "",
      enabled: row.enabled,
      starts_at: dayjs(row.starts_at),
      ends_at: dayjs(row.ends_at),
    });
    setOpen(true);
  }

  async function submit() {
    const v = await form.validateFields();
    const matchers = v.matchers as MatcherForm[];
    const payload = {
      name: String(v.name || "").trim(),
      matchers_json: matchersToJSON(matchers),
      starts_at: (v.starts_at as dayjs.Dayjs).toISOString(),
      ends_at: (v.ends_at as dayjs.Dayjs).toISOString(),
      comment: String(v.comment || "").trim(),
      project_id: Number(v.project_id || 0),
      enabled: Boolean(v.enabled),
    };
    setSaving(true);
    try {
      if (current) {
        await updateMaintenanceWindow(current.id, payload);
        message.success("维护窗口已更新");
      } else {
        await createMaintenanceWindow(payload);
        message.success("维护窗口已创建");
      }
      setOpen(false);
      await load(page, pageSize);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card
      className="table-card"
      title="告警维护窗口"
      extra={
        <Space wrap>
          <Select allowClear style={{ width: 220 }} placeholder="按项目筛选" options={projectOptions} value={projectId} onChange={setProjectId} />
          <Input.Search allowClear style={{ width: 240 }} placeholder="名称/说明" onSearch={setKeyword} />
          <Button icon={<ReloadOutlined />} onClick={() => void load(page, pageSize)}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建维护窗口
          </Button>
        </Space>
      }
    >
      <Alert type="info" showIcon style={{ marginBottom: 12 }} message="维护窗口内匹配的告警将被抑制投递（与平台静默独立，常用于计划内变更）。" />
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        scroll={{ x: 1200 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (p, ps) => void load(p, ps),
        }}
        columns={[
          { title: "ID", dataIndex: "id", width: 70 },
          { title: "名称", dataIndex: "name", width: 180 },
          {
            title: "项目",
            width: 120,
            render: (_: unknown, r: AlertMaintenanceWindowItem) => (r.project_id ? `#${r.project_id}` : "全局"),
          },
          {
            title: "匹配器",
            key: "matchers",
            ellipsis: true,
            render: (_: unknown, r: AlertMaintenanceWindowItem) =>
              r.matchers?.map((m) => `${m.name}=${m.value}`).join(", ") || r.matchers_json?.slice(0, 80) || "—",
          },
          { title: "开始", dataIndex: "starts_at", width: 170, render: (t: string) => formatDateTime(t) },
          { title: "结束", dataIndex: "ends_at", width: 170, render: (t: string) => formatDateTime(t) },
          {
            title: "状态",
            width: 100,
            render: (_: unknown, r: AlertMaintenanceWindowItem) => {
              const now = dayjs();
              const active = r.enabled && !dayjs(r.ends_at).isBefore(now) && !dayjs(r.starts_at).isAfter(now);
              if (!r.enabled) return <Tag>停用</Tag>;
              if (dayjs(r.ends_at).isBefore(now)) return <Tag color="default">已结束</Tag>;
              if (dayjs(r.starts_at).isAfter(now)) return <Tag color="blue">未开始</Tag>;
              return active ? <Tag color="green">生效中</Tag> : <Tag>—</Tag>;
            },
          },
          {
            title: "操作",
            width: 160,
            fixed: "right",
            render: (_: unknown, r: AlertMaintenanceWindowItem) => (
              <Space>
                <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(r)}>
                  编辑
                </Button>
                <Popconfirm title="删除该维护窗口？" onConfirm={() => void deleteMaintenanceWindow(r.id).then(() => load(page, pageSize))}>
                  <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal title={current ? "编辑维护窗口" : "新建维护窗口"} open={open} onCancel={() => setOpen(false)} onOk={() => void submit()} confirmLoading={saving} width={640}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="project_id" label="项目（0 或留空表示全局）">
            <Select allowClear options={projectOptions} placeholder="可选" />
          </Form.Item>
          <Form.List name="matchers">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field) => (
                  <Space key={field.key} align="baseline" style={{ display: "flex", marginBottom: 8 }}>
                    <Form.Item {...field} name={[field.name, "name"]} rules={[{ required: true, message: "label" }]}>
                      <Input placeholder="label" style={{ width: 140 }} />
                    </Form.Item>
                    <Form.Item {...field} name={[field.name, "value"]}>
                      <Input placeholder="value" style={{ width: 180 }} />
                    </Form.Item>
                    <Form.Item {...field} name={[field.name, "is_regex"]} valuePropName="checked">
                      <Switch checkedChildren="正则" unCheckedChildren="精确" />
                    </Form.Item>
                    <Button type="link" danger onClick={() => remove(field.name)}>
                      删除
                    </Button>
                  </Space>
                ))}
                <Button type="dashed" onClick={() => add({ name: "", value: "", is_regex: false })} block>
                  添加匹配器
                </Button>
              </>
            )}
          </Form.List>
          <Form.Item name="starts_at" label="开始时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="ends_at" label="结束时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="comment" label="说明">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
