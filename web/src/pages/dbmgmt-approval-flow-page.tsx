import { ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Select, Space, Switch, Table, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { getDbApprovalFlow, saveDbApprovalFlow, type DbApprovalStage } from "../services/dbmgmt";
import { getProjects, type ProjectItem } from "../services/projects";
import { listUserGroups, type UserGroupItem } from "../services/user-groups";

type StageRow = DbApprovalStage & {
  draftEnabled: boolean;
  draftUserGroupId?: number;
};

export function DbmgmtApprovalFlowPage() {
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

  const load = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      await loadGroups(projectId);
      const res = await getDbApprovalFlow(projectId);
      setRows(
        (res.stages ?? []).map((s) => ({
          ...s,
          draftEnabled: s.enabled,
          draftUserGroupId: s.user_group_id,
        })),
      );
    } finally {
      setLoading(false);
    }
  }, [projectId, loadGroups]);

  useEffect(() => {
    void load();
  }, [load]);

  const groupOptions = useMemo(
    () => groups.map((g) => ({ label: `${g.name}（${g.member_count}人）`, value: g.id })),
    [groups],
  );

  const columns = useMemo<ColumnsType<StageRow>>(
    () => [
      { title: "顺序", dataIndex: "sort_order", width: 72 },
      { title: "审批节点", dataIndex: "stage_name", width: 180 },
      { title: "Key", dataIndex: "stage_key", width: 140 },
      {
        title: "启用",
        width: 88,
        render: (_, row) => (
          <Switch
            checked={row.draftEnabled}
            onChange={(checked) => {
              setRows((prev) =>
                prev.map((r) => (r.stage_key === row.stage_key ? { ...r, draftEnabled: checked } : r)),
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
            style={{ width: 280 }}
            placeholder={row.draftEnabled ? "请选择用户组" : "未启用"}
            disabled={!row.draftEnabled}
            value={row.draftUserGroupId}
            options={groupOptions}
            onChange={(v) => {
              setRows((prev) =>
                prev.map((r) => (r.stage_key === row.stage_key ? { ...r, draftUserGroupId: v } : r)),
              );
            }}
          />
        ),
      },
    ],
    [groupOptions],
  );

  const save = async () => {
    if (!projectId) return;
    for (const row of rows) {
      if (row.draftEnabled && !row.draftUserGroupId) {
        message.warning(`请为「${row.stage_name}」选择用户组`);
        return;
      }
    }
    setSaving(true);
    try {
      const saved = await saveDbApprovalFlow(
        projectId,
        rows.map((s) => ({
          stage_key: s.stage_key,
          enabled: s.draftEnabled,
          user_group_id: s.draftEnabled ? s.draftUserGroupId : undefined,
        })),
      );
      setRows(
        (saved.stages ?? []).map((s) => ({
          ...s,
          draftEnabled: s.enabled,
          draftUserGroupId: s.user_group_id,
        })),
      );
      message.success("已保存");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card
      title="审批流配置"
      extra={
        <Space>
          <Select
            style={{ width: 200 }}
            value={projectId}
            options={projects.map((p) => ({ value: p.id, label: p.name }))}
            onChange={setProjectId}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
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
        message="审批流程说明"
        description={
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            数据库权限申请与 SQL 工单采用<strong>固定 3 个审批节点</strong>（DBA 负责人 → 安全负责人 →
            运维负责人），暂不支持自定义增删节点。关闭节点开关即跳过该环节；启用节点须绑定审批用户组，且
            <strong>当前用户须为该组成员</strong>方可审批（超级管理员除外）。可在「用户组」菜单中创建用户组并添加成员。
          </Typography.Paragraph>
        }
      />
      <Table rowKey="stage_key" loading={loading} pagination={false} dataSource={rows} columns={columns} />
    </Card>
  );
}
