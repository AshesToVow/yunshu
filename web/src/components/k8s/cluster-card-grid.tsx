import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  SettingOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { Button, Card, Col, Popconfirm, Row, Space, Tag, Tooltip, Typography } from "antd";
import type { ClusterItem } from "../../services/clusters";
import { formatDateTime } from "../../utils/format";

export type ClusterCardGridProps = {
  list: ClusterItem[];
  loading?: boolean;
  statusByID: Record<number, { server_version: string; connection_state?: string; last_error?: string }>;
  projectNameById: (id?: number | null) => string;
  statusUpdatingID: number | null;
  onConnectTest: (record: ClusterItem) => void;
  onToggleStatus: (record: ClusterItem) => void;
  onEdit: (record: ClusterItem) => void;
  onDelete: (record: ClusterItem) => void;
  onAuth: (record: ClusterItem) => void;
  onOpenPods?: (record: ClusterItem) => void;
};

function renderConnection(
  record: ClusterItem,
  statusByID: ClusterCardGridProps["statusByID"],
) {
  if (record.status !== 1) return <Tag>disabled</Tag>;
  const st = statusByID[record.id];
  const state = (st?.connection_state || "unknown").toLowerCase();
  const color =
    state === "ready" ? "success" : state === "connecting" ? "processing" : state === "degraded" ? "error" : "default";
  const label = st?.connection_state || "unknown";
  const err = (st?.last_error || "").trim();
  if (!err) return <Tag color={color}>{label}</Tag>;
  return (
    <Tooltip title={err}>
      <Tag color={color}>{label}</Tag>
    </Tooltip>
  );
}

export function ClusterCardGrid({
  list,
  loading,
  statusByID,
  projectNameById,
  statusUpdatingID,
  onConnectTest,
  onToggleStatus,
  onEdit,
  onDelete,
  onAuth,
  onOpenPods,
}: ClusterCardGridProps) {
  return (
    <Row gutter={[12, 12]}>
      {list.map((record) => {
        const version = record.status === 1 ? statusByID[record.id]?.server_version || "—" : "—";
        return (
          <Col xs={24} sm={12} xl={8} key={record.id}>
            <Card
              className="cluster-card"
              loading={loading}
              size="small"
              title={
                <Space direction="vertical" size={0}>
                  <Typography.Text strong>{record.name}</Typography.Text>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    ID {record.id}
                  </Typography.Text>
                </Space>
              }
              extra={record.status === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>}
            >
              <Space direction="vertical" size={8} style={{ width: "100%" }}>
                <div className="cluster-card__meta">
                  <span>项目</span>
                  <Typography.Text ellipsis>{projectNameById(record.owning_project_id) || "—"}</Typography.Text>
                </div>
                <div className="cluster-card__meta">
                  <span>K8s 版本</span>
                  <Typography.Text>{version}</Typography.Text>
                </div>
                <div className="cluster-card__meta">
                  <span>连接</span>
                  {renderConnection(record, statusByID)}
                </div>
                <div className="cluster-card__meta">
                  <span>创建</span>
                  <Typography.Text type="secondary">{formatDateTime(record.created_at)}</Typography.Text>
                </div>
                <Space wrap size={[4, 4]} className="cluster-card__actions">
                  {onOpenPods ? (
                    <Button size="small" type="link" onClick={() => onOpenPods(record)}>
                      进入 Pod
                    </Button>
                  ) : null}
                  <Button size="small" type="link" icon={<TeamOutlined />} onClick={() => onAuth(record)}>
                    授权
                  </Button>
                  <Button size="small" type="link" icon={<SettingOutlined />} onClick={() => onConnectTest(record)}>
                    测试
                  </Button>
                  <Button size="small" type="link" icon={<EditOutlined />} onClick={() => onEdit(record)}>
                    编辑
                  </Button>
                  <Popconfirm
                    title={record.status === 1 ? "确认停用该集群？" : "确认启用该集群？"}
                    onConfirm={() => onToggleStatus(record)}
                  >
                    <Button
                      size="small"
                      type="link"
                      danger={record.status === 1}
                      loading={statusUpdatingID === record.id}
                      icon={record.status === 1 ? <CloseCircleOutlined /> : <CheckCircleOutlined />}
                    >
                      {record.status === 1 ? "停用" : "启用"}
                    </Button>
                  </Popconfirm>
                  <Popconfirm title="确认删除该集群？" onConfirm={() => onDelete(record)}>
                    <Button size="small" type="link" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              </Space>
            </Card>
          </Col>
        );
      })}
    </Row>
  );
}
