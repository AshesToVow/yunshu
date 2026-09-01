// @ts-nocheck
import {
  DeleteOutlined,
  EyeOutlined,
  FileTextOutlined,
  LinkOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  Button,
  Card,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { CicdReleaseDetailPanel } from "../components/cicd-release-detail-panel";
import {
  cicdReleaseStatusLabel,
  cicdReleaseStatusTagColor,
  cicdReleaseTypeTagColor,
  cicdReleaseTypeLabel,
} from "../components/cicd-release-utils";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import { useDictOptions } from "../hooks/use-dict-options";
import { useAuth } from "../contexts/auth-context";
import { deleteReleaseRun, executeReleaseRun, getReleaseRunLog, listReleaseRuns, type CicdReleaseRun } from "../services/cicd";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

function statusTag(s: string) {
  return <Tag color={cicdReleaseStatusTagColor(s)}>{cicdReleaseStatusLabel(s)}</Tag>;
}

function releaseTypeTag(releaseType: string) {
  const label = cicdReleaseTypeLabel(releaseType);
  return <Tag color={cicdReleaseTypeTagColor(releaseType)}>{label || "—"}</Tag>;
}

export function CicdReleaseRecordsPage() {
  const { user } = useAuth();
  const tenvOpts = useDictOptions("cicd_tenv");
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [status, setStatus] = useState<string>();
  const [tenv, setTenv] = useState<string>();
  const [keyword, setKeyword] = useState("");
  const [executeLoading, setExecuteLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<CicdReleaseRun[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [detailRun, setDetailRun] = useState<CicdReleaseRun | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [logRun, setLogRun] = useState<CicdReleaseRun | null>(null);
  const [logOpen, setLogOpen] = useState(false);
  const [logText, setLogText] = useState("");
  const [logLoading, setLogLoading] = useState(false);
  const logPreRef = useRef<HTMLPreElement>(null);

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
        status,
        keyword,
        tenv,
      });
      setRows(res.list ?? []);
      setTotal(res.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [projectId, page, pageSize, status, keyword, tenv]);

  useEffect(() => {
    void load();
  }, [load]);

  const fetchLog = useCallback(
    async (run: CicdReleaseRun, scroll = true) => {
      if (!projectId) return;
      setLogLoading(true);
      try {
        const r = await getReleaseRunLog(projectId, run.id);
        setLogText(r.log || "（暂无日志）");
        if (scroll && logPreRef.current) {
          requestAnimationFrame(() => {
            logPreRef.current!.scrollTop = logPreRef.current!.scrollHeight;
          });
        }
      } finally {
        setLogLoading(false);
      }
    },
    [projectId],
  );

  useEffect(() => {
    if (!logOpen || !logRun || !projectId) return;
    const running = logRun.status === "running" || logRun.status === "pending";
    void fetchLog(logRun);
    if (!running) return;
    const timer = window.setInterval(() => {
      void fetchLog(logRun, true);
      void listReleaseRuns(projectId, { page, page_size: pageSize, status, keyword, tenv }).then((res) => {
        const updated = res.list?.find((r) => r.id === logRun.id);
        if (updated) setLogRun(updated);
      });
    }, 3000);
    return () => window.clearInterval(timer);
  }, [logOpen, logRun, projectId, fetchLog, page, pageSize, status, keyword, tenv]);

  const openDetail = (row: CicdReleaseRun) => {
    setDetailRun(row);
    setDetailOpen(true);
  };

  const openLog = (row: CicdReleaseRun) => {
    setLogRun(row);
    setLogOpen(true);
  };

  const columns = useMemo<ColumnsType<CicdReleaseRun>>(
    () => [
      { title: "工单名称", dataIndex: "title", ellipsis: true, width: 200 },
      { title: "应用", dataIndex: "service_name", width: 140, ellipsis: true },
      { title: "提交人", dataIndex: "submitter_name", width: 96 },
      {
        title: "类型",
        dataIndex: "release_type",
        width: 120,
        render: (v) => releaseTypeTag(String(v)),
      },
      { title: "环境", dataIndex: "tenv", width: 72 },
      {
        title: "提交时间",
        dataIndex: "started_at",
        width: 168,
        render: (v) => formatDateTime(v),
      },
      {
        title: "完成时间",
        dataIndex: "finished_at",
        width: 168,
        render: (v) => formatDateTime(v),
      },
      {
        title: "工单状态",
        dataIndex: "status",
        width: 100,
        render: (v) => statusTag(String(v)),
      },
      {
        title: "操作",
        key: "actions",
        width: 200,
        fixed: "right",
        render: (_, row) => (
          <Space size={4} wrap={false}>
            <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => openDetail(row)}>
              详情
            </Button>
            <Button type="link" size="small" icon={<FileTextOutlined />} onClick={() => openLog(row)}>
              发布日志
            </Button>
            <Popconfirm title="确认删除该工单记录？" onConfirm={() => void deleteReleaseRun(projectId!, row.id).then(load)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [projectId, load],
  );

  return (
    <div className="page-stack">
      <PageTelemetryHeader label="[ CD ]" title="CD 历史工单" subtitle="常规发布与容器化发布执行记录" meta={[`TOTAL / ${total}`]} />
      <Card bordered={false} className="yunshu-panel">
        <div className="yunshu-filter-bar" style={{ marginBottom: 16 }}>
          <Space wrap size={[12, 12]}>
            <Select
              style={{ width: 220 }}
              placeholder="选择项目"
              value={projectId}
              options={projects.map((p) => ({ label: p.name, value: p.id }))}
              onChange={(v) => {
                setProjectId(v);
                setPage(1);
              }}
            />
            <Select
              allowClear
              style={{ width: 140 }}
              placeholder="工单状态"
              value={status}
            options={[
              { label: "执行成功", value: "success" },
              { label: "执行中", value: "running" },
              { label: "执行失败", value: "failure" },
              { label: "待审核", value: "pending_approval" },
              { label: "待执行", value: "pending_execution" },
              { label: "已驳回", value: "rejected" },
              { label: "已中止", value: "aborted" },
            ]}
              onChange={(v) => {
                setStatus(v);
                setPage(1);
              }}
            />
            <Select
              allowClear
              style={{ width: 120 }}
              placeholder="环境"
              value={tenv}
              options={tenvOpts.map((o) => ({ label: o.label, value: o.value }))}
              onChange={(v) => {
                setTenv(v);
                setPage(1);
              }}
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
        </div>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1200 }}
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
      </Card>

      <Modal
        title={
          detailRun ? (
            <Space wrap>
              <span>工单详情</span>
              {statusTag(detailRun.status)}
              {releaseTypeTag(detailRun.release_type ?? "")}
            </Space>
          ) : (
            "工单详情"
          )
        }
        open={detailOpen}
        onCancel={() => {
          setDetailOpen(false);
          setDetailRun(null);
        }}
        width={920}
        style={{ top: 24 }}
        footer={
          detailRun ? (
            <Space>
              {detailRun.jenkins_build_url ? (
                <Button icon={<LinkOutlined />} href={detailRun.jenkins_build_url} target="_blank" rel="noreferrer">
                  打开 Jenkins
                </Button>
              ) : null}
              {detailRun.status === "pending_execution" &&
              user?.id &&
              detailRun.submitter_user_id &&
              user.id === detailRun.submitter_user_id ? (
                <Button
                  type="primary"
                  loading={executeLoading}
                  onClick={() => {
                    if (!projectId) return;
                    setExecuteLoading(true);
                    void executeReleaseRun(projectId, detailRun.id)
                      .then(() => {
                        message.success("已触发发布执行");
                        setDetailOpen(false);
                        setDetailRun(null);
                        void load();
                      })
                      .finally(() => setExecuteLoading(false));
                  }}
                >
                  执行发布
                </Button>
              ) : null}
              <Button onClick={() => setDetailOpen(false)}>关闭</Button>
            </Space>
          ) : null
        }
        destroyOnClose
      >
        {detailRun && projectId ? (
          <CicdReleaseDetailPanel
            projectId={projectId}
            runId={detailRun.id}
            onExecuted={() => {
              setDetailOpen(false);
              setDetailRun(null);
              void load();
            }}
          />
        ) : null}
      </Modal>

      <Modal
        title={
          <Space>
            <span>发布日志</span>
            {logRun?.title ? <Typography.Text type="secondary">— {logRun.title}</Typography.Text> : null}
            {logRun && (logRun.status === "running" || logRun.status === "pending") && (
              <Tag color="processing">实时刷新中</Tag>
            )}
          </Space>
        }
        open={logOpen}
        onCancel={() => {
          setLogOpen(false);
          setLogRun(null);
        }}
        footer={
          <Space>
            <Button onClick={() => logRun && void fetchLog(logRun, false)} loading={logLoading}>
              刷新
            </Button>
            <Button type="primary" onClick={() => setLogOpen(false)}>
              关闭
            </Button>
          </Space>
        }
        width="92vw"
        style={{ top: 16, maxWidth: 1400 }}
        styles={{ body: { padding: "12px 16px" } }}
        destroyOnClose
      >
        <pre
          ref={logPreRef}
          style={{
            height: "calc(100vh - 220px)",
            minHeight: 480,
            overflow: "auto",
            fontSize: 12,
            lineHeight: 1.5,
            margin: 0,
            padding: 12,
            background: "#1e1e1e",
            color: "#d4d4d4",
            borderRadius: 6,
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
          }}
        >
          {logText}
        </pre>
      </Modal>
    </div>
  );
}
