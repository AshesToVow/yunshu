import { DeleteOutlined, EyeOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Input, Modal, Select, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import {
  deleteBuildRun,
  getBuildRun,
  getBuildRunLog,
  listBuildRuns,
  type CicdBuildRun,
} from "../services/cicd";
import { getProjects, type ProjectItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

function resultTag(r: string) {
  const map: Record<string, string> = {
    success: "success",
    failure: "error",
    running: "processing",
    pending: "default",
  };
  return <Tag color={map[r] || "default"}>{r === "success" ? "构建成功" : r === "failure" ? "构建失败" : r}</Tag>;
}

export function CicdBuildRecordsPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [serviceKeyword, setServiceKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<CicdBuildRun[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<CicdBuildRun | null>(null);
  const [logOpen, setLogOpen] = useState(false);
  const [logText, setLogText] = useState("");
  const [logRun, setLogRun] = useState<CicdBuildRun | null>(null);
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
      const res = await listBuildRuns(projectId, { page, page_size: pageSize, keyword: serviceKeyword });
      setRows(res.list ?? []);
      setTotal(res.total ?? 0);
    } finally {
      setLoading(false);
    }
  }, [projectId, page, pageSize, serviceKeyword]);

  useEffect(() => {
    void load();
  }, [load]);

  const fetchLog = useCallback(
    async (run: CicdBuildRun, scroll = true) => {
      if (!projectId) return;
      setLogLoading(true);
      try {
        const r = await getBuildRunLog(projectId, run.id);
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
    const running = logRun.build_result === "running" || logRun.build_result === "pending";
    void fetchLog(logRun);
    if (!running) return;
    const timer = window.setInterval(() => {
      void fetchLog(logRun, true);
      void listBuildRuns(projectId, { page, page_size: pageSize, keyword: serviceKeyword }).then((res) => {
        const updated = res.list?.find((r) => r.id === logRun.id);
        if (updated) setLogRun(updated);
      });
    }, 3000);
    return () => window.clearInterval(timer);
  }, [logOpen, logRun, projectId, fetchLog, page, pageSize, serviceKeyword]);

  const columns = useMemo<ColumnsType<CicdBuildRun>>(
    () => [
      { title: "应用标识符", dataIndex: "service_identifier", width: 140 },
      { title: "应用名称", dataIndex: "service_name", width: 160 },
      { title: "构建人", dataIndex: "builder_name", width: 100 },
      { title: "构建编号", dataIndex: "build_number", width: 90 },
      { title: "打包结果", dataIndex: "build_result", width: 100, render: (v) => resultTag(String(v)) },
      { title: "分支", dataIndex: "branch_name", width: 120 },
      {
        title: "制品路径",
        dataIndex: "package_path",
        ellipsis: true,
        width: 220,
        render: (v) => v || "—",
      },
      { title: "开始时间", dataIndex: "started_at", width: 170, render: (v) => formatDateTime(v) },
      {
        title: "操作",
        key: "actions",
        width: 200,
        fixed: "right",
        render: (_, row) => (
          <Space>
            <Button
              type="link"
              size="small"
              onClick={() => {
                setLogRun(row);
                setLogOpen(true);
              }}
            >
              打包日志
            </Button>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={async () => {
                if (!projectId) return;
                const r = await getBuildRun(projectId, row.id);
                setDetail(r);
                setDetailOpen(true);
              }}
            />
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => void deleteBuildRun(projectId!, row.id).then(load)}
            />
          </Space>
        ),
      },
    ],
    [projectId, load],
  );

  return (
    <div className="page-stack">
      <PageTelemetryHeader label="[ CI ]" title="CI 打包记录" subtitle="Jenkins 构建历史与 Console 日志" meta={[`TOTAL / ${total}`]} />
      <Card bordered={false}>
        <Space wrap style={{ marginBottom: 16 }}>
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
          <Input.Search placeholder="搜索应用/构建人" style={{ width: 240 }} onSearch={(v) => { setServiceKeyword(v); setPage(1); }} />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
        </Space>
        <Table
          rowKey="id"
          loading={loading}
          columns={columns}
          dataSource={rows}
          scroll={{ x: 1200 }}
          pagination={{ current: page, pageSize, total, onChange: (p, ps) => { setPage(p); setPageSize(ps); } }}
        />
      </Card>

      <Drawer title="构建详情" width={640} open={detailOpen} onClose={() => setDetailOpen(false)}>
        {detail && (
          <div style={{ display: "grid", gridTemplateColumns: "120px 1fr", gap: 8 }}>
            {[
              ["应用", detail.service_name],
              ["标识符", detail.service_identifier],
              ["分支", detail.branch_name],
              ["publishMode", detail.publish_mode],
              ["环境", detail.tenv],
              ["版本", detail.version],
              ["结果", detail.build_result],
              ["制品路径", detail.package_path],
              ["镜像", detail.image_address],
              ["Jenkins", detail.jenkins_build_url],
              ["开始", formatDateTime(detail.started_at)],
              ["结束", formatDateTime(detail.finished_at)],
            ].map(([k, v]) => (
              <span key={String(k)}>
                <Typography.Text type="secondary">{k}</Typography.Text>
                <div style={{ wordBreak: "break-all" }}>{v || "—"}</div>
              </span>
            ))}
          </div>
        )}
      </Drawer>

      <Modal
        title={
          <Space>
            <span>打包日志</span>
            {logRun && (logRun.build_result === "running" || logRun.build_result === "pending") && (
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
