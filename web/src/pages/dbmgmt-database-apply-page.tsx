import { InfoCircleOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Col, Form, Input, Row, Select, Space, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { createDbAccessRequest, listDbInstances, type DbInstance } from "../services/dbmgmt";
import { extractApiErrorMessage } from "../services/http";
import { getProjects, type ProjectItem } from "../services/projects";
import { getUsers } from "../services/users";
import type { UserItem } from "../types/api";

const DEFAULT_CHARSET = "utf8mb4";
const DEFAULT_COLLATION = "utf8mb4_general_ci";

const CHARSET_OPTIONS = [
  { value: "utf8mb4", label: "utf8mb4（默认）" },
  { value: "utf8", label: "utf8" },
  { value: "latin1", label: "latin1" },
  { value: "gbk", label: "gbk" },
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
  latin1: [
    { value: "latin1_swedish_ci", label: "latin1_swedish_ci" },
    { value: "latin1_bin", label: "latin1_bin" },
  ],
  gbk: [
    { value: "gbk_chinese_ci", label: "gbk_chinese_ci" },
    { value: "gbk_bin", label: "gbk_bin" },
  ],
};

const WARM_HINTS = [
  "项目：选择项目名称。",
  "实例名称：选择要创建数据库的实例。",
  "数据库名称：由字母、数字、下划线组成，字母开头，字母或数字结尾，最长 50 个字符。",
  "字符集选择：默认 utf8mb4，可选 utf8mb4、utf8、latin1、gbk。",
  "校验规则选择：选择与字符集匹配的排序规则。",
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
  // 实例详情页「创建数据库」跳转时携带项目与实例，减少重复选择。
  const [searchParams] = useSearchParams();
  const presetProjectId = Number(searchParams.get("project")) || undefined;
  const presetInstanceId = Number(searchParams.get("instanceId")) || undefined;
  const presetAppliedRef = useRef(false);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [instances, setInstances] = useState<DbInstance[]>([]);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [instancesLoading, setInstancesLoading] = useState(false);
  const instancesSeqRef = useRef(0);
  const submittingRef = useRef(false);
  const [form] = Form.useForm();
  const charset = Form.useWatch("charset", form) ?? DEFAULT_CHARSET;

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 })
      .then((res) => {
        setProjects(res.list ?? []);
        const preferred = res.list?.some((p) => p.id === presetProjectId) ? presetProjectId : res.list?.[0]?.id;
        if (preferred) {
          setProjectId(preferred);
          form.setFieldValue("project_id", preferred);
        }
      })
      .catch(() => {
        /* 项目列表加载失败不阻断页面 */
      });
    void getUsers({ page: 1, page_size: 500 })
      .then((res) => setUsers(res.list ?? []))
      .catch(() => {
        /* 用户列表加载失败不阻断页面 */
      });
  }, [form, presetProjectId]);

  const loadInstances = useCallback(async () => {
    const seq = ++instancesSeqRef.current;
    if (!projectId) {
      setInstances([]);
      setInstancesLoading(false);
      return;
    }
    setInstancesLoading(true);
    try {
      const res = await listDbInstances(projectId, { page: 1, page_size: 200 });
      if (seq !== instancesSeqRef.current) return;
      const list = res.list ?? [];
      setInstances(list);
      if (list.length && !list.some((i) => i.id === form.getFieldValue("instance_id"))) {
        form.setFieldValue("instance_id", list[0].id);
      }
    } catch (e) {
      if (seq !== instancesSeqRef.current) return;
      setInstances([]);
      message.error(extractApiErrorMessage(e, "加载实例列表失败"));
    } finally {
      if (seq === instancesSeqRef.current) setInstancesLoading(false);
    }
  }, [projectId, form]);

  useEffect(() => {
    void loadInstances();
  }, [loadInstances]);

  useEffect(() => {
    if (presetAppliedRef.current || !presetInstanceId) return;
    if (!instances.some((i) => i.id === presetInstanceId)) return;
    presetAppliedRef.current = true;
    form.setFieldValue("instance_id", presetInstanceId);
  }, [instances, presetInstanceId, form]);

  const collationOptions = useMemo(
    () => COLLATION_BY_CHARSET[charset] ?? COLLATION_BY_CHARSET[DEFAULT_CHARSET],
    [charset],
  );

  const userOptions = useMemo(() => users.map((u) => ({ value: u.id, label: userLabel(u) })), [users]);

  const instanceOptions = useMemo(
    () => instances.map((i) => ({ value: i.id, label: formatInstanceLabel(i) })),
    [instances],
  );

  const onProjectChange = (id: number) => {
    instancesSeqRef.current += 1;
    setProjectId(id);
    setInstances([]);
    form.setFieldsValue({ instance_id: undefined });
  };

  const onCharsetChange = (v: string) => {
    const first = COLLATION_BY_CHARSET[v]?.[0]?.value ?? DEFAULT_COLLATION;
    form.setFieldValue("collation", first);
  };

  const submit = async () => {
    if (!projectId || submittingRef.current) return;
    submittingRef.current = true;
    setSubmitting(true);
    try {
      const values = await form.validateFields();
      const rawCharset = typeof values.charset === "string" ? values.charset.trim().toLowerCase() : "";
      const charsetValue = COLLATION_BY_CHARSET[rawCharset] ? rawCharset : DEFAULT_CHARSET;
      const collationOptionsForCharset = COLLATION_BY_CHARSET[charsetValue];
      const rawCollation = typeof values.collation === "string" ? values.collation.trim() : "";
      const collationValue = collationOptionsForCharset.some((o) => o.value === rawCollation)
        ? rawCollation
        : collationOptionsForCharset[0].value;
      await createDbAccessRequest(projectId, {
        instance_id: values.instance_id,
        scope_type: "new_database",
        database_name: typeof values.database_name === "string" ? values.database_name.trim() : values.database_name,
        privileges: ["create_database"],
        charset: charsetValue,
        collation: collationValue,
        dev_owner_user_id: values.dev_owner_user_id,
        dba_user_id: values.dba_user_id,
        grant_hosts: typeof values.grant_hosts === "string" ? values.grant_hosts.trim() : values.grant_hosts,
        reason: typeof values.reason === "string" ? values.reason.trim() : values.reason,
      });
      message.success("数据库创建申请已提交，请到待审核查看进度");
      form.resetFields();
      form.setFieldsValue({ project_id: projectId, charset: DEFAULT_CHARSET, collation: DEFAULT_COLLATION });
    } catch (e) {
      message.error(extractApiErrorMessage(e, "提交失败"));
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  };

  const reset = () => {
    form.resetFields();
    form.setFieldsValue({ project_id: projectId, charset: DEFAULT_CHARSET, collation: DEFAULT_COLLATION });
  };

  return (
    <Card className="table-card" title="数据库创建申请">
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message={
          <>
            审批通过后将自动 CREATE DATABASE 并写入平台授权。进度请到{" "}
            <Link to="/workflow/inbox?domain=dbmgmt">我的待办</Link> 查看。
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
            initialValues={{ charset: DEFAULT_CHARSET, collation: DEFAULT_COLLATION }}
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
              <Select
                showSearch
                optionFilterProp="label"
                loading={instancesLoading}
                placeholder="请选择实例"
                options={instanceOptions}
              />
            </Form.Item>
            <Form.Item
              name="database_name"
              label="数据库名"
              normalize={(v?: string) => (typeof v === "string" ? v.trim() : v)}
              rules={[
                { required: true, whitespace: true, message: "请填写数据库名" },
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
            <Form.Item
              name="collation"
              label="校验规则"
              rules={[
                { required: true, message: "请选择校验规则" },
                {
                  validator: (_, value?: string) => {
                    const options = COLLATION_BY_CHARSET[charset] ?? [];
                    if (!value || options.some((o) => o.value === value)) return Promise.resolve();
                    return Promise.reject(new Error("校验规则与字符集不匹配，请重新选择"));
                  },
                },
              ]}
            >
              <Select options={collationOptions} placeholder="选择与字符集匹配的排序规则" />
            </Form.Item>
            <Form.Item
              name="dev_owner_user_id"
              label="开发负责人"
              rules={[{ required: true, message: "请选择开发负责人" }]}
            >
              <Select showSearch optionFilterProp="label" placeholder="请选择开发负责人" options={userOptions} />
            </Form.Item>
            <Form.Item name="dba_user_id" label="DBA" rules={[{ required: true, message: "请选择 DBA" }]}>
              <Select showSearch optionFilterProp="label" placeholder="请输入负责此 DB 的 dba" options={userOptions} />
            </Form.Item>
            <Form.Item name="reason" label="备注" rules={[{ required: true, whitespace: true, message: "请填写备注" }]}>
              <Input placeholder="描述信息" />
            </Form.Item>
            <Form.Item
              name="grant_hosts"
              label="授权 IP"
              rules={[{ required: true, whitespace: true, message: "请填写授权 IP" }]}
              extra="多个主机以 | 分隔，例如：192.168.8.12|172.16.%"
            >
              <Input.TextArea
                rows={3}
                placeholder="允许访问数据库应用的 IP 地址，多个主机以 | 分隔符分隔，例如：192.168.8.12|172.16.%"
              />
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
