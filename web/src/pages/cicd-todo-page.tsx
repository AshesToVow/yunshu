import { ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Select, Space, Table, Tabs, Tag, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { CicdReleaseDetailPanel } from "../components/cicd-release-detail-panel";
import { cicdReleaseTypeLabel, cicdReleaseTypeTagColor } from "../components/cicd-release-utils";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import {
  approveReleaseRun,
  batchApproveReleaseRuns,
  batchExecuteReleaseRuns,
  batchRejectReleaseRuns,
  batchTerminateReleaseRuns,
  executeReleaseRun,
  listReleaseRuns,
  rejectReleaseRun,
  terminateReleaseRun,
  type CicdReleaseRun,
} from "../services/cicd";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

function releaseTypeTag(releaseType: string) {
  const label = cicdReleaseTypeLabel(releaseType);
  return <Tag color={cicdReleaseTypeTagColor(releaseType)}>{label || "—"}</Tag>;
}

type TabKey = "pending_approval" | "pending_execution";

export function CicdTodoPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [tab, setTab] = useState<TabKey>("pending_approval");
  const [releaseType, setReleaseType] = useState<string>();
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<CicdReleaseRun[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);

  const [reviewOpen, setReviewOpen] = useState(false);
  const [reviewTarget, setReviewTarget] = useState<CicdReleaseRun | null>(null);
  const [reviewForm] = Form.useForm<{ comment: string }>();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((res) => {
      setProjects(res.list ?? []);
      if (res.list?.length) setProjectId(res.list[0].id);
    });
  }, []);

  const load = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const res = await listReleaseRuns(projectId, {
        page,
        page_size: pageSize,
        status: tab,
        release_type: releaseType,
        keyword,
        mine: true,
      });
      setRows(res.list ?? []);
      setTotal(res.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [projectId, page, pageSize, tab, releaseType, keyword]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    setSelectedRowKeys([]);
    setPage(1);
  }, [tab, projectId]);

  const openReview = (row: CicdReleaseRun) => {
    setReviewTarget(row);
    reviewForm.resetFields();
    setReviewOpen(true);
  };

  const columns = useMemo<ColumnsType<CicdReleaseRun>>(
    () => [
      { title: "申请标题", dataIndex: "title", ellipsis: true },
      { title: "资源组", dataIndex: "project_name", width: 140, ellipsis: true, render: (v) => v || "—" },
      {
        title: "申请类型",
        dataIndex: "release_type",
        width: 120,
        render: (v) => releaseTypeTag(String(v)),
      },
      { title: "申请人", dataIndex: "submitter_name", width: 100 },
      {
        title: "当前环节",
        dataIndex: "current_stage_name",
        width: 140,
        render: (v, row) =>
          tab === "pending_approval" ? v || "—" : row.submitter_name ? "待提交人执行" : "—",
      },
      {
        title: "申请时间",
        dataIndex: "started_at",
        width: 168,
        render: (v) => formatDateTime(v),
      },
      {
        title: "状态",
        width: 90,
        render: () => (
          <Tag color={tab === "pending_approval" ? "orange" : "processing"}>
            {tab === "pending_approval" ? "待审核" : "待执行"}
          </Tag>
        ),
      },
      {
        title: "操作",
        width: 100,
        fixed: "right",
        render: (_, row) =>
          tab === "pending_approval" ? (
            <Button type="link" size="small" onClick={() => openReview(row)}>
              审核
            </Button>
          ) : (
            <Button
              type="link"
              size="small"
              onClick={() => {
                if (!projectId) return;
                void executeReleaseRun(projectId, row.id).then(() => {
                  message.success("已触发发布执行");
                  void load();
                });
              }}
            >
              执行
            </Button>
          ),
      },
    ],
    [tab, projectId, load],
  );

  const selectedIds = selectedRowKeys;

  return (
    <div className="page-stack">
      <PageTelemetryHeader label="[ CD ]" title="待办列表" subtitle="我的审批与待执行任务（执行仅限提交人）" meta={[`TOTAL / ${total}`]} />
      <Card bordered={false}>
        <Tabs
          activeKey={tab}
          onChange={(k) => setTab(k as TabKey)}
          items={[
            { key: "pending_approval", label: "待我审批" },
            { key: "pending_execution", label: "待我执行" },
          ]}
        />
        <Space wrap style={{ marginBottom: 16 }}>
          <Select
            style={{ width: 220 }}
            placeholder="选择项目"
            value={projectId}
            options={projects.map((p) => ({ label: p.name, value: p.id }))}
            onChange={(v) => setProjectId(v)}
          />
          <Select
            allowClear
            style={{ width: 160 }}
            placeholder="工单类型"
            value={releaseType}
            options={[
              { label: "服务上线", value: "frontend_online" },
              { label: "服务回滚", value: "frontend_rollback" },
              { label: "服务初次部署", value: "backend_initial" },
              { label: "服务更新", value: "backend_update" },
              { label: "POD更新", value: "pod_update" },
            ]}
            onChange={setReleaseType}
          />
          <Input.Search
            allowClear
            placeholder="搜索工单名称"
            style={{ width: 220 }}
            onSearch={(v) => {
              setKeyword(v.trim());
              setPage(1);
            }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
        </Space>

        <Table
          rowKey="id"
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys.map(Number)),
          }}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1000 }}
          locale={{
            emptyText:
              tab === "pending_approval"
                ? "暂无待您审批的工单（仅显示当前轮到您所在用户组处理的环节）"
                : "暂无待您执行的工单（仅提交人可执行）",
          }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />

        <Space style={{ marginTop: 12 }}>
          {tab === "pending_approval" ? (
            <>
              <Button
                type="primary"
                disabled={!selectedIds.length || !projectId}
                onClick={() => {
                  if (!projectId) return;
                  void batchApproveReleaseRuns(projectId, selectedIds).then((r) => {
                    message.success(`已批量审批 ${r.count} 条`);
                    setSelectedRowKeys([]);
                    void load();
                  });
                }}
              >
                批量审批
              </Button>
              <Button
                danger
                disabled={!selectedIds.length || !projectId}
                onClick={() => {
                  if (!projectId) return;
                  void batchRejectReleaseRuns(projectId, selectedIds).then((r) => {
                    message.success(`已批量驳回 ${r.count} 条`);
                    setSelectedRowKeys([]);
                    void load();
                  });
                }}
              >
                批量驳回
              </Button>
            </>
          ) : (
            <>
              <Button
                type="primary"
                disabled={!selectedIds.length || !projectId}
                onClick={() => {
                  if (!projectId) return;
                  void batchExecuteReleaseRuns(projectId, selectedIds).then((r) => {
                    message.success(`已批量执行 ${r.count} 条`);
                    setSelectedRowKeys([]);
                    void load();
                  });
                }}
              >
                批量执行
              </Button>
              <Button
                danger
                disabled={!selectedIds.length || !projectId}
                onClick={() => {
                  if (!projectId) return;
                  void batchTerminateReleaseRuns(projectId, selectedIds).then((r) => {
                    message.success(`已批量终止 ${r.count} 条`);
                    setSelectedRowKeys([]);
                    void load();
                  });
                }}
              >
                批量终止
              </Button>
            </>
          )}
        </Space>
      </Card>

      <Modal
        title={`审核 — ${reviewTarget?.title ?? ""}`}
        open={reviewOpen}
        width={920}
        style={{ top: 24 }}
        destroyOnClose
        onCancel={() => setReviewOpen(false)}
        footer={
          tab === "pending_approval" ? (
            <Space>
              <Button onClick={() => setReviewOpen(false)}>取消</Button>
              <Button
                danger
                onClick={async () => {
                  if (!projectId || !reviewTarget) return;
                  const values = await reviewForm.validateFields();
                  await rejectReleaseRun(projectId, reviewTarget.id, values.comment);
                  message.success("已驳回");
                  setReviewOpen(false);
                  void load();
                }}
              >
                驳回
              </Button>
              <Button
                type="primary"
                onClick={async () => {
                  if (!projectId || !reviewTarget) return;
                  const values = await reviewForm.validateFields();
                  await approveReleaseRun(projectId, reviewTarget.id, values.comment);
                  message.success("审批通过");
                  setReviewOpen(false);
                  void load();
                }}
              >
                通过
              </Button>
            </Space>
          ) : (
            <Button type="primary" onClick={() => setReviewOpen(false)}>
              关闭
            </Button>
          )
        }
      >
        {projectId && reviewTarget ? (
          <CicdReleaseDetailPanel
            projectId={projectId}
            runId={reviewTarget.id}
            reviewMode={tab === "pending_approval"}
            reviewForm={reviewForm}
          />
        ) : null}
      </Modal>
    </div>
  );
}
