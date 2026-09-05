import { CheckOutlined, CloseOutlined, PlayCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  executeAIApproval,
  listAIApprovals,
  reviewAIApproval,
  type AIApprovalItem,
} from "../services/ai";
import { getClusters, type ClusterItem } from "../services/clusters";
import { extractApiErrorMessage } from "../services/http";

export function AiApprovalsPage() {
  const [searchParams] = useSearchParams();
  const highlightId = Number(searchParams.get("highlight") || searchParams.get("ticket") || 0);
  const [list, setList] = useState<AIApprovalItem[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState<string>(() => (highlightId > 0 ? "" : "pending"));
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [clusters, setClusters] = useState<ClusterItem[]>([]);

  const clusterNameById = useMemo(() => {
    const m = new Map<number, string>();
    for (const c of clusters) m.set(c.id, c.name || `集群 #${c.id}`);
    return m;
  }, [clusters]);

  async function load() {
    setLoading(true);
    try {
      const res = await listAIApprovals({ status: status || undefined, page, page_size: 10 });
      setList(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载审批失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void getClusters({ page: 1, page_size: 1000 })
      .then((res) => setClusters(res?.list || []))
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 仅随筛选/分页刷新
  }, [status, page]);

  useEffect(() => {
    if (highlightId <= 0) return;
    const el = document.getElementById(`ai-approval-row-${highlightId}`);
    if (el) {
      el.scrollIntoView({ behavior: "smooth", block: "center" });
    }
  }, [highlightId, list]);

  async function review(id: number, approve: boolean, execute?: boolean) {
    try {
      await reviewAIApproval(id, { approve, execute: !!execute, note: approve ? "同意" : "驳回" });
      message.success(approve ? "已批准" : "已驳回");
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "审批失败"));
    }
  }

  async function execute(id: number) {
    try {
      await executeAIApproval(id);
      message.success("已执行");
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "执行失败"));
    }
  }

  const statusTag = (s: string) => {
    const map: Record<string, string> = {
      pending: "processing",
      approved: "success",
      rejected: "default",
      executed: "success",
      failed: "error",
    };
    return <Tag color={map[s] || "default"}>{s}</Tag>;
  };

  return (
    <Card className="table-card">
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message={
          <Typography.Text>
            日常审批请到{" "}
            <Link to="/workflow/inbox?domain=ai">审批中心 → 我的待办</Link>
            ；本页保留为 AI 高危操作的执行台（批准后执行 / 重试）。
          </Typography.Text>
        }
      />
      <div className="toolbar">
        <Space>
          <Select
            style={{ width: 160 }}
            allowClear
            placeholder="状态"
            value={status || undefined}
            onChange={(v) => {
              setPage(1);
              setStatus(v || "");
            }}
            options={[
              { value: "pending", label: "pending" },
              { value: "approved", label: "approved" },
              { value: "rejected", label: "rejected" },
              { value: "executed", label: "executed" },
              { value: "failed", label: "failed" },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()}>
            刷新
          </Button>
        </Space>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{ current: page, pageSize: 10, total, onChange: setPage }}
        onRow={(row) => ({
          id: `ai-approval-row-${row.id}`,
          style:
            highlightId > 0 && row.id === highlightId
              ? { background: "rgba(22, 119, 255, 0.08)" }
              : undefined,
        })}
        columns={[
          { title: "ID", dataIndex: "id", width: 70 },
          { title: "工具", dataIndex: "tool_name", width: 160 },
          {
            title: "集群",
            dataIndex: "cluster_id",
            width: 140,
            ellipsis: true,
            render: (id?: number) => (id ? clusterNameById.get(id) || `#${id}` : "—"),
          },
          { title: "命名空间", dataIndex: "namespace", width: 120 },
          { title: "资源", dataIndex: "resource", ellipsis: true },
          { title: "状态", dataIndex: "status", width: 100, render: statusTag },
          { title: "结果", dataIndex: "result_msg", ellipsis: true },
          {
            title: "操作",
            width: 260,
            render: (_: unknown, row?: AIApprovalItem) =>
              row ? (
                <Space>
                  {row.status === "pending" ? (
                    <>
                      <Button type="link" size="small" icon={<CheckOutlined />} onClick={() => void review(row.id, true, true)}>
                        批准并执行
                      </Button>
                      <Button type="link" size="small" onClick={() => void review(row.id, true, false)}>
                        仅批准
                      </Button>
                      <Button type="link" size="small" danger icon={<CloseOutlined />} onClick={() => void review(row.id, false)}>
                        驳回
                      </Button>
                    </>
                  ) : null}
                  {row.status === "approved" || row.status === "failed" ? (
                    <Button type="link" size="small" icon={<PlayCircleOutlined />} onClick={() => void execute(row.id)}>
                      执行
                    </Button>
                  ) : null}
                </Space>
              ) : null,
          },
        ]}
      />
    </Card>
  );
}

export default AiApprovalsPage;
