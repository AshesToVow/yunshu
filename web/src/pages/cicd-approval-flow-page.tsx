import { ArrowDownOutlined, ArrowUpOutlined, DeleteOutlined, PlusOutlined, ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Input, Popconfirm, Select, Space, Switch, Table, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { getApprovalFlow, saveApprovalFlow, type CicdApprovalFlowStage } from "../services/cicd";
import { getProjects, type ProjectItem } from "../services/projects";
import { listUserGroups, type UserGroupItem } from "../services/user-groups";

type StageRow = CicdApprovalFlowStage & {
  draftEnabled: boolean;
  draftName: string;
  draftUserGroupId?: number;
  clientKey: string;
};

function newStageKey() {
  return `custom_${Math.random().toString(36).slice(2, 10)}`;
}

function toRows(stages: CicdApprovalFlowStage[]): StageRow[] {
  return (stages ?? []).map((st, index) => ({
    ...st,
    sort_order: st.sort_order || index + 1,
    draftEnabled: st.enabled,
    draftName: st.stage_name,
    draftUserGroupId: st.user_group_id,
    clientKey: st.stage_key || `row_${index}`,
  }));
}

export function CicdApprovalFlowPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [rows, setRows] = useState<StageRow[]>([]);
  const [groups, setGroups] = useState<UserGroupItem[]>([]);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) setProjectId(res.list[0].id);
    });
  }, []);

  const loadGroups = useCallback(async (pid: number) => {
    const res = await listUserGroups({ page: 1, page_size: 500, scope_project_id: pid });
    setGroups(res.list ?? []);
  }, []);

  const loadFlow = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      await loadGroups(projectId);
      const flow = await getApprovalFlow(projectId);
      setRows(toRows(flow.stages ?? []));
    } finally {
      setLoading(false);
    }
  }, [projectId, loadGroups]);

  useEffect(() => {
    void loadFlow();
  }, [loadFlow]);

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
      {
        title: "Key",
        dataIndex: "stage_key",
        width: 140,
        ellipsis: true,
      },
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
        render: (_, row) => (
          <Select
            allowClear
            showSearch
            optionFilterProp="label"
            style={{ width: 260 }}
            placeholder={row.draftEnabled ? "请选择用户组" : "未启用"}
            disabled={!row.draftEnabled}
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

  async function handleSave() {
    if (!projectId) return;
    for (const row of rows) {
      if (!row.draftName.trim()) {
        message.warning("审批节点名称不能为空");
        return;
      }
      if (row.draftEnabled && !row.draftUserGroupId) {
        message.warning(`请为「${row.draftName}」选择用户组`);
        return;
      }
    }
    setSaving(true);
    try {
      const saved = await saveApprovalFlow(
        projectId,
        rows.map((r, index) => ({
          stage_key: r.stage_key,
          stage_name: r.draftName.trim(),
          sort_order: index + 1,
          enabled: r.draftEnabled,
          user_group_id: r.draftEnabled ? r.draftUserGroupId : undefined,
        })),
      );
      setRows(toRows(saved.stages ?? []));
      message.success("审批流已保存");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page-stack">
      <PageTelemetryHeader
        label="[ CD ]"
        title="审批管理"
        subtitle="按项目配置 CD 多级审批节点与审批用户组"
      />
      <Card bordered={false}>
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="审批流程说明"
          description={
            <Typography.Paragraph style={{ marginBottom: 0 }}>
              可自定义增删、重排审批节点；发布开启审批后，工单按<strong>启用节点</strong>顺序串行审批。全部通过后进入「待执行」，
              <strong>仅提交人</strong>可触发 Jenkins 发布。配置变更<strong>仅影响新建工单</strong>，在途工单步骤不变。可在「用户组」菜单中创建审批用户组。
            </Typography.Paragraph>
          }
        />
        <Space wrap style={{ marginBottom: 16 }}>
          <Select
            style={{ width: 240 }}
            placeholder="选择项目"
            value={projectId}
            options={projects.map((p) => ({ label: p.name, value: p.id }))}
            onChange={(v) => setProjectId(v)}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void loadFlow()}>
            刷新
          </Button>
          <Button icon={<PlusOutlined />} onClick={addRow}>
            新增节点
          </Button>
          <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => void handleSave()}>
            保存配置
          </Button>
        </Space>
        <Table rowKey="clientKey" loading={loading} columns={columns} dataSource={rows} pagination={false} />
      </Card>
    </div>
  );
}
