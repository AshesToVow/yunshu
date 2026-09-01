import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, message } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  GrantValidityCalendarPicker,
  expiresAtToGrantPeriod,
  grantPeriodToExpiresAt,
  type GrantValidityPeriod,
} from "../components/dbmgmt/grant-validity-calendar";
import { formatInstanceLabel } from "../components/dbmgmt/dbmgmt-ui-shared";
import { deleteDbGrant, listDbGrants, listDbInstances, updateDbGrant, type DbGrant, type DbInstance } from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { getUsers } from "../services/users";
import type { UserItem } from "../types/api";

function isQueryGrant(g: DbGrant) {
  const privs = g.privileges ?? [];
  if (privs.length) return privs.every((p) => p.toLowerCase() === "select");
  return g.can_query && !g.can_dml && !g.can_ddl && !g.can_import && !g.can_export;
}

function formatExpiry(v?: string) {
  if (!v) return "永久";
  const d = v.slice(0, 10);
  if (!d || d >= "9999") return "永久";
  return d;
}

function privSummary(g: DbGrant) {
  const parts: string[] = [];
  if (g.can_query) parts.push("查询");
  if (g.can_dml) parts.push("DML");
  if (g.can_ddl) parts.push("DDL");
  if (g.can_import) parts.push("导入");
  if (g.can_export) parts.push("导出");
  if (g.can_manage) parts.push("管理");
  if (g.privileges?.length) return g.privileges.join(", ");
  return parts.length ? parts.join(" / ") : "—";
}

