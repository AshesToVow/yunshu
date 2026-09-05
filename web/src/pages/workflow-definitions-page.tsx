import { ArrowDownOutlined, ArrowUpOutlined, DeleteOutlined, PlusOutlined, ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Input, Popconfirm, Select, Space, Switch, Table, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { getProjects, type ProjectItem } from "../services/projects";
import { listUserGroups, type UserGroupItem } from "../services/user-groups";
import {
  getWorkflowDefinition,
  saveWorkflowDefinition,
  type WorkflowStageItem,
} from "../services/workflow";

type DomainKey = "dbmgmt" | "cicd" | "incident" | "ai";

type StageRow = WorkflowStageItem & {
  draftEnabled: boolean;
  draftName: string;
  draftUserGroupId?: number;
  clientKey: string;
};

const DOMAIN_OPTIONS: { label: string; value: DomainKey; hint: string }[] = [
  { label: "数据库 (dbmgmt)", value: "dbmgmt", hint: "SQL 工单、权限申请、应用用户共用此流程（ticket_type=default）。" },
  { label: "发布 (cicd)", value: "cicd", hint: "发布审批流；配置变更仅影响新建发布工单。" },
  { label: "AI (ai)", value: "ai", hint: "AI 高危工具审批（全局 project_id=0，ticket_type=tool_approval，平台角色审批）。" },
  { label: "故障 (incident)", value: "incident", hint: "告警转故障单审批；可含值班派单节点。" },
];

function newStageKey() {
  return `custom_${Math.random().toString(36).slice(2, 10)}`;
}

function toRows(stages: WorkflowStageItem[]): StageRow[] {
  return (stages ?? []).map((st, index) => ({
    ...st,
    sort_order: st.sort_order || index + 1,
    draftEnabled: st.enabled,
    draftName: st.stage_name,
    draftUserGroupId: st.user_group_id,
    clientKey: st.stage_key || `row_${index}`,
  }));
}

export function WorkflowDefinitionsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [domain, setDomain] = useState<DomainKey>(() => {
    const d = searchParams.get("domain");
    if (d === "cicd" || d === "incident" || d === "dbmgmt" || d === "ai") return d;
    return "dbmgmt";
  });
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [rows, setRows] = useState<StageRow[]>([]);
  const [groups, setGroups] = useState<UserGroupItem[]>([]);
  const [configured, setConfigured] = useState(false);

  const defProjectId = domain === "ai" ? 0 : projectId;
  const defTicketType = domain === "ai" ? "tool_approval" : "default";

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      const fromQuery = Number(searchParams.get("project") || 0);
      if (fromQuery > 0) setProjectId(fromQuery);
      else if (res.list?.length) setProjectId(res.list[0].id);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 仅首屏读 query
  }, []);

  useEffect(() => {
    const next = new URLSearchParams();
    next.set("domain", domain);
    if (projectId) next.set("project", String(projectId));
    setSearchParams(next, { replace: true });
  }, [domain, projectId, setSearchParams]);

  const domainHint = DOMAIN_OPTIONS.find((d) => d.value === domain)?.hint ?? "";

  const loadGroups = useCallback(async (pid: number) => {
    const res = await listUserGroups({ page: 1, page_size: 500, scope_project_id: pid });
    setGroups(res.list ?? []);
  }, []);

  const load = useCallback(async () => {
    if (domain !== "ai" && !projectId) return;
    setLoading(true);
    try {
      if (projectId) await loadGroups(projectId);
      const res = await getWorkflowDefinition(domain, defProjectId ?? 0, defTicketType);
      setConfigured(Boolean(res.configured));
      setRows(toRows(res.stages ?? []));
    } finally {
      setLoading(false);
    }
  }, [domain, projectId, defProjectId, defTicketType, loadGroups]);

  useEffect(() => {
    void load();
  }, [load]);

  const groupOptions = useMemo(
    () => groups.map((g) => ({ label: `${g.name}（${g.member_count}人）`, value: g.id })),
    [groups],
  );

  function renumber(list: StageRow[]) {
    return list.map((row, index) => ({ ...row, sort_order: index + 1 }));
  }

  function moveRow(clientKey: string, delta: number) {
    setRows((prev) => {
      const idx = prev.findIndex((r) => r.clientKey === clientKey);
      const nextIdx = idx + delta;
      if (idx < 0 || nextIdx < 0 || nextIdx >= prev.length) return prev;
      const next = [...prev];
      const [item] = next.splice(idx, 1);
      next.splice(nextIdx, 0, item);
      return renumber(next);
    });
  }

  function addRow() {
    const key = newStageKey();
    setRows((prev) =>
      renumber([
        ...prev,
        {
          stage_key: key,
          stage_name: "新审批节点",
          sort_order: prev.length + 1,
          enabled: false,
          assignee_rule_type: "user_group",
          draftEnabled: false,
          draftName: "新审批节点",
          clientKey: key,
        },
      ]),
    );
  }

  const columns = useMemo<ColumnsType<StageRow>>(
    () => [
      { title: "顺序", dataIndex: "sort_order", width: 64 },
      {
        title: "审批节点",
        width: 200,
        render: (_, row) => (
          <Input
            value={row.draftName}
            maxLength={64}
            onChange={(e) => {
              const value = e.target.value;
              setRows((prev) => prev.map((r) => (r.clientKey === row.clientKey ? { ...r, draftName: value } : r)));
            }}
          />
        ),
      },
      { title: "Key", dataIndex: "stage_key", width: 140, ellipsis: true },
      {
        title: "启用",
        width: 88,
        render: (_, row) => (
          <Switch
            checked={row.draftEnabled}
            onChange={(checked) => {
              setRows((prev) =>
                prev.map((r) => (r.clientKey === row.clientKey ? { ...r, draftEnabled: checked } : r)),
              );
            }}
          />
        ),
      },
      {
        title: "审批用户组",
        render: (_, row) =>
          row.assignee_rule_type === "platform_role" ? (
            <Typography.Text type="secondary">平台角色（admin / ops-admin / ai-approver）</Typography.Text>
          ) : (
          <Select
            allowClear
            showSearch
            optionFilterProp="label"
            style={{ width: 260 }}
            placeholder={row.draftEnabled ? "请选择用户组" : "未启用"}
            disabled={!row.draftEnabled || row.assignee_rule_type === "duty"}
            value={row.draftUserGroupId}
            options={groupOptions}
            onChange={(v) => {
              setRows((prev) =>
                prev.map((r) => (r.clientKey === row.clientKey ? { ...r, draftUserGroupId: v } : r)),
              );
            }}
          />
          ),
      },
      {
        title: "操作",
        width: 150,
        render: (_, row, index) => (
          <Space size={4}>
            <Button
              type="text"
              size="small"
              icon={<ArrowUpOutlined />}
              disabled={index === 0}
              onClick={() => moveRow(row.clientKey, -1)}
            />
            <Button
              type="text"
              size="small"
              icon={<ArrowDownOutlined />}
              disabled={index === rows.length - 1}
              onClick={() => moveRow(row.clientKey, 1)}
            />
            <Popconfirm
              title="删除该审批节点？"
              description="仅影响新建工单，在途工单步骤不变。"
              onConfirm={() => {
                setRows((prev) => {
                  if (prev.length <= 1) {
                    message.warning("至少保留一个审批节点");
                    return prev;
                  }
                  return renumber(prev.filter((r) => r.clientKey !== row.clientKey));
                });
              }}
            >
              <Button type="text" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [groupOptions, rows.length],
  );

  const save = async () => {
    if (domain !== "ai" && !projectId) return;
    for (const row of rows) {
      if (!row.draftName.trim()) {
        message.warning("审批节点名称不能为空");
        return;
      }
      if (
        row.draftEnabled &&
        row.assignee_rule_type !== "duty" &&
        row.assignee_rule_type !== "platform_role" &&
        !row.draftUserGroupId
      ) {
        message.warning(`请为「${row.draftName}」选择用户组`);
        return;
      }
    }
    setSaving(true);
    try {
      const saved = await saveWorkflowDefinition(
        domain,
        defProjectId ?? 0,
        rows.map((s, index) => ({
          stage_key: s.stage_key,
          stage_name: s.draftName.trim(),
          sort_order: index + 1,
          enabled: s.draftEnabled,
          assignee_rule_type: s.assignee_rule_type || (domain === "ai" ? "platform_role" : "user_group"),
          user_group_id:
            s.draftEnabled && s.assignee_rule_type !== "duty" && s.assignee_rule_type !== "platform_role"
              ? s.draftUserGroupId
              : undefined,
          duty_monitor_rule_id: s.duty_monitor_rule_id,
        })),
        defTicketType,
      );
      setConfigured(Boolean(saved.configured));
      setRows(toRows(saved.stages ?? []));
      message.success("已保存");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <PageTelemetryHeader label="Workflow" title="审批流配置" subtitle="统一纳管各业务域审批节点与审批人用户组" />
      <Card
        extra={
          <Space wrap>
            <Select
              style={{ width: 200 }}
              value={domain}
              options={DOMAIN_OPTIONS.map((d) => ({ value: d.value, label: d.label }))}
              onChange={(v) => setDomain(v)}
            />
            {domain !== "ai" ? (
              <Select
                style={{ width: 200 }}
                value={projectId}
                options={projects.map((p) => ({ value: p.id, label: p.name }))}
                onChange={setProjectId}
              />
            ) : null}
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button icon={<PlusOutlined />} onClick={addRow}>
              新增节点
            </Button>
            <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => void save()}>
              保存
            </Button>
          </Space>
        }
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message={configured ? "已配置（写入 workflow_definitions）" : "尚未保存过，当前为默认骨架"}
          description={
            <Typography.Paragraph style={{ marginBottom: 0 }}>
              {domainHint} 关闭开关即跳过该节点；启用须绑定审批用户组（值班节点除外）。配置变更
              <strong>仅影响新建工单</strong>。待办统一在「工单中心 → 我的待办」处理。
            </Typography.Paragraph>
          }
        />
        <Table rowKey="clientKey" loading={loading} pagination={false} dataSource={rows} columns={columns} />
      </Card>
    </div>
  );
}
