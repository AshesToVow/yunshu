import { Checkbox, Col, Row, Typography } from "antd";
import type { CheckboxChangeEvent } from "antd/es/checkbox";
import type { DbInstance } from "../../services/dbmgmt";

export const APPLY_TYPE_OPTIONS = [
  { value: "add_ip", label: "应用用户权限-已存在用户新增IP授权" },
  { value: "new_user", label: "应用用户权限-新用户创建" },
  { value: "add_priv", label: "应用用户权限-已存在用户新增权限" },
  { value: "revoke", label: "应用用户权限-权限回收" },
] as const;

export type AppUserApplyType = (typeof APPLY_TYPE_OPTIONS)[number]["value"];

export const PRIV_LEVEL_OPTIONS = [
  { value: "global", label: "全局权限" },
  { value: "database", label: "库级权限" },
];

export const MYSQL_PRIV_GROUPS = {
  data: ["SELECT", "INSERT", "UPDATE", "DELETE"],
  structure: [
    "CREATE",
    "ALTER",
    "INDEX",
    "DROP",
    "CREATE TEMPORARY TABLES",
    "SHOW VIEW",
    "CREATE ROUTINE",
    "ALTER ROUTINE",
    "EXECUTE",
    "CREATE VIEW",
    "EVENT",
    "TRIGGER",
  ],
  management: [
    "GRANT",
    "SUPER",
    "PROCESS",
    "RELOAD",
    "SHUTDOWN",
    "SHOW DATABASES",
    "LOCK TABLES",
    "REFERENCES",
    "REPLICATION CLIENT",
    "REPLICATION SLAVE",
    "CREATE USER",
  ],
} as const;

export function formatInstanceLabel(inst: DbInstance) {
  const driver = (inst.driver || "mysql").toUpperCase();
  return `${driver}-${inst.host}-${inst.port}`;
}

export function parseMysqlUserKey(key: string) {
  const idx = key.lastIndexOf("@");
  if (idx <= 0) return { user: key, host: "%" };
  return { user: key.slice(0, idx), host: key.slice(idx + 1) || "%" };
}

export function DbmgmtSectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ display: "flex", alignItems: "center", marginBottom: 16, marginTop: 8 }}>
      <span style={{ width: 4, height: 16, background: "#1890ff", marginRight: 8, borderRadius: 2 }} />
      <Typography.Title level={5} style={{ margin: 0 }}>
        {children}
      </Typography.Title>
    </div>
  );
}

export function PrivilegeCheckboxGroups({
  value = [],
  onChange,
  disabled,
  disabledPrivs = [],
  lockedPrivs = [],
  allowedPrivs,
}: {
  value?: string[];
  onChange?: (v: string[]) => void;
  disabled?: boolean;
  /** 不可操作的权限（置灰） */
  disabledPrivs?: string[];
  /** 已持有权限：置灰且显示勾选，不计入表单 value */
  lockedPrivs?: string[];
  /** 仅展示这些权限（权限回收场景） */
  allowedPrivs?: string[];
}) {
  const allowedSet = allowedPrivs?.length ? new Set(allowedPrivs.map((v) => v.toUpperCase())) : null;
  const filterPrivs = (privs: readonly string[]) =>
    allowedSet ? privs.filter((p) => allowedSet.has(p.toUpperCase())) : [...privs];
  const set = new Set(value.map((v) => v.toUpperCase()));
  const disabledSet = new Set(disabledPrivs.map((v) => v.toUpperCase()));
  const lockedSet = new Set(lockedPrivs.map((v) => v.toUpperCase()));
  const isChecked = (priv: string) => set.has(priv) || lockedSet.has(priv);
  const isItemDisabled = (priv: string) => Boolean(disabled) || disabledSet.has(priv);

  const toggle = (priv: string, checked: boolean) => {
    if (isItemDisabled(priv)) return;
    const next = new Set(set);
    if (checked) next.add(priv);
    else next.delete(priv);
    onChange?.([...next]);
  };
  const toggleGroup = (privs: readonly string[], e: CheckboxChangeEvent) => {
    const next = new Set(set);
    for (const p of privs) {
      if (isItemDisabled(p)) continue;
      if (e.target.checked) next.add(p);
      else next.delete(p);
    }
    onChange?.([...next]);
  };

  const renderGroup = (title: string, privs: readonly string[]) => {
    const visible = filterPrivs(privs);
    if (visible.length === 0) return null;
    const selectable = visible.filter((p) => !isItemDisabled(p));
    const checkedSelectable = selectable.filter((p) => isChecked(p));
    return (
      <Col xs={24} md={8}>
        <div style={{ fontWeight: 600, marginBottom: 8 }}>{title}</div>
        <Checkbox
          disabled={disabled || selectable.length === 0}
          indeterminate={checkedSelectable.length > 0 && checkedSelectable.length < selectable.length}
          checked={selectable.length > 0 && checkedSelectable.length === selectable.length}
          onChange={(e) => toggleGroup(visible, e)}
          style={{ marginBottom: 8 }}
        >
          全选
        </Checkbox>
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          {visible.map((p) => (
            <Checkbox
              key={p}
              disabled={isItemDisabled(p)}
              checked={isChecked(p)}
              onChange={(e) => toggle(p, e.target.checked)}
              style={lockedSet.has(p) ? { color: "rgba(0,0,0,0.35)" } : undefined}
            >
              {p}
            </Checkbox>
          ))}
        </div>
      </Col>
    );
  };

  return (
    <Row gutter={16}>
      {renderGroup("数据", MYSQL_PRIV_GROUPS.data)}
      {renderGroup("结构", MYSQL_PRIV_GROUPS.structure)}
      {renderGroup("管理", MYSQL_PRIV_GROUPS.management)}
    </Row>
  );
}

export const CREATE_TABLE_SAMPLE = `CREATE TABLE \`demo_table\` (
  \`id\` int(11) NOT NULL AUTO_INCREMENT COMMENT '主键',
  \`name\` varchar(64) NOT NULL DEFAULT '' COMMENT '名称',
  PRIMARY KEY (\`id\`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='示例表';`;

export const SQL_AUDIT_RULES = [
  "表必须有主键，类型为 int，并设置 AUTO_INCREMENT",
  "字段、表必须有 COMMENT",
  "存储引擎须为 InnoDB（未指定时默认 InnoDB）",
  "AUTO_INCREMENT 须从 1 开始或不指定",
  "禁止高危 DROP/TRUNCATE 未审批直接执行",
];

export function ticketTypeLabel(t?: string) {
  if (t === "sql_import") return "SQL文件上线";
  if (t === "sql_execute") return "SQL上线申请";
  return t || "sql工单";
}

export function auditModeLabel(mode?: string) {
  return mode === "manual" ? "人工审核" : "系统审核";
}
