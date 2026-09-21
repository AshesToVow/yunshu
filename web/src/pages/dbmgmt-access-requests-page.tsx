import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Form, Input, Modal, Select, Space, Table, Tag, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  GrantValidityCalendarPicker,
  grantPeriodToExpiresAt,
  type GrantValidityPeriod,
} from "../components/dbmgmt/grant-validity-calendar";
import { DB_PRIVILEGE_OPTIONS, privilegeSummary } from "../components/dbmgmt/dbmgmt-ui-shared";
import {
  createDbAccessRequest,
  listDbAccessRequests,
  listDbDatabases,
  listDbInstances,
  listDbTables,
  type DbAccessRequest,
  type DbInstance,
} from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";
import { accessRequestStatusLabel } from "../utils/dbmgmt-labels";

export type DbmgmtAccessRequestPreset = "all" | "query" | "database_create";

const QUERY_LIMIT_OPTIONS = [
  { value: 100, label: "100 行" },
  { value: 500, label: "500 行" },
  { value: 1000, label: "1000 行（默认）" },
  { value: 5000, label: "5000 行" },
  { value: 10000, label: "10000 行" },
];

const PRESET_META: Record<
  DbmgmtAccessRequestPreset,
  { title: string; hint?: string; defaultScope: string; defaultPrivileges: string[]; showList?: boolean }
> = {
  all: { title: "权限申请", defaultScope: "database", defaultPrivileges: ["select"], showList: true },
  query: {
    title: "平台查询权限申请",
    hint: "申请后在 SQL 查询页面对应库/表执行 SELECT；库级权限覆盖该库全部表。",
    defaultScope: "database",
    defaultPrivileges: ["select"],
    showList: false,
  },
  database_create: {
    title: "数据库创建申请",
    hint: "审批通过后将自动 CREATE DATABASE 并写入平台授权（对齐 smartdbs 建库流程）。",
    defaultScope: "new_database",
    defaultPrivileges: ["create_database"],
    showList: false,
  },
};

function defaultGrantPeriod(days = 7): GrantValidityPeriod {
  const start = dayjs().startOf("day");
  return { start, end: start.add(days - 1, "day").endOf("day") };
}

function matchesPreset(row: DbAccessRequest, preset: DbmgmtAccessRequestPreset) {
  if (preset === "all") return true;
  if (preset === "database_create") {
    return row.scope_type === "new_database" || row.privileges?.includes("create_database");
  }
  const privs = row.privileges ?? [];
  const queryOnly = privs.length > 0 && privs.every((p) => p === "select");
  return queryOnly && row.scope_type !== "new_database";
}

