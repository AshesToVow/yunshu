import { CheckOutlined, CloseOutlined, PlayCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import { getData, http, extractApiErrorMessage } from "../services/http";

interface ApprovalItem {
  id: number;
  user_id: number;
  tool_name: string;
  args_json?: string;
  cluster_id?: number;
  namespace?: string;
  resource?: string;
  reason?: string;
  status: string;
  review_note?: string;
  result_msg?: string;
  created_at?: string;
}

export function AiApprovalsPage() {
  const [list, setList] = useState<ApprovalItem[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState<string>("pending");
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);

  async function load() {
    setLoading(true);
    try {
      const res = await getData<{ list: ApprovalItem[]; total: number }>(
        http.get("/ai/approvals", { params: { status: status || undefined, page, page_size: 10 } }),
      );
      setList(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载审批失败"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [status, page]);

  async function review(id: number, approve: boolean, execute?: boolean) {
    try {
      await getData(http.post(`/ai/approvals/${id}/review`, { approve, execute: !!execute, note: approve ? "同意" : "驳回" }));
      message.success(approve ? "已批准" : "已驳回");
      void load();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "审批失败"));
    }
  }

  async function execute(id: number) {
    try {
      await getData(http.post(`/ai/approvals/${id}/execute`, {}));
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
        columns={[
          { title: "ID", dataIndex: "id", width: 70 },
          { title: "工具", dataIndex: "tool_name", width: 160 },
          { title: "集群", dataIndex: "cluster_id", width: 80 },
          { title: "命名空间", dataIndex: "namespace", width: 120 },
          { title: "资源", dataIndex: "resource", ellipsis: true },
          { title: "状态", dataIndex: "status", width: 100, render: statusTag },
          { title: "结果", dataIndex: "result_msg", ellipsis: true },
          {
            title: "操作",
            width: 260,
            render: (_: unknown, row: ApprovalItem) => (
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
            ),
          },
        ]}
      />
    </Card>
  );
}

export default AiApprovalsPage;
