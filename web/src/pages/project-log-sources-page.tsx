import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, DeploymentUnitOutlined } from "@ant-design/icons";
import {
  AutoComplete,
  Button,
  Card,
  Col,
  Form,
  Input,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tag,
  message,
} from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  getProjects,
  getProjectServers,
  getProjectServices,
  getProjectLogSources,
  upsertProjectLogSource,
  deleteProjectLogSource,
  type ProjectItem,
  type ServerItem,
  type ServiceItem,
  type LogSourceItem,
} from "../services/projects";
import { extractApiErrorMessage } from "../services/http";
import { useDictOptions } from "../hooks/use-dict-options";

function isFormValidationError(error: unknown): boolean {
  return typeof error === "object" && error !== null && "errorFields" in error;
}

export function ProjectLogSourcesPage({ embedded = false }: { embedded?: boolean } = {}) {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [sources, setSources] = useState<LogSourceItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [serverId, setServerId] = useState<number>();
  const [serviceId, setServiceId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [deletingId, setDeletingId] = useState<number>();
  const [fileOptions, setFileOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [current, setCurrent] = useState<LogSourceItem | null>(null);
  const loadSeqRef = useRef(0);
  const serversSeqRef = useRef(0);
  const servicesSeqRef = useRef(0);
  const [form] = Form.useForm<{
    id?: number;
    service_id: number;
    log_type: string;
    path: string;
    log_dir?: string;
    include_regex?: string;
    exclude_regex?: string;
    multiline_rule?: string;
    status: number;
  }>();
  const logTypeOptions = useDictOptions("log_source_type");
  const logSourceStatusOptions = useDictOptions("common_status");

  const projectOptions = useMemo(
    () => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })),
    [projects],
  );
  const serverOptions = useMemo(
    () =>
      servers.map((s) => ({ value: s.id, label: `${s.name} ${s.host}:${s.port} (${s.os_type}/${s.os_arch || "-"})` })),
    [servers],
  );
  const serviceOptions = useMemo(() => services.map((s) => ({ value: s.id, label: s.name })), [services]);

  useEffect(() => {
    void (async () => {
      try {
        const p = await getProjects({ page: 1, page_size: 1000 });
        setProjects(p.list);
        if (p.list[0]) setProjectId(p.list[0].id);
      } catch (e: unknown) {
        message.error(extractApiErrorMessage(e, "加载项目列表失败"));
      }
    })();
  }, []);

  useEffect(() => {
    const seq = ++serversSeqRef.current;
    if (!projectId) {
      setServers([]);
      setServerId(undefined);
      setServiceId(undefined);
      setServices([]);
      return;
    }
    void (async () => {
      try {
        const sv = await getProjectServers(projectId, { page: 1, page_size: 1000 });
        if (seq !== serversSeqRef.current) return;
        setServers(sv.list);
        setServerId(undefined);
        setServiceId(undefined);
        setServices([]);
      } catch (e: unknown) {
        if (seq !== serversSeqRef.current) return;
        message.error(extractApiErrorMessage(e, "加载服务器列表失败"));
      }
    })();
  }, [projectId]);

  useEffect(() => {
    const seq = ++servicesSeqRef.current;
    if (!projectId) {
      setServices([]);
      setServiceId(undefined);
      return;
    }
    void (async () => {
      try {
        const list = await getProjectServices(projectId, { page: 1, page_size: 1000, server_id: serverId });
        if (seq !== servicesSeqRef.current) return;
        setServices(list.list);
        setServiceId(undefined);
      } catch (e: unknown) {
        if (seq !== servicesSeqRef.current) return;
        message.error(extractApiErrorMessage(e, "加载服务列表失败"));
      }
    })();
  }, [projectId, serverId]);

  useEffect(() => {
    if (!projectId) {
      setSources([]);
      return;
    }
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, serviceId]);

  function splitFilePathForEditor(rawPath: string): { logDir?: string; filePart: string } {
    const p = String(rawPath || "").trim();
    if (!p) return { filePart: "" };
    const normalized = p.replace(/\\/g, "/");
    const idx = normalized.lastIndexOf("/");
    if (idx <= 0) return { filePart: p };
    return {
      logDir: normalized.slice(0, idx),
      filePart: normalized.slice(idx + 1),
    };
  }

  async function load() {
    const seq = ++loadSeqRef.current;
    if (!projectId) {
      setSources([]);
      setLoading(false);
      return;
    }
    setLoading(true);
    try {
      const res = await getProjectLogSources(projectId, { page: 1, page_size: 1000, service_id: serviceId });
      if (seq !== loadSeqRef.current) return;
      setSources(res.list);
    } catch (e: unknown) {
      if (seq !== loadSeqRef.current) return;
      message.error(extractApiErrorMessage(e, "加载日志源失败"));
    } finally {
      if (seq === loadSeqRef.current) setLoading(false);
    }
  }

  function handleProjectChange(id: number) {
    loadSeqRef.current += 1;
    serversSeqRef.current += 1;
    servicesSeqRef.current += 1;
    setProjectId(id);
    setServers([]);
    setServices([]);
    setSources([]);
    setServerId(undefined);
    setServiceId(undefined);
    setLoading(false);
  }

  function openCreate() {
    if (!serviceId) {
      message.warning("请先选择服务");
      return;
    }
    setCurrent(null);
    setFileOptions([]);
    form.resetFields();
    form.setFieldsValue({ service_id: serviceId, log_type: "file", status: 1 });
    setEditorOpen(true);
  }

  function openEdit(record: LogSourceItem) {
    setCurrent(record);
    setFileOptions(record.path ? [{ value: record.path, label: record.path }] : []);
    const split = record.log_type === "file" ? splitFilePathForEditor(record.path) : { filePart: record.path };
    form.setFieldsValue({
      id: record.id,
      service_id: record.service_id,
      log_type: record.log_type,
      path: split.filePart || record.path,
      log_dir: split.logDir,
      status: record.status,
      include_regex: record.include_regex ?? undefined,
      exclude_regex: record.exclude_regex ?? undefined,
      multiline_rule: record.multiline_rule ?? undefined,
    });
    setEditorOpen(true);
  }

  async function onSubmit() {
    if (!projectId || submitting) return;
    setSubmitting(true);
    try {
      const v = await form.validateFields();
      let finalPath = String(v.path || "").trim();
      if (v.log_type === "file") {
        const logDir = String(v.log_dir || "").trim();
        const isAbsWin = /^[a-zA-Z]:[\\/]/.test(finalPath);
        const isAbsUnix = finalPath.startsWith("/");
        if (logDir && !isAbsWin && !isAbsUnix) {
          const d = logDir.replace(/[\\/]+$/, "");
          const p = finalPath.replace(/^[\\/]+/, "");
          finalPath = `${d}/${p}`;
        }
      }
      await upsertProjectLogSource(projectId, {
        id: v.id,
        service_id: v.service_id,
        log_type: v.log_type,
        path: finalPath,
        include_regex: v.include_regex,
        exclude_regex: v.exclude_regex,
        multiline_rule: v.multiline_rule,
        status: v.status,
      });
      message.success(current ? "已更新日志源" : "已创建日志源");
      setEditorOpen(false);
      await load();
    } catch (e: unknown) {
      if (isFormValidationError(e)) return;
      message.error(extractApiErrorMessage(e, current ? "更新日志源失败" : "创建日志源失败"));
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(record: LogSourceItem) {
    if (!projectId || deletingId != null) return;
    setDeletingId(record.id);
    try {
      await deleteProjectLogSource(projectId, record.id);
      message.success("已删除");
      await load();
    } catch (e: unknown) {
      message.error(extractApiErrorMessage(e, "删除日志源失败"));
    } finally {
      setDeletingId(undefined);
    }
  }

  const toolbar = (
    <Space wrap>
      <Select
        style={{ width: 240 }}
        value={projectId}
        onChange={handleProjectChange}
        options={projectOptions}
        placeholder="项目"
      />
      <Select
        style={{ width: 300 }}
        value={serverId}
        onChange={setServerId}
        options={serverOptions}
        placeholder="服务器"
        allowClear
      />
      <Select
        style={{ width: 220 }}
        value={serviceId}
        onChange={setServiceId}
        options={serviceOptions}
        placeholder="服务"
        allowClear
      />
      <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
        刷新
      </Button>
      <Button icon={<DeploymentUnitOutlined />} onClick={() => navigate("/loggie-status")} disabled={!serverId}>
        Agent 管理
      </Button>
      <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
        新增日志源
      </Button>
    </Space>
  );

  const body = (
    <>
      <div className="toolbar">{toolbar}</div>
      <Table
        rowKey="id"
        dataSource={sources}
        loading={loading}
        pagination={false}
        scroll={{ x: 1000 }}
        columns={[
          { title: "类型", dataIndex: "log_type", width: 100 },
          { title: "路径/Unit", dataIndex: "path" },
          { title: "include", dataIndex: "include_regex", render: (v?: string | null) => v || "-" },
          { title: "exclude", dataIndex: "exclude_regex", render: (v?: string | null) => v || "-" },
          {
            title: "状态",
            dataIndex: "status",
            width: 90,
            render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>),
          },
          {
            title: "操作",
            width: 200,
            fixed: "right",
            render: (_: unknown, record: LogSourceItem) => (
              <Space>
                <Button icon={<EditOutlined />} disabled={deletingId != null} onClick={() => openEdit(record)}>
                  编辑
                </Button>
                <Popconfirm
                  title="确认删除日志源？"
                  okText="删除"
                  cancelText="取消"
                  onConfirm={() => handleDelete(record)}
                >
                  <Button danger icon={<DeleteOutlined />} loading={deletingId === record.id} disabled={deletingId != null}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        open={editorOpen}
        title={current ? "编辑日志源" : "新增日志源"}
        onCancel={() => setEditorOpen(false)}
        onOk={() => void onSubmit()}
        confirmLoading={submitting}
        width={920}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="id" hidden>
            <Input />
          </Form.Item>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item label="服务" name="service_id" rules={[{ required: true }]}>
                <Select options={serviceOptions} />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item label="类型" name="log_type" rules={[{ required: true }]}>
                <Select options={logTypeOptions} />
              </Form.Item>
            </Col>
            <Col span={4}>
              <Form.Item label="状态" name="status" rules={[{ required: true }]}>
                <Select options={logSourceStatusOptions} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item label="日志目录（file 类型）" name="log_dir">
                <Input placeholder="/var/log/app" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={24}>
              <span style={{ color: "#999" }}>配置完成后到「Agent 管理」引导并安装采集端</span>
            </Col>
          </Row>
          <Row gutter={12} style={{ marginTop: 12 }}>
            <Col span={24}>
              <Form.Item label="路径/Unit" name="path" rules={[{ required: true }]}>
                <AutoComplete
                  allowClear
                  options={fileOptions}
                  placeholder="可从扫描结果选择，也可手动输入"
                  filterOption={(input, option) =>
                    String(option?.label ?? "")
                      .toLowerCase()
                      .includes(input.toLowerCase())
                  }
                />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item label="include regex" name="include_regex">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="exclude regex" name="exclude_regex">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item
                label="解析模板（multiline_rule）"
                name="multiline_rule"
                tooltip="留空按路径自动识别。Elasticsearch ERROR 堆栈需选 elasticsearch 并热更 Agent。"
              >
                <Select
                  allowClear
                  placeholder="自动识别"
                  options={[
                    { value: "elasticsearch", label: "Elasticsearch / Java [ERROR ] 方括号堆栈" },
                    { value: "spring", label: "Spring Boot 2024-01-01 INFO（Logback）" },
                    { value: "cityeyes-vap", label: "CityEyesVap 管道分隔（JSON/堆栈多行）" },
                    { value: "kafka", label: "Kafka [时间] INFO" },
                    { value: "redis", label: "Redis 1:M 日期 * / #" },
                    { value: "mysql", label: "MySQL 5.7 error log" },
                    { value: "mysql_slow", label: "MySQL slow query（# Time:）" },
                    { value: "zookeeper", label: "Zookeeper [myid:1] - INFO" },
                    { value: "cri", label: "K8s CRI 容器日志（/var/log/pods）" },
                    { value: "syslog", label: "Syslog /var/log/messages" },
                    { value: "nginx_access", label: "Nginx access" },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </>
  );

  if (embedded) {
    return body;
  }
  return (
    <Card className="table-card" title="日志源配置">
      {body}
    </Card>
  );
}