export function DbmgmtGrantsPage({ preset = "all" }: { preset?: "all" | "query" }) {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [rows, setRows] = useState<DbGrant[]>([]);
  const [instances, setInstances] = useState<DbInstance[]>([]);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [keyword, setKeyword] = useState("");
  const [searchText, setSearchText] = useState("");
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<DbGrant>();
  const [editForm] = Form.useForm();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) setProjectId(res.list[0].id);
    });
    void getUsers({ page: 1, page_size: 500 }).then((res) => setUsers(res.list ?? []));
  }, []);

  const load = useCallback(async () => {
    if (!projectId) return;
    try {
      const [grants, inst] = await Promise.all([
        listDbGrants(projectId),
        listDbInstances(projectId, { page: 1, page_size: 200 }),
      ]);
      setRows((grants ?? []).filter((g) => (preset === "query" ? isQueryGrant(g) : true)));
      setInstances(inst.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载授权列表失败");
    }
  }, [projectId, preset]);

  useEffect(() => {
    void load();
  }, [load]);

  const instanceMap = useMemo(() => new Map(instances.map((i) => [i.id, i])), [instances]);
  const userNameMap = useMemo(() => {
    const m = new Map<string, string>();
    for (const u of users) {
      m.set(String(u.id), u.nickname || u.username);
      m.set(u.username, u.nickname || u.username);
    }
    return m;
  }, [users]);

  const displayName = (g: DbGrant) => {
    if (g.principal_kind === "user") return userNameMap.get(g.principal_ref) ?? g.principal_ref;
    return g.principal_ref;
  };

  const filteredRows = useMemo(() => {
    const kw = searchText.trim().toLowerCase();
    if (!kw) return rows;
    return rows.filter((g) => {
      const inst = instanceMap.get(g.instance_id);
      const label = inst ? formatInstanceLabel(inst).toLowerCase() : "";
      return (
        displayName(g).toLowerCase().includes(kw) ||
        label.includes(kw) ||
        (g.database_name ?? "").toLowerCase().includes(kw)
      );
    });
  }, [rows, searchText, instanceMap, userNameMap]);

  const instanceIp = (id: number) => {
    const inst = instanceMap.get(id);
    return inst ? formatInstanceLabel(inst) : String(id);
  };

  const openEdit = (row: DbGrant) => {
    setEditing(row);
    editForm.setFieldsValue({
      query_limit_num: row.query_limit_num ?? 1000,
      grant_period: expiresAtToGrantPeriod(row.expires_at),
      remark: row.remark,
    });
    setEditOpen(true);
  };

  const submitEdit = async () => {
    if (!projectId || !editing) return;
    const values = await editForm.validateFields();
    const period = values.grant_period as GrantValidityPeriod | null | undefined;
    await updateDbGrant(projectId, editing.id, {
      query_limit_num: values.query_limit_num,
      expires_at: grantPeriodToExpiresAt(period ?? null),
      remark: values.remark,
    });
    message.success("已更新查询权限");
    setEditOpen(false);
    void load();
  };

  if (preset === "query") {
    return (
      <Card title="SQL查询权限管理">
        <Space style={{ marginBottom: 16 }} wrap>
          <Link to="/dbmgmt/apply/query">
            <Button type="primary" icon={<PlusOutlined />}>
              SQL查询权限申请
            </Button>
          </Link>
          <Input
            style={{ width: 220 }}
            placeholder="请输入名称进行搜索"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onPressEnter={() => setSearchText(keyword)}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={() => setSearchText(keyword)}>
            搜索
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => { setKeyword(""); setSearchText(""); void load(); }}>
            刷新重置
          </Button>
          <Select style={{ width: 180 }} value={projectId} options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={setProjectId} />
        </Space>
        <Table
          rowKey="id"
          dataSource={filteredRows}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "中文名称", render: (_: unknown, r: DbGrant) => displayName(r) },
            { title: "实例IP", dataIndex: "instance_id", render: (v: number) => instanceIp(v) },
            {
              title: "权限级别",
              render: (_: unknown, r: DbGrant) => (r.table_names?.length ? "TABLE" : "DATABASE"),
            },
            { title: "数据库", dataIndex: "database_name", render: (v?: string) => v || "—" },
            { title: "表", render: (_: unknown, r: DbGrant) => (r.table_names?.length ? r.table_names.join(", ") : "—") },
            { title: "结果集", dataIndex: "query_limit_num", width: 90, render: (v?: number) => v ?? 1000 },
            { title: "有效时间", dataIndex: "expires_at", width: 120, render: (v?: string) => formatExpiry(v) },
            {
              title: "操作",
              width: 140,
              render: (_, r) => (
                <Space>
                  <Button size="small" type="primary" icon={<EditOutlined />} onClick={() => openEdit(r)}>
                    编辑
                  </Button>
                  <Popconfirm title="删除该查询权限？" onConfirm={() => void deleteDbGrant(projectId!, r.id).then(load)}>
                    <Button size="small" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
        <Modal title="编辑查询权限" open={editOpen} onCancel={() => setEditOpen(false)} onOk={() => void submitEdit()} destroyOnClose width={760}>
          {editing ? (
            <Form form={editForm} layout="vertical">
              <Form.Item label="用户">
                <Input value={displayName(editing)} disabled />
              </Form.Item>
              <Form.Item label="实例">
                <Input value={instanceIp(editing.instance_id)} disabled />
              </Form.Item>
              <Form.Item label="数据库">
                <Input value={editing.database_name || "—"} disabled />
              </Form.Item>
              <Form.Item name="query_limit_num" label="结果集行数上限" rules={[{ required: true }]}>
                <InputNumber min={1} max={100000} style={{ width: "100%" }} />
              </Form.Item>
              <Form.Item
                name="grant_period"
                label="授权有效期"
                extra="在日历上调整起止日期；点选「永久有效」表示不过期"
              >
                <GrantValidityCalendarPicker />
              </Form.Item>
              <Form.Item name="remark" label="备注">
                <Input.TextArea rows={2} />
              </Form.Item>
            </Form>
          ) : null}
        </Modal>
      </Card>
    );
  }

  return (
    <Card
      title="授权管理"
      extra={
        <Space>
          <Select style={{ width: 200 }} value={projectId} options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={setProjectId} />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} />
        </Space>
      }
    >
      <Table
        rowKey="id"
        dataSource={rows}
        pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        columns={[
          { title: "ID", dataIndex: "id", width: 70 },
          { title: "主体", render: (_: unknown, r: DbGrant) => displayName(r) },
          { title: "类型", dataIndex: "principal_kind", width: 80 },
          { title: "实例", dataIndex: "instance_id", render: (v: number) => instanceIp(v) },
          { title: "数据库", dataIndex: "database_name", render: (v?: string) => v || "—" },
          { title: "权限", render: (_: unknown, r: DbGrant) => privSummary(r) },
          { title: "有效期", dataIndex: "expires_at", width: 120, render: (v?: string) => formatExpiry(v) },
          { title: "备注", dataIndex: "remark", ellipsis: true, render: (v?: string) => v || "—" },
          {
            title: "操作",
            width: 80,
            render: (_, r) => (
              <Popconfirm title="删除该授权？" onConfirm={() => void deleteDbGrant(projectId!, r.id).then(load)}>
                <Button size="small" danger icon={<DeleteOutlined />} />
              </Popconfirm>
            ),
          },
        ]}
      />
    </Card>
  );
}

export function DbmgmtQueryGrantsPage() {
  return <DbmgmtGrantsPage preset="query" />;
}
