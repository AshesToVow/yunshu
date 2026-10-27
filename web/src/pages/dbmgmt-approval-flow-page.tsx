import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  DeleteOutlined,
  PlusOutlined,
  ReloadOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getDbApprovalFlow, saveDbApprovalFlow, type DbApprovalStage } from "../services/dbmgmt";
import { extractApiErrorMessage } from "../services/http";
import { getProjects, type ProjectItem } from "../services/projects";
import { listUserGroups, type UserGroupItem } from "../services/user-groups";

type StageRow = DbApprovalStage & {
  draftEnabled: boolean;
  draftName: string;
  draftUserGroupId?: number;
  clientKey: string;
};

function newStageKey() {
  return `custom_${Math.random().toString(36).slice(2, 10)}`;
}

function toRows(stages: DbApprovalStage[]): StageRow[] {
  return (stages ?? []).map((st, index) => ({
    ...st,
    sort_order: st.sort_order || index + 1,
    draftEnabled: st.enabled,
    draftName: st.stage_name,
    draftUserGroupId: st.user_group_id,
    clientKey: st.stage_key || `row_${index}`,
  }));
}

export function DbmgmtApprovalFlowPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [rows, setRows] = useState<StageRow[]>([]);
  const [groups, setGroups] = useState<UserGroupItem[]>([]);
  const [dirty, setDirty] = useState(false);
  const loadSeqRef = useRef(0);
  const savingRef = useRef(false);

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 })
      .then((res) => {
        setProjects(res.list ?? []);
        if (res.list?.length) setProjectId(res.list[0].id);
      })
      .catch((e) => message.error(extractApiErrorMessage(e, "加载项目列表失败")));
  }, []);

  const load = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeqRef.current;
    setLoading(true);
    try {
      // 用户组与审批流并行获取，整体做序号守卫，避免慢响应覆盖当前项目
      const [groupsRes, flowRes] = await Promise.all([
        listUserGroups({ page: 1, page_size: 500, scope_project_id: projectId }),
        getDbApprovalFlow(projectId),
      ]);
      if (seq !== loadSeqRef.current) return; // 已有更新的请求，丢弃过期响应
      setGroups(groupsRes.list ?? []);
      setRows(toRows(flowRes.stages ?? []));
      setDirty(false);
    } catch (e) {
      if (seq !== loadSeqRef.current) return;
      message.error(extractApiErrorMessage(e, "加载审批流失败"));
    } finally {
      if (seq === loadSeqRef.current) setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    // 切换项目先清空旧项目数据，避免误编辑
    setRows([]);
    setGroups([]);
    setDirty(false);
  }, [projectId]);

  useEffect(() => {
    void load();
  }, [load]);

  const groupOptions = useMemo(
    () => groups.map((g) => ({ label: `${g.name}（${g.member_count}人）`, value: g.id })),
    [groups],
  );

  const editingDisabled = saving || loading;

  function renumber(list: StageRow[]) {
    return list.map((row, index) => ({ ...row, sort_order: index + 1 }));
  }

  const moveRow = useCallback((clientKey: string, delta: number) => {
    setRows((prev) => {
      const idx = prev.findIndex((r) => r.clientKey === clientKey);
      const nextIdx = idx + delta;
      if (idx < 0 || nextIdx < 0 || nextIdx >= prev.length) return prev;
      const next = [...prev];
      const [item] = next.splice(idx, 1);
      next.splice(nextIdx, 0, item);
      return renumber(next);
    });
    setDirty(true);
  }, []);

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
    setDirty(true);
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
            disabled={editingDisabled}
            onChange={(e) => {
              const value = e.target.value;
              setRows((prev) => prev.map((r) => (r.clientKey === row.clientKey ? { ...r, draftName: value } : r)));
              setDirty(true);
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
            disabled={editingDisabled}
            onChange={(checked) => {
              setRows((prev) => prev.map((r) => (r.clientKey === row.clientKey ? { ...r, draftEnabled: checked } : r)));
              setDirty(true);
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
            disabled={!row.draftEnabled || editingDisabled}
            value={row.draftUserGroupId}
            options={groupOptions}
            onChange={(v) => {
              setRows((prev) => prev.map((r) => (r.clientKey === row.clientKey ? { ...r, draftUserGroupId: v } : r)));
              setDirty(true);
            }}
          />
        ),
      },
      {
        title: "操作",
        width: 150,
        fixed: "right",
        render: (_, row, index) => (
          <Space size={4}>
            <Button
              type="text"
              size="small"
              icon={<ArrowUpOutlined />}
              disabled={index === 0 || editingDisabled}
              onClick={() => moveRow(row.clientKey, -1)}
            />
            <Button
              type="text"
              size="small"
              icon={<ArrowDownOutlined />}
              disabled={index === rows.length - 1 || editingDisabled}
              onClick={() => moveRow(row.clientKey, 1)}
            />
            <Popconfirm
              title="删除该审批节点？"
              description="仅影响新建工单，在途工单步骤不变。"
              disabled={editingDisabled}
              onConfirm={() => {
                if (rows.length <= 1) {
                  message.warning("至少保留一个审批节点");
                  return;
                }
                setRows((prev) => renumber(prev.filter((r) => r.clientKey !== row.clientKey)));
                setDirty(true);
              }}
            >
              <Button type="text" size="small" danger icon={<DeleteOutlined />} disabled={editingDisabled} />
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [editingDisabled, groupOptions, moveRow, rows.length],
  );

  const save = async () => {
    if (!projectId || savingRef.current) return;
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
    savingRef.current = true;
    setSaving(true);
    try {
      const saved = await saveDbApprovalFlow(
        projectId,
        rows.map((s, index) => ({
          stage_key: s.stage_key,
          stage_name: s.draftName.trim(),
          sort_order: index + 1,
          enabled: s.draftEnabled,
          user_group_id: s.draftEnabled ? s.draftUserGroupId : undefined,
        })),
      );
      setRows(toRows(saved.stages ?? []));
      setDirty(false);
      message.success("已保存");
    } catch (e) {
      message.error(extractApiErrorMessage(e, "保存审批流失败"));
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  };

  const onProjectChange = (next: number) => {
    if (next === projectId) return;
    if (!dirty) {
      setProjectId(next);
      return;
    }
    Modal.confirm({
      title: "有未保存的修改",
      content: "切换项目将丢失当前未保存的审批流配置，确认切换？",
      okText: "放弃修改并切换",
      okButtonProps: { danger: true },
      cancelText: "取消",
      onOk: () => setProjectId(next),
    });
  };

  return (
    <Card className="table-card" title="审批流配置">
      <div className="toolbar">
        <Space wrap>
          <Select
            style={{ width: 200 }}
            value={projectId}
            options={projects.map((p) => ({ value: p.id, label: p.name }))}
            onChange={onProjectChange}
            disabled={saving || loading}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} disabled={saving || loading}>
            刷新
          </Button>
          <Button icon={<PlusOutlined />} onClick={addRow} disabled={saving || loading}>
            新增节点
          </Button>
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={saving}
            disabled={saving || loading}
            onClick={() => void save()}
          >
            保存
          </Button>
        </Space>
      </div>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="审批流程说明"
        description={
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            数据库权限申请、SQL
            工单与应用用户申请共用此审批流。可自定义增删、重排节点；关闭开关即跳过该环节，启用须绑定审批用户组。
            配置变更<strong>仅影响新建工单</strong>。审批人须为对应用户组成员（超级管理员除外）。
          </Typography.Paragraph>
        }
      />
      <Table
        rowKey="clientKey"
        loading={loading}
        pagination={false}
        dataSource={rows}
        columns={columns}
        scroll={{ x: 900 }}
      />
    </Card>
  );
}
