import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  KeyOutlined,
  PlusOutlined,
  ReloadOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { DbmgmtSectionTitle, formatInstanceLabel } from "../components/dbmgmt/dbmgmt-ui-shared";
import {
  getDbInstance,
  getInstanceAccountPassword,
  listDbColumnMaskRules,
  listDbDatabases,
  listInstanceMySQLUsers,
  upsertDbColumnMaskRule,
  deleteDbColumnMaskRule,
  type DbColumnMaskRule,
  type DbInstance,
  type DbInstanceMySQLUser,
} from "../services/dbmgmt";
import { extractApiErrorMessage } from "../services/http";
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
  const navigate = useNavigate();
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
  const [dbSearch, setDbSearch] = useState("");
  const [userKeyword, setUserKeyword] = useState("");
  const [userSearch, setUserSearch] = useState("");
  const [dbGrantFilter, setDbGrantFilter] = useState<"all" | "granted" | "ungranted">("all");
  const [userGrantFilter, setUserGrantFilter] = useState<"all" | "granted" | "ungranted">("all");
  const [maskRules, setMaskRules] = useState<DbColumnMaskRule[]>([]);
  const [maskLoading, setMaskLoading] = useState(false);
  const [maskOpen, setMaskOpen] = useState(false);
  const [maskSubmitting, setMaskSubmitting] = useState(false);
  const [deletingRuleId, setDeletingRuleId] = useState<number | null>(null);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [passwordUser, setPasswordUser] = useState<DbInstanceMySQLUser>();
  const [passwordValue, setPasswordValue] = useState("");
  const [revealingId, setRevealingId] = useState<number | null>(null);
  const [maskForm] = Form.useForm();
  const loadSeqRef = useRef(0);
  const usersSeqRef = useRef(0);
  const maskSeqRef = useRef(0);
  const tab = searchParams.get("tab") ?? "info";

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 })
      .then((res) => {
        setProjects(res.list ?? []);
        if (projectFromUrl) {
          setProjectId(projectFromUrl);
        } else if (res.list?.length) {
          setProjectId(res.list[0].id);
        }
      })
      .catch(() => undefined);
  }, [projectFromUrl]);

  const load = useCallback(async () => {
    const seq = ++loadSeqRef.current;
    if (!projectId || !instanceId) return;
    setLoading(true);
    try {
      const inst = await getDbInstance(projectId, instanceId);
      if (seq !== loadSeqRef.current) return;
      setInstance(inst);
      const dbs = await listDbDatabases(projectId, instanceId);
      if (seq !== loadSeqRef.current) return;
      setDatabases(dbs ?? []);
    } catch (e) {
      if (seq !== loadSeqRef.current) return;
      message.error(extractApiErrorMessage(e, "加载实例失败"));
    } finally {
      if (seq === loadSeqRef.current) setLoading(false);
    }
  }, [projectId, instanceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const loadUsers = useCallback(async () => {
    const seq = ++usersSeqRef.current;
    if (!projectId || !instanceId || instance?.driver !== "mysql") return;
    setUsersLoading(true);
    try {
      const list = await listInstanceMySQLUsers(projectId, instanceId);
      if (seq !== usersSeqRef.current) return;
      setUsers(list ?? []);
    } catch (e) {
      if (seq !== usersSeqRef.current) return;
      message.error(extractApiErrorMessage(e, "加载用户失败"));
    } finally {
      if (seq === usersSeqRef.current) setUsersLoading(false);
    }
  }, [projectId, instanceId, instance?.driver]);

  const loadMaskRules = useCallback(async () => {
    const seq = ++maskSeqRef.current;
    if (!projectId || !instanceId) return;
    setMaskLoading(true);
    try {
      const rules = await listDbColumnMaskRules(projectId, instanceId);
      if (seq !== maskSeqRef.current) return;
      setMaskRules(rules ?? []);
    } catch (e) {
      if (seq !== maskSeqRef.current) return;
      message.error(extractApiErrorMessage(e, "加载脱敏规则失败"));
    } finally {
      if (seq === maskSeqRef.current) setMaskLoading(false);
    }
  }, [projectId, instanceId]);

  useEffect(() => {
    if (tab === "users" || tab === "databases") void loadUsers();
    if (tab === "mask") void loadMaskRules();
  }, [tab, loadUsers, loadMaskRules]);

  const closePassword = () => {
    setPasswordOpen(false);
    setPasswordUser(undefined);
    setPasswordValue("");
  };

  const revealPassword = async (row: DbInstanceMySQLUser) => {
    if (!projectId || !instanceId || !row.id || revealingId !== null) return;
    setRevealingId(row.id);
    try {
      const res = await getInstanceAccountPassword(projectId, instanceId, row.id);
      setPasswordUser(row);
      setPasswordValue(res.password ?? "");
      setPasswordOpen(true);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "查看密码失败"));
    } finally {
      setRevealingId(null);
    }
  };

  const copyPassword = async () => {
    if (!passwordValue) return;
    try {
      await navigator.clipboard.writeText(passwordValue);
      message.success("密码已复制");
    } catch {
      message.error("复制失败，请手动选择复制");
    }
  };

  const submitMaskRule = async () => {
    if (!projectId || !instanceId || maskSubmitting) return;
    let values: { schema_name?: string; table_name: string; column_name: string; mask_type: string; pattern?: string };
    try {
      values = await maskForm.validateFields();
    } catch {
      return;
    }
    setMaskSubmitting(true);
    try {
      await upsertDbColumnMaskRule(projectId, instanceId, values);
      message.success("已保存");
      setMaskOpen(false);
      await loadMaskRules();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "保存脱敏规则失败"));
    } finally {
      setMaskSubmitting(false);
    }
  };

  const removeMaskRule = async (ruleId: number) => {
    if (!projectId || !instanceId || deletingRuleId !== null) return;
    setDeletingRuleId(ruleId);
    try {
      await deleteDbColumnMaskRule(projectId, instanceId, ruleId);
      message.success("已删除");
      await loadMaskRules();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "删除脱敏规则失败"));
    } finally {
      setDeletingRuleId(null);
    }
  };

  const onProjectChange = (id: number) => {
    setProjectId(id);
    setInstance(undefined);
    setDatabases([]);
    setUsers([]);
    setMaskRules([]);
    setDbKeyword("");
    setDbSearch("");
    setUserKeyword("");
    setUserSearch("");
    setDbGrantFilter("all");
    setUserGrantFilter("all");
    loadSeqRef.current += 1;
    usersSeqRef.current += 1;
    maskSeqRef.current += 1;
    setLoading(false);
    setUsersLoading(false);
    setMaskLoading(false);
    const next = new URLSearchParams(searchParams);
    next.set("project", String(id));
    if (tab) next.set("tab", tab);
    setSearchParams(next);
  };

  const projectName = projects.find((p) => p.id === (instance?.project_id ?? projectId))?.name ?? "—";

  const grantAccountsForDb = useCallback(
    (dbName: string) => {
      const lines: string[] = [];
      for (const u of users) {
        const grants = u.grant_lines ?? [];
        if (grants.some((g) => g.includes(`\`${dbName}\``) || g.includes(`${dbName}.`))) {
          lines.push(`'${u.username}'@'${u.host}'`);
        }
      }
      return lines.length ? lines.join(" ") : "—";
    },
    [users],
  );

  const filteredDbs = useMemo(() => {
    const kw = dbSearch.trim().toLowerCase();
    return databases.filter((d) => {
      if (kw && !d.name.toLowerCase().includes(kw)) return false;
      if (dbGrantFilter !== "all") {
        const granted = grantAccountsForDb(d.name) !== "—";
        if (dbGrantFilter === "ungranted" ? granted : !granted) return false;
      }
      return true;
    });
  }, [databases, dbSearch, dbGrantFilter, grantAccountsForDb]);

  const filteredUsers = useMemo(() => {
    const kw = userSearch.trim().toLowerCase();
    return users.filter((u) => {
      if (kw && !`${u.username}@${u.host}`.toLowerCase().includes(kw)) return false;
      if (userGrantFilter !== "all") {
        const granted = (u.grant_lines?.length ?? 0) > 0;
        if (userGrantFilter === "ungranted" ? granted : !granted) return false;
      }
      return true;
    });
  }, [users, userSearch, userGrantFilter]);

  // 账号与库的创建/变更/回收都必须走申请审批，这里统一构造跳转参数。
  const gotoAppUserApply = (extra: Record<string, string | number | undefined>) => {
    const q = new URLSearchParams();
    if (projectId) q.set("project", String(projectId));
    q.set("instanceId", String(instanceId));
    for (const [k, v] of Object.entries(extra)) {
      if (v !== undefined && v !== "") q.set(k, String(v));
    }
    navigate(`/dbmgmt/apply/app-user?${q.toString()}`);
  };

  const gotoDatabaseApply = () => {
    if (!projectId || !instanceId) {
      message.warning("请先选择项目");
      return;
    }
    const q = new URLSearchParams({ project: String(projectId), instanceId: String(instanceId) });
    navigate(`/dbmgmt/apply/database?${q.toString()}`);
  };

  const gotoQueryConsole = (dbName: string) => {
    const q = new URLSearchParams({ instanceId: String(instanceId), database: dbName });
    if (projectId) q.set("project", String(projectId));
    navigate(`/dbmgmt/sql/query?${q.toString()}`);
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
            ) : instance.primary_instance_id ? (
              `#${instance.primary_instance_id}`
            ) : (
              "—"
            )}
          </Descriptions.Item>
        ) : null}
        <Descriptions.Item label="只读">{instance.read_only ? "是" : "否"}</Descriptions.Item>
        <Descriptions.Item label="DML需工单">{instance.require_ticket_for_dml ? "是" : "否"}</Descriptions.Item>
        {instance.tags ? <Descriptions.Item label="标签">{instance.tags}</Descriptions.Item> : null}
        {instance.remark ? (
          <Descriptions.Item label="备注" span={3}>
            {instance.remark}
          </Descriptions.Item>
        ) : null}
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
        <Descriptions.Item label="创建时间">
          {instance.created_at ? formatDateTime(instance.created_at) : "—"}
        </Descriptions.Item>
        <Descriptions.Item label="更新时间">
          {instance.updated_at ? formatDateTime(instance.updated_at) : "—"}
        </Descriptions.Item>
        <Descriptions.Item label="最近探活">
          {instance.last_ping_at ? formatDateTime(instance.last_ping_at) : "—"}
        </Descriptions.Item>
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
      <div className="toolbar">
        <Space wrap>
          <Button type="primary" icon={<PlusOutlined />} onClick={gotoDatabaseApply}>
            创建数据库
          </Button>
          <Input
            allowClear
            placeholder="请输入名称进行搜索"
            value={dbKeyword}
            onChange={(e) => {
              setDbKeyword(e.target.value);
              if (!e.target.value) setDbSearch("");
            }}
            onPressEnter={() => setDbSearch(dbKeyword.trim())}
            style={{ width: 200 }}
          />
          <Select
            value={dbGrantFilter}
            style={{ width: 140 }}
            onChange={setDbGrantFilter}
            options={[
              { value: "all", label: "全部" },
              { value: "granted", label: "已授权账号" },
              { value: "ungranted", label: "无授权账号" },
            ]}
          />
          <Button type="primary" icon={<SearchOutlined />} onClick={() => setDbSearch(dbKeyword.trim())}>
            查询
          </Button>
          <Button
            icon={<ReloadOutlined />}
            loading={loading}
            disabled={loading}
            onClick={() => {
              setDbKeyword("");
              setDbSearch("");
              setDbGrantFilter("all");
              void load();
            }}
          >
            刷新重置
          </Button>
        </Space>
      </div>
      <Table
        rowKey="name"
        loading={loading}
        dataSource={filteredDbs.map((d, i) => ({ ...d, id: i + 1 }))}
        pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
        scroll={{ x: 1200 }}
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
            width: 180,
            fixed: "right",
            render: (_, r) => (
              <Space>
                <Button
                  size="small"
                  type="primary"
                  icon={<KeyOutlined />}
                  title="为该库申请应用用户授权"
                  onClick={() =>
                    gotoAppUserApply({ apply_type: "new_user", database: r.name, priv_level: "database" })
                  }
                >
                  授权
                </Button>
                <Button size="small" icon={<SearchOutlined />} onClick={() => gotoQueryConsole(r.name)}>
                  查询
                </Button>
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
        <div className="toolbar">
          <Space wrap>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => gotoAppUserApply({ apply_type: "new_user" })}
            >
              创建用户
            </Button>
            <Input
              allowClear
              placeholder="请输入用户名称进行搜索"
              value={userKeyword}
              onChange={(e) => {
                setUserKeyword(e.target.value);
                if (!e.target.value) setUserSearch("");
              }}
              onPressEnter={() => setUserSearch(userKeyword.trim())}
              style={{ width: 220 }}
            />
            <Select
              value={userGrantFilter}
              style={{ width: 140 }}
              onChange={setUserGrantFilter}
              options={[
                { value: "all", label: "全部" },
                { value: "granted", label: "有库授权" },
                { value: "ungranted", label: "无库授权" },
              ]}
            />
            <Button type="primary" icon={<SearchOutlined />} onClick={() => setUserSearch(userKeyword.trim())}>
              搜索
            </Button>
            <Button
              icon={<ReloadOutlined />}
              loading={usersLoading}
              disabled={usersLoading}
              onClick={() => {
                setUserKeyword("");
                setUserSearch("");
                setUserGrantFilter("all");
                void loadUsers();
              }}
            >
              刷新重置
            </Button>
          </Space>
        </div>
        <Table
          rowKey={(r) => `${r.username}@${r.host}`}
          loading={usersLoading}
          dataSource={filteredUsers.map((u, i) => ({ ...u, rowId: i + 1 }))}
          pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
          scroll={{ x: 1000 }}
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
                  <Button
                    type="link"
                    size="small"
                    loading={revealingId === r.id}
                    disabled={revealingId !== null}
                    onClick={() => void revealPassword(r)}
                  >
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
              fixed: "right",
              render: (_, r) => {
                const userKey = `${r.username}@${r.host}`;
                return (
                  <Space>
                    <Button
                      size="small"
                      type="primary"
                      icon={<EditOutlined />}
                      title="申请为该账号新增权限"
                      onClick={() => gotoAppUserApply({ apply_type: "add_priv", user: userKey })}
                    />
                    <Popconfirm
                      title="提交权限回收申请？"
                      description="回收需审批通过后才会执行"
                      onConfirm={() => gotoAppUserApply({ apply_type: "revoke", user: userKey })}
                    >
                      <Button size="small" danger icon={<DeleteOutlined />} title="申请回收该账号权限" />
                    </Popconfirm>
                  </Space>
                );
              },
            },
          ]}
        />
      </div>
    );

  const maskTab = (
    <div>
      <div className="toolbar">
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              maskForm.resetFields();
              maskForm.setFieldsValue({ mask_type: "partial", pattern: "3,4" });
              setMaskOpen(true);
            }}
          >
            新增脱敏规则
          </Button>
          <Button
            icon={<ReloadOutlined />}
            loading={maskLoading}
            disabled={maskLoading}
            onClick={() => void loadMaskRules()}
          />
        </Space>
      </div>
      <Table
        rowKey="id"
        loading={maskLoading}
        dataSource={maskRules}
        pagination={{ pageSize: 10 }}
        scroll={{ x: 800 }}
        columns={[
          { title: "库/Schema", dataIndex: "schema_name", width: 120 },
          { title: "表", dataIndex: "table_name", width: 140 },
          { title: "列", dataIndex: "column_name", width: 140 },
          { title: "类型", dataIndex: "mask_type", width: 100 },
          { title: "模式", dataIndex: "pattern", width: 100 },
          {
            title: "操作",
            width: 80,
            fixed: "right",
            render: (_: unknown, r: DbColumnMaskRule) => (
              <Popconfirm
                title="删除该规则？"
                onConfirm={() => void removeMaskRule(r.id)}
                disabled={deletingRuleId !== null}
              >
                <Button
                  type="link"
                  danger
                  size="small"
                  icon={<DeleteOutlined />}
                  loading={deletingRuleId === r.id}
                  disabled={deletingRuleId !== null}
                />
              </Popconfirm>
            ),
          },
        ]}
      />
      <Modal
        title="列级脱敏规则"
        open={maskOpen}
        onCancel={() => {
          if (!maskSubmitting) setMaskOpen(false);
        }}
        onOk={() => void submitMaskRule()}
        confirmLoading={maskSubmitting}
        cancelButtonProps={{ disabled: maskSubmitting }}
        maskClosable={!maskSubmitting}
      >
        <Form form={maskForm} layout="vertical">
          <Form.Item label="Schema（库名）" name="schema_name">
            <Input placeholder="可选，MySQL 填库名" />
          </Form.Item>
          <Form.Item label="表名" name="table_name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="列名" name="column_name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="脱敏类型" name="mask_type" rules={[{ required: true }]}>
            <Select
              options={[
                { value: "partial", label: "partial（保留前后字符）" },
                { value: "hash", label: "hash" },
                { value: "redact", label: "redact（***）" },
              ]}
            />
          </Form.Item>
          <Form.Item label="partial 模式" name="pattern" extra="如 3,4 表示保留前3后4">
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );

  return (
    <Card
      className="table-card"
      loading={loading}
      title={instance ? `实例详情 · ${formatInstanceLabel(instance)}` : "实例详情"}
      extra={
        <Space>
          <Select
            style={{ width: 200 }}
            value={projectId}
            options={projects.map((p) => ({ value: p.id, label: p.name }))}
            onChange={onProjectChange}
          />
          <Link to="/dbmgmt/instances">
            <Button>返回列表</Button>
          </Link>
          <Button icon={<ReloadOutlined />} loading={loading} disabled={loading} onClick={() => void load()} />
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
            { key: "mask", label: "列脱敏", children: maskTab },
          ]}
        />
      ) : null}
      <Modal
        title={passwordUser ? `${passwordUser.username}@${passwordUser.host} 密码` : "账号密码"}
        open={passwordOpen}
        onCancel={closePassword}
        footer={
          <Button type="primary" onClick={closePassword}>
            关闭
          </Button>
        }
        destroyOnClose
        width={480}
      >
        <Space direction="vertical" style={{ width: "100%" }} size="middle">
          <Input.Password readOnly value={passwordValue} />
          <Button icon={<CopyOutlined />} onClick={() => void copyPassword()} block>
            复制密码
          </Button>
          <Typography.Text type="secondary">密码查看已记录审计日志</Typography.Text>
        </Space>
      </Modal>
    </Card>
  );
}
