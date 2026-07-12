import { InfoCircleOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, Form, Input, Row, Select, Space, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { createDbAccessRequest, listDbInstances, type DbInstance } from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { getUsers } from "../services/users";
import type { UserItem } from "../types/api";

const CHARSET_OPTIONS = [
  { value: "utf8mb4", label: "utf8mb4" },
  { value: "utf8", label: "utf8" },
];

const COLLATION_BY_CHARSET: Record<string, { value: string; label: string }[]> = {
  utf8mb4: [
    { value: "utf8mb4_general_ci", label: "utf8mb4_general_ci" },
    { value: "utf8mb4_bin", label: "utf8mb4_bin" },
    { value: "utf8mb4_unicode_ci", label: "utf8mb4_unicode_ci" },
  ],
  utf8: [
    { value: "utf8_general_ci", label: "utf8_general_ci" },
    { value: "utf8_bin", label: "utf8_bin" },
  ],
};

const WARM_HINTS = [
  "项目：选择项目名称。",
  "实例名称：选择要创建数据库的实例。",
  "数据库名称：由字母、数字、下划线组成，字母开头，字母或数字结尾，最长 50 个字符。",
  "字符集选择：仅支持 utf8、utf8mb4。",
  "校验规则选择：选择相对应的排序字符集。",
  "开发负责人：申请数据库的开发负责人，有问题方便沟通联系。",
  "DBA：负责处理此数据库的 DBA。",
  "备注：请简单的描述一下数据库信息，尽量使用中文。例如：用户中心数据库。",
  "授权 IP：允许访问数据库应用的 IP 地址，多个主机以 | 分隔符分隔，例如：192.168.8.12|172.16.%",
];

function isValidSmartdbsDbName(name: string) {
  if (!name || name.length > 50) return false;
  return /^[a-zA-Z]([a-zA-Z0-9_]*[a-zA-Z0-9])?$/.test(name);
}

function formatInstanceLabel(inst: DbInstance) {
  const driver = (inst.driver || "mysql").toUpperCase();
  return `${driver}-${inst.host}-${inst.port}`;
}

function userLabel(u: UserItem) {
  const name = u.nickname?.trim() || u.username;
  return u.username && u.nickname ? `${name}（${u.username}）` : name;
}

export function DbmgmtDatabaseApplyPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instances, setInstances] = useState<DbInstance[]>([]);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();
  const charset = Form.useWatch("charset", form) ?? "utf8mb4";

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) {
        setProjectId(res.list[0].id);
        form.setFieldValue("project_id", res.list[0].id);
      }
    });
    void getUsers({ page: 1, page_size: 500 }).then((res) => setUsers(res.list ?? []));
  }, [form]);

  const loadInstances = useCallback(async () => {
    if (!projectId) return;
    const res = await listDbInstances(projectId, { page: 1, page_size: 200 });
    const list = res.list ?? [];
    setInstances(list);
    if (list.length && !list.some((i) => i.id === form.getFieldValue("instance_id"))) {
      form.setFieldValue("instance_id", list[0].id);
    }
  }, [projectId, form]);

  useEffect(() => {
    void loadInstances();
  }, [loadInstances]);

  const collationOptions = useMemo(() => COLLATION_BY_CHARSET[charset] ?? COLLATION_BY_CHARSET.utf8mb4, [charset]);

  const userOptions = useMemo(
    () => users.map((u) => ({ value: u.id, label: userLabel(u) })),
    [users],
  );

  const instanceOptions = useMemo(
    () => instances.map((i) => ({ value: i.id, label: formatInstanceLabel(i) })),
    [instances],
  );

  const onProjectChange = (id: number) => {
    setProjectId(id);
    form.setFieldsValue({ instance_id: undefined });
  };

  const onCharsetChange = (v: string) => {
    const first = COLLATION_BY_CHARSET[v]?.[0]?.value;
    form.setFieldValue("collation", first);
  };

  const submit = async () => {
    if (!projectId || submitting) return;
    setSubmitting(true);
    try {
      const values = await form.validateFields();
      await createDbAccessRequest(projectId, {
        instance_id: values.instance_id,
        scope_type: "new_database",
        database_name: values.database_name,
        privileges: ["create_database"],
        charset: values.charset,
        collation: values.collation,
        dev_owner_user_id: values.dev_owner_user_id,
        dba_user_id: values.dba_user_id,
        grant_hosts: values.grant_hosts,
        reason: values.reason,
      });
      message.success("数据库创建申请已提交，请到待审核查看进度");
      form.resetFields();
      form.setFieldsValue({ project_id: projectId, charset: "utf8mb4", collation: "utf8mb4_general_ci" });
    } catch (e) {
      if (e instanceof Error) message.error(e.message);
    } finally {
      setSubmitting(false);
    }
  };

  const reset = () => {
    form.resetFields();
    form.setFieldsValue({ project_id: projectId, charset: "utf8mb4", collation: "utf8mb4_general_ci" });
  };

  return (
    <Card title="数据库创建申请">
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message={
          <>
            审批通过后将自动 CREATE DATABASE 并写入平台授权。进度请到{" "}
            <Link to="/dbmgmt/workflow/pending">待审核</Link> 查看。
          </>
        }
      />
      <Row gutter={24}>
        <Col xs={24} lg={7}>
          <div
            style={{
              background: "#fdf6ec",
              border: "1px solid #faecd8",
              borderRadius: 8,
              padding: "16px 20px",
              minHeight: 480,
            }}
          >
            <Space align="start" style={{ marginBottom: 12 }}>
              <InfoCircleOutlined style={{ color: "#e6a23c", fontSize: 18, marginTop: 2 }} />
              <Typography.Title level={5} style={{ margin: 0, color: "#e6a23c" }}>
                温馨提示
              </Typography.Title>
            </Space>
            <ol style={{ margin: 0, paddingLeft: 20, color: "#606266", lineHeight: 1.9, fontSize: 13 }}>
              {WARM_HINTS.map((hint) => (
                <li key={hint}>{hint}</li>
              ))}
            </ol>
          </div>
        </Col>
        <Col xs={24} lg={17}>
          <Form
            form={form}
            layout="horizontal"
            labelCol={{ span: 5 }}
            wrapperCol={{ span: 16 }}
            initialValues={{ charset: "utf8mb4", collation: "utf8mb4_general_ci" }}
            onFinish={() => void submit()}
          >
            <Form.Item name="project_id" label="项目名称" rules={[{ required: true, message: "请选择项目" }]}>
              <Select
                placeholder="请选择项目"
                options={projects.map((p) => ({ value: p.id, label: p.name }))}
                onChange={onProjectChange}
              />
            </Form.Item>
            <Form.Item name="instance_id" label="实例名称" rules={[{ required: true, message: "请选择实例" }]}>
              <Select showSearch optionFilterProp="label" placeholder="请选择实例" options={instanceOptions} />
            </Form.Item>
            <Form.Item
              name="database_name"
              label="数据库名"
              rules={[
                { required: true, message: "请填写数据库名" },
                {
                  validator: (_, value: string) => {
                    if (!value || isValidSmartdbsDbName(value.trim())) return Promise.resolve();
                    return Promise.reject(new Error("库名须字母开头，字母或数字结尾，最长 50 字符"));
                  },
                },
              ]}
              extra="由字母、数字、下划线组成，字母开头，字母或数字结尾，最长 50 个字符"
            >
              <Input placeholder="由字母、数字、下划线组成，字母开头，字母或数字结尾，最长 50 个字符" />
            </Form.Item>
            <Form.Item name="charset" label="字符集" rules={[{ required: true, message: "请选择字符集" }]}>
              <Select options={CHARSET_OPTIONS} onChange={onCharsetChange} />
            </Form.Item>
            <Form.Item name="collation" label="校验规则" rules={[{ required: true, message: "请选择校验规则" }]}>
              <Select options={collationOptions} placeholder="选择相对应的排序字符集" />
            </Form.Item>
            <Form.Item name="dev_owner_user_id" label="开发负责人" rules={[{ required: true, message: "请选择开发负责人" }]}>
              <Select showSearch optionFilterProp="label" placeholder="请选择开发负责人" options={userOptions} />
            </Form.Item>
            <Form.Item name="dba_user_id" label="DBA" rules={[{ required: true, message: "请选择 DBA" }]}>
              <Select showSearch optionFilterProp="label" placeholder="请输入负责此 DB 的 dba" options={userOptions} />
            </Form.Item>
            <Form.Item name="reason" label="备注" rules={[{ required: true, message: "请填写备注" }]}>
              <Input placeholder="描述信息" />
            </Form.Item>
            <Form.Item
              name="grant_hosts"
              label="授权 IP"
              rules={[{ required: true, message: "请填写授权 IP" }]}
              extra="多个主机以 | 分隔，例如：192.168.8.12|172.16.%"
            >
              <Input.TextArea rows={3} placeholder="允许访问数据库应用的 IP 地址，多个主机以 | 分隔符分隔，例如：192.168.8.12|172.16.%" />
            </Form.Item>
            <Form.Item wrapperCol={{ offset: 5, span: 16 }}>
              <Space size="middle">
                <Button type="primary" htmlType="submit" loading={submitting}>
                  提交
                </Button>
                <Button danger onClick={reset}>
                  重置
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Col>
      </Row>
    </Card>
  );
}
