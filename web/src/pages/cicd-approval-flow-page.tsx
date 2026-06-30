import { ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Select, Space, Switch, Table, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { getApprovalFlow, saveApprovalFlow, type CicdApprovalFlowStage } from "../services/cicd";
import { getProjects, type ProjectItem } from "../services/projects";
import { listUserGroups, type UserGroupItem } from "../services/user-groups";

type StageRow = CicdApprovalFlowStage & {
  draftEnabled: boolean;
  draftUserGroupId?: number;
};

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
      setRows(
        (flow.stages ?? []).map((st) => ({
          ...st,
          draftEnabled: st.enabled,
          draftUserGroupId: st.user_group_id,
        })),
      );
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

  const columns = useMemo<ColumnsType<StageRow>>(
    () => [
      { title: "顺序", dataIndex: "sort_order", width: 72 },
      { title: "审批节点", dataIndex: "stage_name", width: 180 },
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

  async function handleSave() {
    if (!projectId) return;
    for (const row of rows) {
      if (row.draftEnabled && !row.draftUserGroupId) {
        message.warning(`请为「${row.stage_name}」选择用户组`);
        return;
      }
    }
    setSaving(true);
    try {
      const saved = await saveApprovalFlow(
        projectId,
        rows.map((r) => ({
          stage_key: r.stage_key,
          enabled: r.draftEnabled,
          user_group_id: r.draftEnabled ? r.draftUserGroupId : undefined,
        })),
      );
      setRows(
        (saved.stages ?? []).map((st) => ({
          ...st,
          draftEnabled: st.enabled,
          draftUserGroupId: st.user_group_id,
        })),
      );
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
              发布开启审批后，工单将按下方启用的节点依次审批（测试负责人 → 研发负责人 → 项目/产品负责人 →
              运维负责人）。全部通过后进入「待执行」，<strong>仅提交人</strong>可点击执行触发 Jenkins 发布。可在
              「用户组」菜单中为各节点创建用户组并添加成员。
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
          <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={() => void handleSave()}>
            保存配置
          </Button>
        </Space>
        <Table rowKey="stage_key" loading={loading} columns={columns} dataSource={rows} pagination={false} />
      </Card>
    </div>
  );
}
