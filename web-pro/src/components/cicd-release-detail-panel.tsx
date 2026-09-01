// @ts-nocheck
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloseCircleOutlined,
  RightOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Descriptions,
  Divider,
  Form,
  Input,
  Space,
  Steps,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from "antd";
import type { FormInstance } from "antd/es/form";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  abortProgressiveRelease,
  executeReleaseRun,
  getReleaseRunDetail,
  getReleaseRunLog,
  platformRollbackRelease,
  promoteProgressiveRelease,
  verifyReleaseRun,
  type CicdReleaseApprovalStep,
  type CicdReleaseOperationLog,
  type CicdReleaseRunDetail,
  type ReleaseVerifyResult,
} from "../services/cicd";
import { listAlertEvents, type AlertEventItem } from "../services/alerts";
import { useAuth } from "../contexts/auth-context";
import { formatDateTime } from "../utils/format";
import {
  cicdReleaseKindLabel,
  cicdReleaseStatusLabel,
  cicdReleaseStatusTagColor,
  cicdReleaseTypeLabel,
  cicdReleaseTypeTagColor,
  formatReleaseDuration,
  handlerDisplayName,
  parseReleaseParams,
} from "./cicd-release-utils";

type Props = {
  projectId: number;
  runId: number;
  reviewMode?: boolean;
  reviewForm?: FormInstance<{ comment: string }>;
  /** 执行成功后回调（例如关闭弹窗并刷新列表） */
  onExecuted?: () => void;
};

function stepStatus(st: CicdReleaseApprovalStep): "finish" | "error" | "process" | "wait" {
  if (st.status === "approved") return "finish";
  if (st.status === "rejected") return "error";
  if (st.status === "pending") return "process";
  return "wait";
}

function stepDescription(st: CicdReleaseApprovalStep) {
  if (st.status === "approved") {
    const who = st.reviewer_name || "—";
    const comment = st.review_comment?.trim();
    return comment ? `${who}：${comment}` : `${who}：通过`;
  }
  if (st.status === "rejected") {
    return `${st.reviewer_name || "—"}：驳回${st.review_comment ? `，${st.review_comment}` : ""}`;
  }
  if (st.user_group_name) {
    return `待审批（${st.user_group_name}）`;
  }
  return "待审批";
}

