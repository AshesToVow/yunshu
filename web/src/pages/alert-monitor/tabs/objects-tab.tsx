import { extractApiErrorMessage } from "../../../services/http";
import { Alert, Button, Form, Input, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from "antd";
import { PlusOutlined, ReloadOutlined, SyncOutlined } from "@ant-design/icons";
import { useCallback, useEffect, useState } from "react";
import { useAlertMonitor } from "../context";
import {
  createConsulEndpoint,
  deleteConsulEndpoint,
  listConsulEndpoints,
  listMonitorObjects,
  pingConsulEndpoint,
  syncConsulEndpoint,
  updateConsulEndpoint,
  type AlertConsulEndpointItem,
  type AlertMonitorObjectItem,
} from "../../../services/alert-platform";
import { formatDateTime } from "../../../utils/format";
import { DEFAULT_PAGE_SIZE, tablePagination } from "../../../utils/table-pagination";

export function ObjectsTab() {
  const ctx = useAlertMonitor();
  const [eps, setEps] = useState<AlertConsulEndpointItem[]>([]);
  const [objs, setObjs] = useState<AlertMonitorObjectItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [objLoading, setObjLoading] = useState(false);
  const [objPage, setObjPage] = useState(1);
  const [objPageSize, setObjPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [objTotal, setObjTotal] = useState(0);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<AlertConsulEndpointItem | null>(null);
  const [form] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await listConsulEndpoints({
        project_id: ctx.projectContextId || undefined,
        page: 1,
        page_size: 100,
      });
      setEps(r.list ?? []);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载 Consul 端点失败"));
    } finally {
      setLoading(false);
    }
  }, [ctx.projectContextId]);

  const loadObjects = useCallback(async (page: number, pageSize: number) => {
    setObjLoading(true);
    try {
      const r = await listMonitorObjects({
        project_id: ctx.projectContextId || undefined,
        page,
        page_size: pageSize,
      });
      setObjs(r.list ?? []);
      setObjTotal(Number(r.total) || 0);
      setObjPage(r.page ?? page);
      setObjPageSize(r.page_size ?? pageSize);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载监控对象失败"));
    } finally {
      setObjLoading(false);
    }
  }, [ctx.projectContextId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    void loadObjects(1, DEFAULT_PAGE_SIZE);
  }, [loadObjects]);

  function openCreate() {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      project_id: ctx.projectContextId || undefined,
      service_tag: "yunshu-metrics",
      enabled: true,
    });
    setModalOpen(true);
  }

  function openEdit(row: AlertConsulEndpointItem) {
    setEditing(row);
    form.setFieldsValue({
      project_id: row.project_id,
      name: row.name,
      address: row.address,
      datacenter: row.datacenter,
      service_tag: row.service_tag || "yunshu-metrics",
      enabled: row.enabled,
      remark: row.remark,
      token: "",
    });
    setModalOpen(true);
  }

  async function submit() {
    try {
      const v = await form.validateFields();
      const payload = {
        project_id: Number(v.project_id),
        name: v.name,
        address: v.address,
        datacenter: v.datacenter || "",
        service_tag: v.service_tag || "yunshu-metrics",
        enabled: !!v.enabled,
        remark: v.remark || "",
        token: v.token || "",
        clear_token: !!v.clear_token,
      };
      if (editing) {
        await updateConsulEndpoint(editing.id, payload);
        message.success("已更新");
      } else {
        await createConsulEndpoint(payload);
        message.success("已创建");
      }
      setModalOpen(false);
      await load();
    } catch (e) {
      if (e && typeof e === "object" && "errorFields" in e) return;
      message.error(extractApiErrorMessage(e, "保存失败"));
    }
  }

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Alert
        type="info"
        showIcon
        message="监控对象（Consul 目录）"
        description={
          <span>
            从 Consul 同步带 <Typography.Text code>yunshu-metrics</Typography.Text> tag 的服务实例，供规则与项目对照。
            scrape 仍由 Prometheus Consul SD 完成；样例见 <Typography.Text code>deploy/monitoring/</Typography.Text>。
          </span>
        }
      />
      <Space wrap>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建 Consul 端点
        </Button>
        <Button
          icon={<ReloadOutlined />}
          onClick={() => void Promise.all([load(), loadObjects(objPage, objPageSize)])}
        >
          刷新
        </Button>
      </Space>
      <Typography.Title level={5} style={{ margin: 0 }}>
        Consul 端点
      </Typography.Title>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={eps}
        pagination={tablePagination()}
        columns={[
          { title: "名称", dataIndex: "name", width: 140 },
          { title: "地址", dataIndex: "address", ellipsis: true },
          { title: "Tag", dataIndex: "service_tag", width: 120 },
          {
            title: "状态",
            width: 80,
            render: (_, r) => (r.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>),
          },
          {
            title: "上次同步",
            width: 170,
            render: (_, r) => formatDateTime(r.last_sync_at) || "-",
          },
          {
            title: "操作",
            width: 280,
            render: (_, r) => (
              <Space wrap size="small">
                <Button
                  size="small"
                  onClick={async () => {
                    try {
                      await pingConsulEndpoint(r.id);
                      message.success("Consul 连通正常");
                    } catch (e) {
                      message.error(extractApiErrorMessage(e, "连通失败"));
                    }
                  }}
                >
                  Ping
                </Button>
                <Button
                  size="small"
                  icon={<SyncOutlined />}
                  onClick={async () => {
                    try {
                      const res = await syncConsulEndpoint(r.id);
                      message.success(`同步完成：写入 ${res.upserted}，移除 ${res.removed}`);
                      await Promise.all([load(), loadObjects(1, objPageSize)]);
                    } catch (e) {
                      message.error(extractApiErrorMessage(e, "同步失败"));
                    }
                  }}
                >
                  同步
                </Button>
                <Button size="small" onClick={() => openEdit(r)}>
                  编辑
                </Button>
                <Popconfirm
                  title="删除端点及已同步对象？"
                  onConfirm={async () => {
                    try {
                      await deleteConsulEndpoint(r.id);
                      message.success("已删除");
                      await Promise.all([load(), loadObjects(1, objPageSize)]);
                    } catch (e) {
                      message.error(extractApiErrorMessage(e, "删除失败"));
                    }
                  }}
                >
                  <Button size="small" danger>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
      <Typography.Title level={5} style={{ margin: 0 }}>
        已同步对象
      </Typography.Title>
      <Table
        rowKey="id"
        loading={objLoading}
        dataSource={objs}
        pagination={tablePagination({
          current: objPage,
          pageSize: objPageSize,
          total: objTotal,
          onChange: (page, pageSize) => void loadObjects(page, pageSize),
        })}
        scroll={{ x: 1240 }}
        columns={[
          { title: "服务", dataIndex: "service_name", width: 120 },
          { title: "实例 ID", dataIndex: "service_id", width: 180, ellipsis: true },
          { title: "地址", dataIndex: "address", width: 140 },
          { title: "端口", dataIndex: "port", width: 70 },
          { title: "角色", dataIndex: "exporter_role", width: 110 },
          { title: "项目 meta", dataIndex: "yunshu_project", width: 120 },
          {
            title: "健康",
            dataIndex: "health",
            width: 90,
            render: (v: string) => {
              const c = v === "passing" ? "green" : v === "warning" ? "orange" : "red";
              return <Tag color={c}>{v || "-"}</Tag>;
            },
          },
          { title: "同步时间", dataIndex: "synced_at", width: 170, render: (v) => formatDateTime(v) || "-" },
          {
            title: "操作",
            width: 140,
            fixed: "right",
            render: (_: unknown, r: AlertMonitorObjectItem) => (
              <Button
                type="link"
                size="small"
                onClick={() => {
                  const fn = ctx.openRuleCreateFromObject as
                    | ((obj: AlertMonitorObjectItem) => void)
                    | undefined;
                  if (typeof fn === "function") {
                    fn(r);
                  } else {
                    message.warning("无法打开规则创建");
                  }
                }}
              >
                生成规则
              </Button>
            ),
          },
        ]}
      />
      <Modal
        title={editing ? "编辑 Consul 端点" : "新建 Consul 端点"}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => void submit()}
        destroyOnClose
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="project_id" label="项目名称" rules={[{ required: true, message: "请选择项目" }]}>
            <Select
              options={ctx.projectOptions}
              placeholder="请选择项目"
              showSearch
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="address" label="Consul 地址" rules={[{ required: true }]} extra="例如 http://consul:8500">
            <Input />
          </Form.Item>
          <Form.Item name="token" label="ACL Token" extra="留空表示不修改；勾选清空可删除 Token">
            <Input.Password autoComplete="new-password" />
          </Form.Item>
          <Form.Item name="clear_token" label="清空 Token" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="datacenter" label="Datacenter">
            <Input placeholder="可选" />
          </Form.Item>
          <Form.Item name="service_tag" label="服务 Tag 过滤">
            <Input placeholder="yunshu-metrics" />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </Space>
  );
}
