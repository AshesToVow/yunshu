// @ts-nocheck
import { ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Form, Input, Radio, Select, Space, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from '@umijs/max';
import {
  APPLY_TYPE_OPTIONS,
  PRIV_LEVEL_OPTIONS,
  PrivilegeCheckboxGroups,
  formatInstanceLabel,
  parseMysqlUserKey,
  type AppUserApplyType,
} from "../components/dbmgmt/dbmgmt-ui-shared";
import {
  createDbAppUserRequest,
  getInstanceMySQLUserPrivileges,
  listDbDatabases,
  listDbInstances,
  listInstanceMySQLUsers,
  type DbInstance,
  type DbInstanceMySQLUser,
} from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { MYSQL_GLOBAL_ONLY_PRIVS, parseMysqlGrantPrivileges } from "../utils/mysql-grants";

export function DbmgmtAppUserApplyPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instances, setInstances] = useState<DbInstance[]>([]);
  const [databases, setDatabases] = useState<string[]>([]);
  const [mysqlUsers, setMysqlUsers] = useState<DbInstanceMySQLUser[]>([]);
  const [heldPrivs, setHeldPrivs] = useState<string[]>([]);
  const [privsLoading, setPrivsLoading] = useState(false);
  const [privsLoaded, setPrivsLoaded] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();
  const instanceId = Form.useWatch("instance_id", form);
  const applyType = (Form.useWatch("apply_type", form) ?? "new_user") as AppUserApplyType;
  const privLevel = Form.useWatch("priv_level", form) ?? "global";
  const mysqlUserKey = Form.useWatch("mysql_user_key", form) as string | undefined;
  const databaseName = Form.useWatch("database_name", form) as string | undefined;

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) {
        setProjectId(res.list[0].id);
        form.setFieldValue("project_id", res.list[0].id);
      }
    });
  }, [form]);

  const loadInstances = useCallback(async () => {
    if (!projectId) return;
    const res = await listDbInstances(projectId, { page: 1, page_size: 200 });
    const list = (res.list ?? []).filter((i) => i.driver === "mysql");
    setInstances(list);
  }, [projectId]);

  useEffect(() => {
    void loadInstances();
  }, [loadInstances]);

  useEffect(() => {
    if (!projectId || !instanceId) {
      setDatabases([]);
      setMysqlUsers([]);
      return;
    }
    void listDbDatabases(projectId, instanceId).then((dbs) => setDatabases((dbs ?? []).map((d) => d.name)));
    void listInstanceMySQLUsers(projectId, instanceId)
      .then((list) => setMysqlUsers(list ?? []))
      .catch(() => setMysqlUsers([]));
  }, [projectId, instanceId]);

  const isNewUser = applyType === "new_user";
  const isExistingUser = applyType === "add_ip" || applyType === "add_priv" || applyType === "revoke";
  const showPrivs = applyType !== "add_ip";
  const showDatabase = privLevel === "database" && showPrivs;
  const showGrantIp = applyType === "add_ip";
  const showExtraHosts = applyType === "add_priv";

  useEffect(() => {
    if (!projectId || !instanceId || !isExistingUser || !mysqlUserKey || !showPrivs) {
      setHeldPrivs([]);
      setPrivsLoaded(false);
      return;
    }
    if (privLevel === "database" && !databaseName) {
      setHeldPrivs([]);
      setPrivsLoaded(false);
      return;
    }
    const parsed = parseMysqlUserKey(mysqlUserKey);
    const selectedUser = mysqlUsers.find((u) => u.username === parsed.user && u.host === parsed.host);
    let cancelled = false;
    setPrivsLoading(true);
    setPrivsLoaded(false);
    void getInstanceMySQLUserPrivileges(projectId, instanceId, {
      mysql_user: parsed.user,
      mysql_host: parsed.host,
      priv_level: privLevel,
      database: databaseName,
    })
      .then((res) => {
        if (cancelled) return;
        let privs = (res.privileges ?? []).map((p) => p.toUpperCase());
        if (privs.length === 0 && selectedUser?.grant_lines?.length) {
          privs = [...parseMysqlGrantPrivileges(selectedUser.grant_lines, {
            level: privLevel as "global" | "database",
            database: databaseName,
          })];
        }
        setHeldPrivs(privs);
        setPrivsLoaded(true);
      })
      .catch(() => {
        if (cancelled) return;
        if (selectedUser?.grant_lines?.length) {
          setHeldPrivs([
            ...parseMysqlGrantPrivileges(selectedUser.grant_lines, {
              level: privLevel as "global" | "database",
              database: databaseName,
            }),
          ]);
        } else {
          setHeldPrivs([]);
        }
        setPrivsLoaded(true);
      })
      .finally(() => {
        if (!cancelled) setPrivsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, instanceId, isExistingUser, mysqlUserKey, showPrivs, privLevel, databaseName, mysqlUsers]);

  const instanceOptions = useMemo(() => instances.map((i) => ({ value: i.id, label: formatInstanceLabel(i) })), [instances]);
  const userOptions = useMemo(
    () => mysqlUsers.map((u) => ({ value: `${u.username}@${u.host}`, label: `${u.username}@${u.host}` })),
    [mysqlUsers],
  );

  const existingPrivs = useMemo(() => new Set(heldPrivs), [heldPrivs]);

  const isRevoke = applyType === "revoke";

  const lockedPrivs = useMemo(() => (applyType === "add_priv" ? heldPrivs : []), [applyType, heldPrivs]);

  const globalOnlyDisabled = useMemo(
    () => (privLevel === "database" && applyType !== "revoke" ? [...MYSQL_GLOBAL_ONLY_PRIVS] : []),
    [privLevel, applyType],
  );

  const disabledPrivs = useMemo(() => {
    if (isRevoke) return [];
    const base = applyType === "add_priv" ? heldPrivs : [];
    return [...base, ...globalOnlyDisabled];
  }, [applyType, isRevoke, heldPrivs, globalOnlyDisabled]);

  const revokeAllowedPrivs = useMemo(() => (isRevoke ? heldPrivs : undefined), [isRevoke, heldPrivs]);

  useEffect(() => {
    if (!showPrivs) return;
    const cur: string[] = form.getFieldValue("privileges") ?? [];
    if (isRevoke) {
      const allowed = new Set(heldPrivs.map((p) => p.toUpperCase()));
      const filtered = cur.filter((p) => allowed.has(String(p).toUpperCase()));
      if (filtered.length !== cur.length) {
        form.setFieldValue("privileges", filtered);
      }
      return;
    }
    const blocked = new Set([...lockedPrivs, ...disabledPrivs].map((p) => p.toUpperCase()));
    const filtered = cur.filter((p) => !blocked.has(String(p).toUpperCase()));
    if (filtered.length !== cur.length) {
      form.setFieldValue("privileges", filtered);
    }
  }, [form, showPrivs, isRevoke, heldPrivs, lockedPrivs, disabledPrivs]);

  const onProjectChange = (id: number) => {
    setProjectId(id);
    form.setFieldsValue({ instance_id: undefined, mysql_user_key: undefined });
  };

  const submit = async () => {
    if (!projectId || submitting) return;
    setSubmitting(true);
    try {
      const values = await form.validateFields();
      let mysqlUser = values.mysql_user as string | undefined;
      let mysqlHost = "%";
      if (values.mysql_user_key) {
        const parsed = parseMysqlUserKey(values.mysql_user_key);
        mysqlUser = parsed.user;
        mysqlHost = parsed.host;
      }
      await createDbAppUserRequest(projectId, {
        instance_id: values.instance_id,
        apply_type: values.apply_type,
        mysql_user: mysqlUser!,
        mysql_host: mysqlHost,
        database_name: values.database_name,
        priv_level: values.priv_level || "global",
        privileges: values.privileges ?? [],
        grant_hosts: values.grant_hosts,
        reason: values.reason,
      });
      message.success("已提交应用用户权限申请");
      reset();
    } catch (e) {
      if (e instanceof Error) message.error(e.message);
    } finally {
      setSubmitting(false);
    }
  };

  const reset = () => {
    form.resetFields();
    form.setFieldsValue({ project_id: projectId, apply_type: "new_user", priv_level: "global", privileges: [] });
  };

  return (
    <Card
      title="应用用户权限申请"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void loadInstances()}>
            刷新
          </Button>
        </Space>
      }
    >
      <Form
        form={form}
        layout="horizontal"
        labelCol={{ span: 4 }}
        wrapperCol={{ span: 16 }}
        initialValues={{ apply_type: "new_user", priv_level: "global", privileges: [] }}
        onFinish={() => void submit()}
      >
        <Form.Item name="apply_type" label="权限类型" rules={[{ required: true }]}>
          <Radio.Group
            optionType="button"
            buttonStyle="solid"
            options={APPLY_TYPE_OPTIONS.map((o) => ({ value: o.value, label: o.label }))}
            onChange={() => form.setFieldsValue({ mysql_user: undefined, mysql_user_key: undefined, privileges: [], database_name: undefined })}
          />
        </Form.Item>

        <Form.Item name="project_id" label="项目名称" rules={[{ required: true, message: "请选择项目" }]}>
          <Select options={projects.map((p) => ({ value: p.id, label: p.name }))} onChange={onProjectChange} />
        </Form.Item>

        <Form.Item name="instance_id" label="实例名称" rules={[{ required: true, message: "请选择实例" }]}>
          <Select showSearch optionFilterProp="label" options={instanceOptions} placeholder="请选择 MySQL 实例" />
        </Form.Item>

        {isNewUser ? (
          <Form.Item
            name="mysql_user"
            label="应用用户"
            rules={[
              { required: true, message: "请填写应用用户名" },
              { pattern: /^[a-zA-Z0-9_]+$/, message: "用户名仅允许字母、数字和下划线" },
              { max: 32, message: "用户名不超过 32 个字符" },
            ]}
            extra={
              <Alert
                type="info"
                showIcon
                style={{ marginTop: 8 }}
                message="用户名命名规范：建议以数据库名称命名，推荐格式为 数据库名_"
              />
            }
          >
            <Input placeholder="应用用户" />
          </Form.Item>
        ) : null}

        {isExistingUser ? (
          <Form.Item name="mysql_user_key" label="应用用户" rules={[{ required: true, message: "请选择应用用户" }]}>
            <Select showSearch optionFilterProp="label" options={userOptions} placeholder="应用用户" />
          </Form.Item>
        ) : null}

        {showPrivs ? (
          <>
            <Form.Item name="priv_level" label="权限级别" rules={[{ required: true }]}>
              <Select options={PRIV_LEVEL_OPTIONS} />
            </Form.Item>
            {showDatabase ? (
              <Form.Item name="database_name" label="数据库" rules={[{ required: true, message: "请选择数据库" }]}>
                <Select showSearch options={databases.map((d) => ({ value: d, label: d }))} placeholder="选择目标库" />
              </Form.Item>
            ) : null}
            <Form.Item
              name="privileges"
              label={isRevoke ? "回收权限" : "用户权限"}
              rules={[
                {
                  validator: async (_, value: string[] | undefined) => {
                    if (isRevoke) {
                      if (!privsLoaded) throw new Error("正在加载已有权限，请稍候");
                      if (heldPrivs.length === 0) throw new Error("该用户在当前级别下无可回收的业务权限");
                      if (!value?.length) throw new Error("请勾选需要回收的权限");
                      return;
                    }
                    if (!value?.length) throw new Error("请至少选择一项权限");
                  },
                },
              ]}
              labelCol={{ span: 4 }}
              wrapperCol={{ span: 20 }}
              extra={
                isRevoke ? (
                  privsLoading ? (
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      正在查询该用户已有权限…
                    </Typography.Text>
                  ) : privsLoaded && heldPrivs.length > 0 ? (
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      该用户已持有 {heldPrivs.length} 项权限，请勾选需要回收的项（USAGE 等基础权限不在列表中）
                    </Typography.Text>
                  ) : privsLoaded ? (
                    <Alert
                      type="warning"
                      showIcon
                      style={{ marginTop: 8 }}
                      message="未查询到可回收的业务权限"
                      description="请确认权限级别、目标库是否正确；若用户仅有 USAGE 等基础权限，则无需回收。"
                    />
                  ) : null
                ) : existingPrivs.size > 0 ? (
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    该用户已持有 {existingPrivs.size} 项权限（已置灰），请仅勾选需要新增的权限
                  </Typography.Text>
                ) : privLevel === "database" ? (
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    库级权限下，「管理」类中的 SUPER/PROCESS 等仅全局权限项已置灰；若需授予请选择「全局权限」级别。
                  </Typography.Text>
                ) : null
              }
            >
              {isRevoke && privsLoaded && heldPrivs.length === 0 ? (
                <Typography.Text type="secondary">暂无可选权限</Typography.Text>
              ) : (
                <PrivilegeCheckboxGroups
                  disabled={privsLoading || (isRevoke && heldPrivs.length === 0)}
                  disabledPrivs={disabledPrivs}
                  lockedPrivs={lockedPrivs}
                  allowedPrivs={revokeAllowedPrivs}
                />
              )}
            </Form.Item>
          </>
        ) : null}

        {showGrantIp ? (
          <Form.Item
            name="grant_hosts"
            label="授权IP"
            rules={[{ required: true, message: "请填写授权 IP" }]}
            extra={
              <Alert
                type="info"
                showIcon
                style={{ marginTop: 8 }}
                message="多个主机以 | 分隔符分隔，例如：192.168.8.12|172.16.%"
              />
            }
          >
            <Input placeholder="请输入允许访问数据库应用的IP地址" />
          </Form.Item>
        ) : null}

        {showExtraHosts ? (
          <Form.Item
            name="grant_hosts"
            label="扩展主机"
            extra={
              <Alert
                type="info"
                showIcon
                style={{ marginTop: 8 }}
                message="可选。为同一用户名新增其他 @host 并授予相同权限时使用，多个以 | 分隔；留空则仅对上方已选用户（如 monitor@10.10.10.103）追加权限。"
              />
            }
          >
            <Input placeholder="例如：10.10.10.1|172.16.%（留空则只作用于已选用户）" />
          </Form.Item>
        ) : null}

        {isNewUser ? (
          <Form.Item
            name="grant_hosts"
            label="授权IP"
            extra="可选；多个主机以 | 分隔。留空则默认 %"
          >
            <Input placeholder="例如：192.168.8.12|172.16.%" />
          </Form.Item>
        ) : null}

        <Form.Item
          name="reason"
          label="备注"
          rules={[{ required: true, message: "请填写备注" }]}
          extra={
            <span style={{ color: "#1890ff" }}>请简单的描述一下数据库信息，尽量使用中文，例如：用户中心数据库</span>
          }
        >
          <Input placeholder="描述信息" />
        </Form.Item>

        <Form.Item wrapperCol={{ offset: 4, span: 16 }}>
          <Space size="large">
            <Button type="primary" htmlType="submit" loading={submitting}>
              提交
            </Button>
            <Button danger onClick={reset}>
              重置
            </Button>
            <Link to="/workflow/inbox?domain=dbmgmt">查看我的待办</Link>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
}