export function CicdReleaseDetailPanel({ projectId, runId, reviewMode, reviewForm, onExecuted }: Props) {
  const { user } = useAuth();
  const [detail, setDetail] = useState<CicdReleaseRunDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState("info");
  const [logText, setLogText] = useState("");
  const [logLoading, setLogLoading] = useState(false);
  const [verifyLoading, setVerifyLoading] = useState(false);
  const [verifyAlerts, setVerifyAlerts] = useState<AlertEventItem[]>([]);
  const [verifyResult, setVerifyResult] = useState<ReleaseVerifyResult | null>(null);
  const [rollbackLoading, setRollbackLoading] = useState(false);
  const [progressiveLoading, setProgressiveLoading] = useState(false);
  const [executeLoading, setExecuteLoading] = useState(false);
  const logPreRef = useRef<HTMLPreElement>(null);

  const loadDetail = useCallback(async () => {
    setLoading(true);
    try {
      const d = await getReleaseRunDetail(projectId, runId);
      setDetail(d);
    } finally {
      setLoading(false);
    }
  }, [projectId, runId]);

  const loadLog = useCallback(async () => {
    setLogLoading(true);
    try {
      if (
        detail &&
        (detail.status === "pending_approval" ||
          detail.status === "pending_execution" ||
          detail.jenkins_build_number == null ||
          detail.jenkins_build_number <= 0)
      ) {
        setLogText("（审批未完成或尚未执行发布，暂无 Jenkins 控制台日志）");
        return;
      }
      const r = await getReleaseRunLog(projectId, runId);
      setLogText(r.log?.trim() || "（暂无 Jenkins 发布日志）");
    } catch {
      setLogText("（获取 Jenkins 日志失败，请稍后重试）");
    } finally {
      setLogLoading(false);
    }
  }, [projectId, runId, detail]);

  useEffect(() => {
    void loadDetail();
  }, [loadDetail]);

  useEffect(() => {
    if (tab === "log" && detail) void loadLog();
  }, [tab, loadLog, detail]);

  useEffect(() => {
    if (tab !== "verify" || !detail) return;
    void (async () => {
      setVerifyLoading(true);
      try {
        const result = await verifyReleaseRun(projectId, runId);
        setVerifyResult(result);
        const res = await listAlertEvents({
          page: 1,
          page_size: 30,
          status: "firing",
          projectId,
          severity: "critical,warning",
        });
        const since = detail.started_at ? new Date(detail.started_at).getTime() : 0;
        const list = (res.list || []).filter((a) => {
          const t = new Date(a.createdAt || (a as any).created_at || 0).getTime();
          return !since || t >= since - 60_000;
        });
        setVerifyAlerts(list);
      } catch {
        setVerifyResult(null);
      } finally {
        setVerifyLoading(false);
      }
    })();
  }, [tab, detail, projectId, runId]);

  const run = detail;
  const releaseParams = run ? parseReleaseParams(run.params_json) : {};
  const artifactName = run?.artifact_name || releaseParams.artifact || "—";
  const currentPending = run?.approval_steps?.find((s) => s.status === "pending");
  const canExecute =
    !!run &&
    run.status === "pending_execution" &&
    !!user?.id &&
    !!run.submitter_user_id &&
    user.id === run.submitter_user_id;

  const logColumns = useMemo<ColumnsType<CicdReleaseOperationLog>>(
    () => [
      { title: "操作", dataIndex: "action", width: 100 },
      { title: "处理人", dataIndex: "actor_name", width: 120 },
      { title: "操作时间", dataIndex: "operated_at", width: 168 },
      { title: "操作信息", dataIndex: "message", ellipsis: true },
    ],
    [],
  );

  const handleExecute = async () => {
    setExecuteLoading(true);
    try {
      await executeReleaseRun(projectId, runId);
      message.success("已触发发布执行");
      await loadDetail();
      onExecuted?.();
    } finally {
      setExecuteLoading(false);
    }
  };

  if (loading && !run) {
    return <Typography.Text type="secondary">加载中…</Typography.Text>;
  }
  if (!run) {
    return <Typography.Text type="danger">工单不存在或无权查看</Typography.Text>;
  }

  return (
    <div>
      {canExecute ? (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          message="审批已全部通过，等待你执行发布"
          description="统一待办只处理审批；执行发布需由提交人确认后触发 Jenkins。"
          action={
            <Button type="primary" loading={executeLoading} onClick={() => void handleExecute()}>
              执行发布
            </Button>
          }
        />
      ) : null}
      <Tabs
      activeKey={tab}
      onChange={setTab}
      items={[
        {
          key: "info",
          label: "任务信息",
          children: (
            <div className="cicd-release-detail-panel">
              <Typography.Title level={5} style={{ marginTop: 0 }}>
                基本信息
              </Typography.Title>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="工单 ID">{run.id}</Descriptions.Item>
                <Descriptions.Item label="申请人">{run.submitter_name || "—"}</Descriptions.Item>
                <Descriptions.Item label="工单名称" span={2}>
                  {run.title}
                </Descriptions.Item>
                <Descriptions.Item label="资源组">{run.project_name || "—"}</Descriptions.Item>
                <Descriptions.Item label="工单类型">
                  <Tag color={cicdReleaseTypeTagColor(run.release_type ?? "")}>
                    {cicdReleaseTypeLabel(run.release_type ?? "")}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="提交时间">{formatDateTime(run.started_at)}</Descriptions.Item>
                <Descriptions.Item label="是否审批">
                  {run.audit_enabled ? <Tag color="blue">审批已开启</Tag> : <Tag>无需审批</Tag>}
                </Descriptions.Item>
                <Descriptions.Item label="当前状态">
                  <Tag color={cicdReleaseStatusTagColor(run.status)}>{cicdReleaseStatusLabel(run.status)}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="当前审批" span={2}>
                  {run.status === "pending_approval" && (currentPending || run.current_stage_name) ? (
                    <Tag color="processing" icon={<ClockCircleOutlined />}>
                      {currentPending?.stage_name || run.current_stage_name}审批
                    </Tag>
                  ) : run.status === "pending_execution" ? (
                    <Tag color="cyan">待提交人执行</Tag>
                  ) : (
                    "—"
                  )}
                </Descriptions.Item>
                {run.approval_flow_text ? (
                  <Descriptions.Item label="审批流程" span={2}>
                    <Space wrap size={[4, 4]}>
                      {run.approval_flow_text.split(" → ").map((part, i, arr) => (
                        <span key={part}>
                          <Tag>{part}</Tag>
                          {i < arr.length - 1 ? <RightOutlined style={{ fontSize: 10, color: "#999" }} /> : null}
                        </span>
                      ))}
                    </Space>
                  </Descriptions.Item>
                ) : null}
              </Descriptions>

              <Divider />
              <Typography.Title level={5}>任务信息</Typography.Title>
              <Descriptions bordered size="small" column={2}>
                <Descriptions.Item label="项目">{run.project_name || "—"}</Descriptions.Item>
                <Descriptions.Item label="服务名称">{run.service_name || "—"}</Descriptions.Item>
                <Descriptions.Item label="环境">{run.tenv || "—"}</Descriptions.Item>
                <Descriptions.Item label="服务类型">{cicdReleaseKindLabel(run.release_kind)}</Descriptions.Item>
                {run.dest_hosts?.length ? (
                  <Descriptions.Item label="操作主机" span={2}>
                    <Space wrap size={[4, 4]}>
                      {run.dest_hosts.map((h) => (
                        <Tag key={h} color="orange">
                          {h}
                        </Tag>
                      ))}
                    </Space>
                  </Descriptions.Item>
                ) : null}
                {run.dest_path ? (
                  <Descriptions.Item label="部署路径" span={2}>
                    {run.dest_path}
                  </Descriptions.Item>
                ) : null}
                {run.deploy_config_name ? (
                  <Descriptions.Item label="发布配置">{run.deploy_config_name}</Descriptions.Item>
                ) : null}
                <Descriptions.Item label="制品/版本" span={run.deploy_config_name ? 1 : 2}>
                  {artifactName !== "—" ? (
                    <Typography.Text copyable={{ text: String(artifactName) }}>{artifactName}</Typography.Text>
                  ) : (
                    "—"
                  )}
                </Descriptions.Item>
                {run.release_kind === "container" && run.image_address ? (
                  <Descriptions.Item label="镜像地址" span={2}>
                    <Typography.Text copyable={{ text: run.image_address }}>{run.image_address}</Typography.Text>
                  </Descriptions.Item>
                ) : null}
              </Descriptions>

              {run.approval_steps?.length ? (
                <>
                  <Divider />
                  <Typography.Title level={5}>审批进度</Typography.Title>
                  <Steps
                    direction="vertical"
                    size="small"
                    current={run.approval_steps.findIndex((s) => s.status === "pending")}
                    items={run.approval_steps.map((st) => ({
                      title: st.stage_name,
                      status: stepStatus(st),
                      icon:
                        st.status === "approved" ? (
                          <CheckCircleOutlined />
                        ) : st.status === "rejected" ? (
                          <CloseCircleOutlined />
                        ) : undefined,
                      description: stepDescription(st),
                    }))}
                  />
                </>
              ) : null}

              {run.current_handlers?.length ? (
                <>
                  <Divider />
                  <Typography.Title level={5}>操作信息</Typography.Title>
                  <Descriptions bordered size="small" column={1}>
                    <Descriptions.Item label="当前处理人">
                      <Space wrap>
                        {run.current_handlers.map((h) => (
                          <Tag key={h.user_id} color="blue">
                            {handlerDisplayName(h)}
                          </Tag>
                        ))}
                      </Space>
                    </Descriptions.Item>
                  </Descriptions>
                </>
              ) : null}

              {(run.status === "success" || run.status === "failure" || run.status === "running") && (
                <>
                  <Divider />
                  <Typography.Title level={5}>执行信息</Typography.Title>
                  <Descriptions bordered size="small" column={2}>
                    <Descriptions.Item label="Jenkins 构建号">
                      {run.jenkins_build_number ? `#${run.jenkins_build_number}` : "—"}
                    </Descriptions.Item>
                    <Descriptions.Item label="耗时">
                      {formatReleaseDuration(run.started_at, run.finished_at)}
                    </Descriptions.Item>
                    <Descriptions.Item label="完成时间" span={2}>
                      {formatDateTime(run.finished_at)}
                    </Descriptions.Item>
                  </Descriptions>
                </>
              )}

              {reviewMode && reviewForm ? (
                <>
                  <Divider />
                  <Form form={reviewForm} layout="vertical">
                    <Form.Item name="comment" label="审核备注 / 终止原因">
                      <Input.TextArea rows={3} placeholder="选填" maxLength={512} showCount />
                    </Form.Item>
                  </Form>
                </>
              ) : null}
            </div>
          ),
        },
        {
          key: "log",
          label: "任务日志",
          children: (
            <div>
              <Alert
                type="info"
                showIcon
                style={{ marginBottom: 12 }}
                message="操作审计"
                description="下方表格记录提交、各级审批与执行状态；Jenkins 控制台输出见底部。"
              />
              <Table
                rowKey={(r, i) => `${r.operated_at}-${i}`}
                size="small"
                pagination={false}
                columns={logColumns}
                dataSource={run.operation_logs ?? []}
                style={{ marginBottom: 16 }}
              />
              <Typography.Text strong>Jenkins 发布日志</Typography.Text>
              <pre
                ref={logPreRef}
                style={{
                  marginTop: 8,
                  maxHeight: 360,
                  overflow: "auto",
                  fontSize: 12,
                  lineHeight: 1.5,
                  padding: 12,
                  background: "#1e1e1e",
                  color: "#d4d4d4",
                  borderRadius: 6,
                  whiteSpace: "pre-wrap",
                  wordBreak: "break-word",
                }}
              >
                {logLoading ? "加载中…" : logText}
              </pre>
            </div>
          ),
        },
        {
          key: "verify",
          label: "发布后验证",
          children: (
            <div>
              <Alert
                type="info"
                showIcon
                style={{ marginBottom: 12 }}
                message="后端验证：Ready / 错误日志 / 新告警"
                description="结果写回 release.verify_status，并记录 change_event。"
              />
              {run.release_kind === "container" ? (
                <Space style={{ marginBottom: 12 }} wrap>
                  <Button
                    danger
                    loading={rollbackLoading}
                    onClick={async () => {
                      setRollbackLoading(true);
                      try {
                        const out = await platformRollbackRelease(projectId, runId);
                        message.success((out?.message as string) || "平台回滚已提交");
                        const result = await verifyReleaseRun(projectId, runId);
                        setVerifyResult(result);
                        await loadDetail();
                      } finally {
                        setRollbackLoading(false);
                      }
                    }}
                  >
                    平台回滚（Deployment/STS）
                  </Button>
                  {releaseParams.deployStrategy === "canary" || releaseParams.deployStrategy === "blue_green" ? (
                    <>
                      <Button
                        type="primary"
                        loading={progressiveLoading}
                        onClick={async () => {
                          setProgressiveLoading(true);
                          try {
                            const out = await promoteProgressiveRelease(projectId, runId);
                            message.success(
                              (out?.state as { last_action?: string } | undefined)?.last_action
                                ? `晋级完成：${(out.state as { last_action: string }).last_action}`
                                : "渐进式晋级已执行",
                            );
                            await loadDetail();
                          } finally {
                            setProgressiveLoading(false);
                          }
                        }}
                      >
                        {releaseParams.deployStrategy === "canary" ? "金丝雀晋级" : "蓝绿切换到 Green"}
                      </Button>
                      {releaseParams.deployStrategy === "canary" ? (
                        <Button
                          loading={progressiveLoading}
                          onClick={async () => {
                            setProgressiveLoading(true);
                            try {
                              await promoteProgressiveRelease(projectId, runId, { final: true });
                              message.success("金丝雀已最终晋级到稳定版");
                              await loadDetail();
                            } finally {
                              setProgressiveLoading(false);
                            }
                          }}
                        >
                          最终晋级（100%）
                        </Button>
                      ) : null}
                      <Button
                        danger
                        loading={progressiveLoading}
                        onClick={async () => {
                          setProgressiveLoading(true);
                          try {
                            await abortProgressiveRelease(projectId, runId);
                            message.success("已中止渐进式发布");
                            await loadDetail();
                          } finally {
                            setProgressiveLoading(false);
                          }
                        }}
                      >
                        中止渐进式发布
                      </Button>
                    </>
                  ) : null}
                  <Typography.Text type="secondary">
                    策略：{releaseParams.deployStrategy || "rolling"}；优先平台操作，再 Jenkins 回滚
                  </Typography.Text>
                </Space>
              ) : null}
              {run.progressive_json ? (
                <Alert
                  type="info"
                  showIcon
                  style={{ marginBottom: 12 }}
                  message="渐进式发布状态"
                  description={
                    <Typography.Paragraph
                      copyable
                      style={{ marginBottom: 0, whiteSpace: "pre-wrap", fontFamily: "monospace", fontSize: 12 }}
                    >
                      {run.progressive_json}
                    </Typography.Paragraph>
                  }
                />
              ) : null}
              {verifyLoading ? (
                <Typography.Text type="secondary">验证中…</Typography.Text>
              ) : verifyResult ? (
                <Space direction="vertical" style={{ width: "100%", marginBottom: 12 }}>
                  <Tag color={verifyResult.status === "passed" ? "green" : verifyResult.status === "failed" ? "red" : "orange"}>
                    {verifyResult.status}
                  </Tag>
                  <Descriptions size="small" column={1} bordered>
                    <Descriptions.Item label="Ready">{verifyResult.ready_detail || "-"}</Descriptions.Item>
                    <Descriptions.Item label="日志错误">{verifyResult.log_detail || `${verifyResult.log_errors}`}</Descriptions.Item>
                    <Descriptions.Item label="新告警">{verifyResult.alert_detail || `${verifyResult.new_alerts}`}</Descriptions.Item>
                    <Descriptions.Item label="检查时间">{verifyResult.checked_at}</Descriptions.Item>
                  </Descriptions>
                </Space>
              ) : (
                <Alert type="warning" showIcon message="验证接口调用失败，以下为前端告警抽样" style={{ marginBottom: 12 }} />
              )}
              {verifyAlerts.length === 0 ? (
                <Alert type="success" showIcon message="未发现同期高优告警列表项" />
              ) : (
                <Table
                  rowKey="id"
                  size="small"
                  pagination={false}
                  dataSource={verifyAlerts}
                  columns={[
                    { title: "级别", dataIndex: "severity", width: 90 },
                    { title: "标题", dataIndex: "title" },
                    {
                      title: "时间",
                      dataIndex: "createdAt",
                      width: 170,
                      render: (v: string, row) => formatDateTime(v || (row as any).created_at),
                    },
                  ]}
                />
              )}
            </div>
          ),
        },
      ]}
    />
    </div>
  );
}
