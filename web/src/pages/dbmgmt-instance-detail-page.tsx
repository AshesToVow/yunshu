import { DeleteOutlined, EditOutlined, EyeOutlined, PlusOutlined, ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Descriptions, Input, Modal, Select, Space, Table, Tabs, Tag, message } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { DbmgmtSectionTitle, formatInstanceLabel } from "../components/dbmgmt/dbmgmt-ui-shared";
import {
  getDbInstance,
  getInstanceAccountPassword,
  listDbDatabases,
  listInstanceMySQLUsers,
  type DbInstance,
  type DbInstanceMySQLUser,
} from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";
import { envLabel as dbEnvLabel, instanceRoleLabel } from "../utils/dbmgmt-labels";

function envLabel(env?: string) {
  const label = dbEnvLabel(env);
  return label === "—" ? "—" : `${label}环境`;
}

function statusTag(instance?: DbInstance) {
  if (!instance) return null;
  const ok = instance.last_ping_ok;
  const label = ok ? "运行中" : instance.status || "未知";
  return <Tag color={ok ? "green" : "default"}>{label}</Tag>;
}

export function DbmgmtInstanceDetailPage() {
  const { instanceId: instanceIdParam } = useParams<{ instanceId: string }>();
  const [searchParams, setSearchParams] = useSearchParams();
  const instanceId = Number(instanceIdParam);
  const projectFromUrl = Number(searchParams.get("project")) || undefined;
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instance, setInstance] = useState<DbInstance>();
  const [databases, setDatabases] = useState<{ name: string }[]>([]);
  const [users, setUsers] = useState<DbInstanceMySQLUser[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [dbKeyword, setDbKeyword] = useState("");
  const [userKeyword, setUserKeyword] = useState("");
  const tab = searchParams.get("tab") ?? "info";

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (projectFromUrl) {
        setProjectId(projectFromUrl);
      } else if (res.list?.length) {
        setProjectId(res.list[0].id);
      }
    });
  }, [projectFromUrl]);

  const load = useCallback(async () => {
    if (!projectId || !instanceId) return;
    setLoading(true);
    try {
      const inst = await getDbInstance(projectId, instanceId);
      setInstance(inst);
      const dbs = await listDbDatabases(projectId, instanceId);
      setDatabases(dbs ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载实例失败");
    } finally {
      setLoading(false);
    }
  }, [projectId, instanceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const loadUsers = useCallback(async () => {
    if (!projectId || !instanceId || instance?.driver !== "mysql") return;
    setUsersLoading(true);
    try {
      const list = await listInstanceMySQLUsers(projectId, instanceId);
      setUsers(list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载用户失败");
    } finally {
      setUsersLoading(false);
    }
  }, [projectId, instanceId, instance?.driver]);

  useEffect(() => {
    if (tab === "users" || tab === "databases") void loadUsers();
  }, [tab, loadUsers]);

  const revealPassword = async (row: DbInstanceMySQLUser) => {
    if (!projectId || !instanceId || !row.id) return;
    try {
      const res = await getInstanceAccountPassword(projectId, instanceId, row.id);
      Modal.info({ title: `${row.username}@${row.host} 密码`, content: res.password, width: 480 });
    } catch (e) {
      message.error(e instanceof Error ? e.message : "查看密码失败");
    }
  };

  const onProjectChange = (id: number) => {
    setProjectId(id);
    const next = new URLSearchParams(searchParams);
    next.set("project", String(id));
    if (tab) next.set("tab", tab);
    setSearchParams(next);
  };

  const projectName = projects.find((p) => p.id === (instance?.project_id ?? projectId))?.name ?? "—";
  const filteredDbs = useMemo(() => {
    const kw = dbKeyword.trim().toLowerCase();
    if (!kw) return databases;
    return databases.filter((d) => d.name.toLowerCase().includes(kw));
  }, [databases, dbKeyword]);

  const filteredUsers = useMemo(() => {
    const kw = userKeyword.trim().toLowerCase();
    if (!kw) return users;
    return users.filter((u) => `${u.username}@${u.host}`.toLowerCase().includes(kw));
  }, [users, userKeyword]);

  const grantAccountsForDb = (dbName: string) => {
    const lines: string[] = [];
    for (const u of users) {
      const grants = u.grant_lines ?? [];
      if (grants.some((g) => g.includes(`\`${dbName}\``) || g.includes(`${dbName}.`))) {
        lines.push(`'${u.username}'@'${u.host}'`);
      }
    }
    return lines.length ? lines.join(" ") : "—";
  };

  const infoTab = instance ? (
    <div>
      <DbmgmtSectionTitle>基本信息</DbmgmtSectionTitle>
      <Descriptions bordered column={3} size="small" style={{ marginBottom: 24 }}>
        <Descriptions.Item label="实例名称">{instance.name}</Descriptions.Item>
        <Descriptions.Item label="连接地址">{formatInstanceLabel(instance)}</Descriptions.Item>
        <Descriptions.Item label="驱动">{instance.driver}</Descriptions.Item>
        <Descriptions.Item label="连接模式">{instance.connect_mode || "—"}</Descriptions.Item>
        <Descriptions.Item label="库角色">
          <Tag color={(instance.role ?? "primary") === "replica" ? "blue" : "green"}>
            {instanceRoleLabel(instance.role)}
          </Tag>
        </Descriptions.Item>
        {instance.role === "replica" ? (
          <Descriptions.Item label="关联主库">
            {instance.primary_instance_name ? (
              <Link to={`/dbmgmt/instances/${instance.primary_instance_id}?project=${projectId ?? ""}`}>
                {instance.primary_instance_name}
              </Link>
            ) : (
              instance.primary_instance_id ? `#${instance.primary_instance_id}` : "—"
            )}
          </Descriptions.Item>
        ) : null}
        <Descriptions.Item label="只读">{instance.read_only ? "是" : "否"}</Descriptions.Item>
        <Descriptions.Item label="DML需工单">{instance.require_ticket_for_dml ? "是" : "否"}</Descriptions.Item>
        {instance.tags ? <Descriptions.Item label="标签">{instance.tags}</Descriptions.Item> : null}
        {instance.remark ? <Descriptions.Item label="备注" span={3}>{instance.remark}</Descriptions.Item> : null}
      </Descriptions>

      <DbmgmtSectionTitle>项目环境</DbmgmtSectionTitle>
      <Descriptions bordered column={3} size="small" style={{ marginBottom: 24 }}>
        <Descriptions.Item label="项目名称">{projectName}</Descriptions.Item>
        <Descriptions.Item label="环境">{envLabel(instance.env)}</Descriptions.Item>
        <Descriptions.Item label="运行状态">{statusTag(instance)}</Descriptions.Item>
      </Descriptions>

      <DbmgmtSectionTitle>连接信息</DbmgmtSectionTitle>
      <Descriptions bordered column={3} size="small" style={{ marginBottom: 24 }}>
        <Descriptions.Item label="主机">{instance.host}</Descriptions.Item>
        <Descriptions.Item label="端口">{instance.port}</Descriptions.Item>
        <Descriptions.Item label="默认库">{instance.database || "—"}</Descriptions.Item>
        <Descriptions.Item label="管理用户">{instance.username}</Descriptions.Item>
        <Descriptions.Item label="SSL">{instance.ssl_mode || "—"}</Descriptions.Item>
        <Descriptions.Item label="写入能力">
          <Tag color={instance.read_only ? "default" : "green"}>{instance.read_only ? "只读" : "可写"}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="创建时间">{instance.created_at ? formatDateTime(instance.created_at) : "—"}</Descriptions.Item>
        <Descriptions.Item label="更新时间">{instance.updated_at ? formatDateTime(instance.updated_at) : "—"}</Descriptions.Item>
        <Descriptions.Item label="最近探活">{instance.last_ping_at ? formatDateTime(instance.last_ping_at) : "—"}</Descriptions.Item>
      </Descriptions>

      {instance.backup_link ? (
        <>
          <DbmgmtSectionTitle>备份</DbmgmtSectionTitle>
          <Descriptions bordered column={1} size="small">
            <Descriptions.Item label="备份链接">{instance.backup_link}</Descriptions.Item>
          </Descriptions>
        </>
      ) : null}
    </div>
  ) : null;

  const dbTab = (
    <div>
      <Space style={{ marginBottom: 16 }} wrap>
        <Button type="primary" icon={<PlusOutlined />} disabled>
          创建数据库
        </Button>
        <Input placeholder="请输入名称进行搜索" value={dbKeyword} onChange={(e) => setDbKeyword(e.target.value)} style={{ width: 200 }} />
        <Select defaultValue="all" style={{ width: 100 }} options={[{ value: "all", label: "全部" }]} />
        <Button type="primary" icon={<SearchOutlined />}>
          查询
        </Button>
        <Button icon={<ReloadOutlined />} onClick={() => { setDbKeyword(""); void load(); }}>
          刷新重置
        </Button>
      </Space>
      <Table
        rowKey="name"
        dataSource={filteredDbs.map((d, i) => ({ ...d, id: i + 1 }))}
        pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
        columns={[
          { title: "ID", dataIndex: "id", width: 60 },
          { title: "数据库", dataIndex: "name" },
          { title: "字符集", render: () => "—" },
          { title: "校验规则", render: () => "—" },
          { title: "开发负责人", render: () => "—" },
          { title: "DBA", render: () => "—" },
          { title: "授权账号", render: (_, r) => grantAccountsForDb(r.name) },
          { title: "备注", render: () => "—" },
          {
            title: "操作",
            width: 140,
            render: (_, r) => (
              <Space>
                <Button size="small" type="primary" icon={<EditOutlined />} disabled />
                <Button size="small" danger icon={<DeleteOutlined />} disabled />
              </Space>
            ),
          },
        ]}
      />
    </div>
  );

  const usersTab =
    instance?.driver !== "mysql" ? (
      <Alert type="info" showIcon message="仅 MySQL 实例支持用户管理" />
    ) : (
      <div>
        <Space style={{ marginBottom: 16 }} wrap>
          <Button type="primary" icon={<PlusOutlined />} disabled>
            创建用户
          </Button>
          <Input placeholder="请输入用户名称进行搜索" value={userKeyword} onChange={(e) => setUserKeyword(e.target.value)} style={{ width: 220 }} />
          <Select defaultValue="all" style={{ width: 100 }} options={[{ value: "all", label: "全部" }]} />
          <Button type="primary" icon={<SearchOutlined />}>
            搜索
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => { setUserKeyword(""); void loadUsers(); }}>
            刷新重置
          </Button>
        </Space>
        <Table
          rowKey={(r) => `${r.username}@${r.host}`}
          loading={usersLoading}
          dataSource={filteredUsers.map((u, i) => ({ ...u, rowId: i + 1 }))}
          pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
          columns={[
            { title: "ID", dataIndex: "rowId", width: 60 },
            {
              title: "用户名",
              render: (_, r) => `'${r.username}'@'${r.host}'`,
            },
            {
              title: "授权语句",
              dataIndex: "grant_lines",
              render: (lines?: string[]) =>
                lines?.length ? (
                  <div style={{ whiteSpace: "pre-wrap", maxHeight: 120, overflow: "auto" }}>{lines.join("\n")}</div>
                ) : (
                  "—"
                ),
            },
            {
              title: "密码",
              width: 90,
              render: (_, r) =>
                r.has_password && r.id ? (
                  <Button type="link" size="small" onClick={() => void revealPassword(r)}>
                    查看
                  </Button>
                ) : (
                  "—"
                ),
            },
            { title: "备注", dataIndex: "remark", render: (v?: string) => v || "—" },
            {
              title: "操作",
              width: 140,
              render: () => (
                <Space>
                  <Button size="small" type="primary" icon={<EditOutlined />} disabled />
                  <Button size="small" danger icon={<DeleteOutlined />} disabled />
                </Space>
              ),
            },
          ]}
        />
      </div>
    );

  return (
    <Card
      loading={loading}
      title={instance ? `实例详情 · ${formatInstanceLabel(instance)}` : "实例详情"}
      extra={
        <Space>
          <Select style={{ width: 200 }} value={projectId} options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={onProjectChange} />
          <Link to="/dbmgmt/instances">
            <Button>返回列表</Button>
          </Link>
          <Button icon={<ReloadOutlined />} onClick={() => void load()} />
        </Space>
      }
    >
      {instance ? (
        <Tabs
          activeKey={tab}
          onChange={(k) => {
            const next = new URLSearchParams(searchParams);
            next.set("tab", k);
            if (projectId) next.set("project", String(projectId));
            setSearchParams(next);
          }}
          items={[
            { key: "info", label: "实例详情", children: infoTab },
            { key: "databases", label: "DB管理", children: dbTab },
            { key: "users", label: "用户管理", children: usersTab },
          ]}
        />
      ) : null}
    </Card>
  );
}