export function DbmgmtAccessRequestsPage({ preset = "all" }: { preset?: DbmgmtAccessRequestPreset }) {
  const meta = PRESET_META[preset];
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [rows, setRows] = useState<DbAccessRequest[]>([]);
  const [instances, setInstances] = useState<DbInstance[]>([]);
  const [databases, setDatabases] = useState<string[]>([]);
  const [tables, setTables] = useState<string[]>([]);
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();
  const instanceId = Form.useWatch("instance_id", form);
  const databaseName = Form.useWatch("database_name", form);
  const scopeType = Form.useWatch("scope_type", form) ?? "database";

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) setProjectId(res.list[0].id);
    });
  }, []);

  const load = useCallback(async () => {
    if (!projectId) return;
    try {
      const [reqs, inst] = await Promise.all([
        listDbAccessRequests(projectId, { page: 1, page_size: 100 }),
        listDbInstances(projectId, { page: 1, page_size: 200 }),
      ]);
      setRows((reqs.list ?? []).filter((r) => matchesPreset(r, preset)));
      setInstances(inst.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载申请列表失败");
    }
  }, [projectId, preset]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!projectId || !instanceId) {
      setDatabases([]);
      return;
    }
    void listDbDatabases(projectId, instanceId)
      .then((res) => setDatabases((res ?? []).map((d) => d.name)))
      .catch(() => setDatabases([]));
  }, [projectId, instanceId]);

  useEffect(() => {
    if (!projectId || !instanceId || !databaseName || scopeType !== "table") {
      setTables([]);
      return;
    }
    void listDbTables(projectId, instanceId, databaseName)
      .then((res) => setTables((res ?? []).map((t) => t.name)))
      .catch(() => setTables([]));
  }, [projectId, instanceId, databaseName, scopeType]);

  const columns: ColumnsType<DbAccessRequest> = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "实例", dataIndex: "instance_name" },
    { title: "库", dataIndex: "database_name" },
    {
      title: "范围",
      render: (_, r) => {
        if (r.scope_type === "new_database" || r.privileges?.includes("create_database")) {
          return `新建库：${r.database_name}`;
        }
        if (r.table_names?.length) return `表：${r.table_names.join(", ")}`;
        return "整库";
      },
    },
    {
      title: "权限",
      dataIndex: "privileges",
      ellipsis: true,
      render: (v?: string[]) => privilegeSummary(v),
    },
    { title: "申请人", dataIndex: "requester_name" },
    { title: "理由", dataIndex: "reason", ellipsis: true },
    { title: "状态", dataIndex: "status", render: (v: string) => <Tag>{accessRequestStatusLabel(v)}</Tag> },
    {
      title: "过期时间",
      dataIndex: "expires_at",
      render: (v?: string) => {
        if (!v) return "永久";
        const expired = dayjs(v).isBefore(dayjs());
        return <span style={expired ? { color: "#cf1322" } : undefined}>{formatDateTime(v)}</span>;
      },
    },
    { title: "申请时间", dataIndex: "created_at", render: (v) => formatDateTime(v) },
  ];

  const submit = async () => {
    if (!projectId || submitting) return;
    setSubmitting(true);
    try {
      const values = await form.validateFields();
      const period = values.grant_period as GrantValidityPeriod | null | undefined;
      await createDbAccessRequest(projectId, {
        instance_id: values.instance_id,
        scope_type: values.scope_type,
        database_name: values.database_name,
        table_names: values.scope_type === "table" ? values.table_names : undefined,
        privileges: values.privileges,
        query_limit_num: values.query_limit_num,
        reason: values.reason,
        expires_at: grantPeriodToExpiresAt(period ?? null),
      });
      message.success("已提交申请");
      setOpen(false);
      form.resetFields();
      void load();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "提交失败");
    } finally {
      setSubmitting(false);
    }
  };

  const scopeOptions = useMemo(() => {
    if (preset === "database_create") {
      return [{ value: "new_database", label: "新建库" }];
    }
    if (preset === "query") {
      return [
        { value: "database", label: "DATABASE（整库查询）" },
        { value: "table", label: "TABLE（单表查询）" },
      ];
    }
    return [
      { value: "database", label: "库级（已有库，全部表）" },
      { value: "table", label: "表级（已有库，指定表）" },
      { value: "new_database", label: "新建库（实例级 CREATE DATABASE）" },
    ];
  }, [preset]);

  const privilegeOptions = useMemo((): { value: string; label: string }[] => {
    if (preset === "query") return DB_PRIVILEGE_OPTIONS.filter((o) => o.value === "select");
    if (preset === "database_create") return DB_PRIVILEGE_OPTIONS.filter((o) => o.value === "create_database");
    return [...DB_PRIVILEGE_OPTIONS];
  }, [preset]);

  const openCreate = () => {
    form.setFieldsValue({
      scope_type: meta.defaultScope,
      privileges: meta.defaultPrivileges,
      grant_period: preset === "query" ? defaultGrantPeriod(7) : defaultGrantPeriod(30),
      query_limit_num: preset === "query" ? 1000 : undefined,
    });
    setOpen(true);
  };

  const grantPeriodRules = [
    {
      validator: (_: unknown, value: GrantValidityPeriod | null | undefined) => {
        if (value === null) return Promise.resolve();
        if (value?.start && value?.end) {
          if (!value.end.endOf("day").isBefore(dayjs().startOf("day"))) return Promise.resolve();
          return Promise.reject(new Error("结束日期不能早于今天"));
        }
        return Promise.reject(new Error("请在日历上选择授权起止日期，或点选「永久有效」"));
      },
    },
  ];

  return (
    <Card
      title={meta.title}
      extra={
        <Space>
          <Select style={{ width: 200 }} value={projectId} options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={setProjectId} />
          {meta.showList !== false ? <Button icon={<ReloadOutlined />} onClick={() => void load()} /> : null}
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            提交申请
          </Button>
        </Space>
      }
    >
      {meta.hint ? <Alert type="info" showIcon message={meta.hint} style={{ marginBottom: 16 }} /> : null}
      {preset === "query" ? (
        <Alert
          type="warning"
          showIcon
          message={
            <>
              已有查询权限可在 <Link to="/dbmgmt/apply/query-grants">查询权限管理</Link> 查看；审批进度请到{" "}
              <Link to="/workflow/inbox?domain=dbmgmt">我的待办</Link>。
            </>
          }
          style={{ marginBottom: 16 }}
        />
      ) : null}
      {meta.showList !== false ? <Table rowKey="id" columns={columns} dataSource={rows} /> : null}
      <Modal
        title={meta.title}
        open={open}
        confirmLoading={submitting}
        onCancel={() => {
          if (!submitting) setOpen(false);
        }}
        onOk={() => void submit()}
        width={preset === "query" ? 760 : 680}
      >
        <Form form={form} layout="vertical" initialValues={{ scope_type: meta.defaultScope, privileges: meta.defaultPrivileges }}>
          <Form.Item name="instance_id" label="实例" rules={[{ required: true }]}>
            <Select
              options={instances.map((i) => ({ value: i.id, label: i.name }))}
              onChange={() => {
                form.setFieldsValue({ database_name: undefined, table_names: undefined });
              }}
            />
          </Form.Item>
          <Form.Item name="scope_type" label={preset === "query" ? "权限级别" : "申请范围"}>
            <Select
              options={scopeOptions}
              disabled={preset === "database_create"}
              onChange={(v) => {
                form.setFieldsValue({
                  database_name: undefined,
                  table_names: undefined,
                  privileges: v === "new_database" ? ["create_database"] : preset === "query" ? ["select"] : ["select"],
                });
              }}
            />
          </Form.Item>
          {scopeType === "new_database" ? (
            <Form.Item
              name="database_name"
              label="新建库名"
              rules={[{ required: true, message: "请填写要创建的数据库名" }]}
              extra="库尚未存在，填写计划创建的库名，如 test"
            >
              <Input placeholder="例如 test" />
            </Form.Item>
          ) : (
            <Form.Item name="database_name" label="目标库" rules={[{ required: true, message: "请选择数据库" }]}>
              <Select
                showSearch
                placeholder="从实例元数据加载"
                options={databases.map((d) => ({ value: d, label: d }))}
                onChange={() => form.setFieldValue("table_names", undefined)}
              />
            </Form.Item>
          )}
          {scopeType === "table" ? (
            <Form.Item name="table_names" label="目标表" rules={[{ required: true, message: "请选择至少一个表" }]}>
              <Select mode="multiple" showSearch placeholder="选择表" options={tables.map((t) => ({ value: t, label: t }))} />
            </Form.Item>
          ) : null}
          <Form.Item name="privileges" label="权限项" rules={[{ required: true, message: "请至少选择一项权限" }]} hidden={preset !== "all"}>
            <Select mode="multiple" placeholder="选择需要的 SQL 权限" options={privilegeOptions} />
          </Form.Item>
          {preset === "query" ? (
            <>
              <Form.Item name="query_limit_num" label="查询行数上限" rules={[{ required: true, message: "请选择行数上限" }]}>
                <Select options={QUERY_LIMIT_OPTIONS} placeholder="单次查询最大返回行数" />
              </Form.Item>
              <Form.Item
                name="grant_period"
                label="授权有效期"
                rules={grantPeriodRules}
                extra="在日历上点选开始与结束日期；快捷标签可一键填充。审批通过后仅在所选日期内可查询。"
              >
                <GrantValidityCalendarPicker />
              </Form.Item>
            </>
          ) : (
            <Form.Item
              name="grant_period"
              label="权限有效期"
              rules={grantPeriodRules}
              extra="留空请点选「永久有效」；到期后授权自动失效"
            >
              <GrantValidityCalendarPicker />
            </Form.Item>
          )}
          <Form.Item name="reason" label={preset === "database_create" ? "备注" : "申请理由"} rules={[{ required: true }]}>
            <Input.TextArea rows={3} placeholder={preset === "database_create" ? "例如：用户中心业务库" : undefined} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}

export function DbmgmtQueryApplyPage() {
  return <DbmgmtAccessRequestsPage preset="query" />;
}