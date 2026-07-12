import { Checkbox, Divider, Form, Radio, Select, Space, Typography } from "antd";
import type { FormInstance } from "antd";

export const DB_PRIVILEGE_OPTIONS = [
  { value: "select", label: "SELECT（查询）" },
  { value: "insert", label: "INSERT" },
  { value: "update", label: "UPDATE" },
  { value: "delete", label: "DELETE" },
  { value: "create", label: "CREATE（建表等）" },
  { value: "create_database", label: "CREATE DATABASE（新建库）" },
  { value: "alter", label: "ALTER" },
  { value: "drop", label: "DROP" },
  { value: "truncate", label: "TRUNCATE" },
  { value: "index", label: "INDEX" },
  { value: "export", label: "导出" },
  { value: "import", label: "导入" },
];

export const DB_PRIVILEGE_GROUPS = [
  { title: "查询", options: DB_PRIVILEGE_OPTIONS.filter((o) => o.value === "select") },
  { title: "数据变更", options: DB_PRIVILEGE_OPTIONS.filter((o) => ["insert", "update", "delete"].includes(o.value)) },
  { title: "结构变更", options: DB_PRIVILEGE_OPTIONS.filter((o) => ["create", "alter", "drop", "truncate", "index"].includes(o.value)) },
  { title: "导入导出", options: DB_PRIVILEGE_OPTIONS.filter((o) => ["export", "import"].includes(o.value)) },
];

export function DbPrivilegeFields() {
  return (
    <>
      <Form.Item name="scope_type" label="授权范围" initialValue="database">
        <Radio.Group>
          <Radio value="database">库级（该库下全部表）</Radio>
          <Radio value="table">表级（指定表）</Radio>
        </Radio.Group>
      </Form.Item>
      <Form.Item
        noStyle
        shouldUpdate={(prev, cur) => prev.scope_type !== cur.scope_type || prev.database_name !== cur.database_name}
      >
        {({ getFieldValue }) =>
          getFieldValue("scope_type") === "table" ? (
            <Form.Item name="table_names" label="目标表" rules={[{ required: true, message: "请选择至少一个表" }]}>
              <Select mode="multiple" placeholder="选择表" options={[]} />
            </Form.Item>
          ) : null
        }
      </Form.Item>
      <Form.Item name="privileges" label="权限项" rules={[{ required: true, message: "请至少选择一项权限" }]}>
        <Checkbox.Group style={{ width: "100%" }}>
          <Space direction="vertical" style={{ width: "100%" }}>
            {DB_PRIVILEGE_GROUPS.map((group) => (
              <div key={group.title}>
                <Typography.Text type="secondary">{group.title}</Typography.Text>
                <div style={{ marginTop: 8 }}>
                  <Space wrap>
                    {group.options.map((opt) => (
                      <Checkbox key={opt.value} value={opt.value}>
                        {opt.label}
                      </Checkbox>
                    ))}
                  </Space>
                </div>
                <Divider style={{ margin: "12px 0" }} />
              </div>
            ))}
          </Space>
        </Checkbox.Group>
      </Form.Item>
    </>
  );
}

export function bindTableOptions(form: FormInstance, tables: { name: string }[]) {
  const scope = form.getFieldValue("scope_type");
  if (scope !== "table") return;
  // Ant Design Form doesn't support dynamic options on nested Select easily; caller renders table select.
  void tables;
}

export function privilegeSummary(privileges?: string[]) {
  if (!privileges?.length) return "—";
  const labels: Record<string, string> = {
    select: "SELECT",
    insert: "INSERT",
    update: "UPDATE",
    delete: "DELETE",
    create: "CREATE",
    alter: "ALTER",
    drop: "DROP",
    truncate: "TRUNCATE",
    index: "INDEX",
    export: "导出",
    import: "导入",
  };
  return privileges.map((p) => labels[p] ?? p.toUpperCase()).join("、");
}
