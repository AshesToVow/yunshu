import { DeleteOutlined, EyeOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Input, Modal, Select, Space, Steps, Table, Tag, Tooltip, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { PageTelemetryHeader } from "../components/page-telemetry-header";
import {
  deleteBuildRun,
  getBuildRun,
  getBuildRunLog,
  listBuildRunArtifactsMeta,
  listBuildRunStages,
  listBuildRuns,
  type CicdArtifactMeta,
  type CicdBuildRun,
  type CicdRunStage,
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

function buildArtifactLabel(row: CicdBuildRun) {
  if (row.package_path) return row.package_path;
  if (row.image_address) return row.image_address;
  return "";
}

function buildArtifactType(row: CicdBuildRun): "minio" | "helm" | "image" | null {
  if (!row.package_path) {
    return row.image_address ? "image" : null;
  }
  if (row.package_path.startsWith("oci://") || row.package_path.includes("/chartrepo/")) return "helm";
  return "minio";
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
  const [stages, setStages] = useState<CicdRunStage[]>([]);
  const [artifactsMeta, setArtifactsMeta] = useState<CicdArtifactMeta[]>([]);
  const [activeStageId, setActiveStageId] = useState<number>();
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
        key: "artifact",
        ellipsis: true,
        width: 260,
        render: (_, row) => {
          const label = buildArtifactLabel(row);
          if (!label) return "—";
          const type = buildArtifactType(row);
          const typeTag =
            type === "helm" ? (
              <Tag color="geekblue" style={{ marginRight: 6 }}>
                Helm
              </Tag>
            ) : type === "image" ? (
              <Tag color="purple" style={{ marginRight: 6 }}>
                镜像
              </Tag>
            ) : null;
          const tip =
            row.package_path && row.image_address
              ? `Chart/MinIO: ${row.package_path}\n镜像: ${row.image_address}`
              : undefined;
          return (
            <Tooltip title={tip}>
              <span>
                {typeTag}
                {label}
              </span>
            </Tooltip>
          );
        },
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
                const [st, arts] = await Promise.all([
                  listBuildRunStages(projectId, row.id),
                  listBuildRunArtifactsMeta(projectId, row.id),
                ]);
                setStages(st || []);
                setArtifactsMeta(arts || []);
                setActiveStageId(st?.[0]?.id);
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

      <Drawer
        title="构建详情"
        width={720}
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false);
          setStages([]);
          setArtifactsMeta([]);
          setActiveStageId(undefined);
        }}
      >
        {detail && (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            {stages.length > 0 && (
              <div>
                <Typography.Text type="secondary">流水线阶段</Typography.Text>
                <Steps
                  size="small"
                  style={{ marginTop: 8 }}
                  current={Math.max(
                    0,
                    stages.findIndex((s) => s.id === activeStageId),
                  )}
                  onChange={(i) => setActiveStageId(stages[i]?.id)}
                  items={stages.map((s) => ({
                    title: s.stage_name || s.stage_type,
                    status:
                      s.status === "success"
                        ? "finish"
                        : s.status === "failure" || s.status === "failed"
                          ? "error"
                          : s.status === "running"
                            ? "process"
                            : "wait",
                    description: s.duration_sec ? `${s.duration_sec}s` : undefined,
                  }))}
                />
                {(() => {
                  const st = stages.find((s) => s.id === activeStageId) || stages[0];
                  if (!st) return null;
                  let quality = "";
                  let dashboard = detail.sonar_dashboard_url || "";
                  if (st.extra_json) {
                    try {
                      const extra = JSON.parse(st.extra_json) as Record<string, unknown>;
                      if (extra.quality_gate) quality = String(extra.quality_gate);
                      if (extra.dashboard_url) dashboard = String(extra.dashboard_url);
                    } catch {
                      /* ignore */
                    }
                  }
                  return (
                    <div style={{ marginTop: 12 }}>
                      {(st.stage_type === "sonar" || quality || dashboard) && (
                        <Space wrap style={{ marginBottom: 8 }}>
                          {quality ? <Tag color="blue">Quality Gate: {quality}</Tag> : null}
                          {dashboard ? (
                            <Typography.Link href={dashboard} target="_blank" rel="noreferrer">
                              Sonar Dashboard
                            </Typography.Link>
                          ) : null}
                        </Space>
                      )}
                      {st.error_message ? (
                        <Typography.Text type="danger">{st.error_message}</Typography.Text>
                      ) : null}
                      <pre
                        style={{
                          maxHeight: 220,
                          overflow: "auto",
                          fontSize: 12,
                          background: "#1e1e1e",
                          color: "#d4d4d4",
                          padding: 8,
                          borderRadius: 6,
                          whiteSpace: "pre-wrap",
                        }}
                      >
                        {st.logs?.trim() || "（该阶段暂无日志）"}
                      </pre>
                    </div>
                  );
                })()}
              </div>
            )}

            <div style={{ display: "grid", gridTemplateColumns: "120px 1fr", gap: 8 }}>
              {[
                ["应用", detail.service_name],
                ["标识符", detail.service_identifier],
                ["分支", detail.branch_name],
                ["publishMode", detail.publish_mode],
                ["环境", detail.tenv],
                ["版本", detail.version],
                ["结果", detail.build_result],
                ["制品路径", detail.package_path || detail.image_address],
                ["镜像", detail.image_address],
                ["Sonar", detail.sonar_project_key],
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
            {detail.image_address ? (
              <Link to="/cicd/image-browser">在仓库中查看镜像 →</Link>
            ) : null}
            {artifactsMeta.length > 0 && (
              <Table
                size="small"
                rowKey="id"
                pagination={false}
                dataSource={artifactsMeta}
                columns={[
                  { title: "类型", dataIndex: "artifact_type", width: 90 },
                  { title: "名称", dataIndex: "name", ellipsis: true },
                  { title: "路径", dataIndex: "storage_path", ellipsis: true },
                  { title: "Digest", dataIndex: "digest", ellipsis: true, width: 160 },
                ]}
              />
            )}
          </Space>
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
