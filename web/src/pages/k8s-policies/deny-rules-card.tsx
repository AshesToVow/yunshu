import { DeleteOutlined } from "@ant-design/icons";
import { Button, Card, Empty, Form, Popconfirm, Select, Space, Table, Tag, Typography } from "antd";
import type { FormInstance } from "antd";
import type { K8sNamespaceDenyRule } from "../../services/k8s-namespace-deny";
import type { UserGroupItem } from "../../services/user-groups";
import type { RoleItem, UserItem } from "../../types/api";
import type { SubjectKind } from "./scoped-subject";

export type DenyFormValues = {
  cluster_id?: number;
  namespace?: string;
};

export type DenyRulesCardProps = {
  denyLoading: boolean;
  activeSubjectReady: boolean;
  denyForm: FormInstance<DenyFormValues>;
  denySubmitting: boolean;
  subjectKind: SubjectKind;
  selectedRole: RoleItem | null;
  selectedGroup: UserGroupItem | null;
  selectedUser: UserItem | null;
  clusterOptions: { id: number; name: string }[];
  watchedDenyClusterId: number | undefined;
  denyNsLoading: boolean;
  denyNsOptions: { label: string; value: string }[];
  denyRules: K8sNamespaceDenyRule[];
  clusterNameById: Map<number, string>;
  onDenyFinish: (values: DenyFormValues) => void | Promise<void>;
  onDeleteDenyRule: (rule: K8sNamespaceDenyRule) => void | Promise<void>;
};

export function DenyRulesCard({
  denyLoading,
  activeSubjectReady,
  denyForm,
  denySubmitting,
  subjectKind,
  selectedRole,
  selectedGroup,
  selectedUser,
  clusterOptions,
  watchedDenyClusterId,
  denyNsLoading,
  denyNsOptions,
  denyRules,
  clusterNameById,
  onDenyFinish,
  onDeleteDenyRule,
}: DenyRulesCardProps) {
  return (
    <Card
      className="table-card"
      style={{ marginTop: 16 }}
      title="命名空间黑名单（对齐 k8m：黑名单优先于白名单与档位）"
      loading={denyLoading}
    >
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        若某<strong>主体</strong>（角色 / 用户 / 组）在指定集群下配置了禁止的命名空间，则即使用户拥有该集群档位，也会在请求进入集群前被拒绝。对已纳入 K8s 范围校验的接口，含
        super-admin 也会被拦截。白名单规则见接口 <Typography.Text code>/api/v1/k8s-namespace-allow-rules</Typography.Text>。
      </Typography.Paragraph>
      {activeSubjectReady ? (
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <Form form={denyForm} layout="inline" onFinish={onDenyFinish}>
            <Typography.Text>主体：</Typography.Text>
            <Tag>
              {subjectKind === "role"
                ? selectedRole?.code
                : subjectKind === "user"
                  ? selectedUser?.username
                  : selectedGroup?.code}
            </Tag>
            <Form.Item name="cluster_id" rules={[{ required: true, message: "请选择集群" }]}>
              <Select
                style={{ minWidth: 220 }}
                placeholder="集群"
                allowClear
                options={clusterOptions.map((c) => ({ label: c.name, value: c.id }))}
              />
            </Form.Item>
            <Form.Item name="namespace" rules={[{ required: true, message: "请选择命名空间" }]}>
              <Select
                showSearch
                optionFilterProp="label"
                allowClear
                loading={denyNsLoading}
                disabled={!watchedDenyClusterId}
                style={{ minWidth: 220 }}
                placeholder={watchedDenyClusterId ? "选择命名空间" : "请先选择集群"}
                options={denyNsOptions}
              />
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" loading={denySubmitting}>
                添加禁止规则
              </Button>
            </Form.Item>
          </Form>
          <Table<K8sNamespaceDenyRule>
            rowKey="id"
            size="small"
            dataSource={denyRules}
            pagination={{ pageSize: 8 }}
            columns={[
              { title: "ID", dataIndex: "id", width: 70 },
              {
                title: "主体",
                key: "p",
                width: 160,
                render: (_: unknown, r: K8sNamespaceDenyRule) => (
                  <span>
                    <Tag>{r.principal_kind}</Tag> <Typography.Text code>{r.principal_ref}</Typography.Text>
                  </span>
                ),
              },
              {
                title: "集群",
                dataIndex: "cluster_id",
                width: 140,
                render: (v: number) => (v === 0 ? <Tag color="blue">全部</Tag> : clusterNameById.get(v) ?? `#${v}`),
              },
              { title: "命名空间", dataIndex: "namespace" },
              {
                title: "操作",
                key: "op",
                width: 110,
                className: "yunshu-table-actions-cell",
                render: (_, r) => (
                  <Popconfirm
                    title="确定删除该黑名单规则？"
                    onConfirm={() => void onDeleteDenyRule(r)}
                  >
                    <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                ),
              },
            ]}
          />
        </Space>
      ) : (
        <Empty description="请先在上方选择角色模板、用户或用户组" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      )}
    </Card>
  );
}
