import {

  CodeOutlined,

  DeleteOutlined,

  EditOutlined,

  ExpandOutlined,

  FolderOpenOutlined,

  MedicineBoxOutlined,

  UndoOutlined,

} from "@ant-design/icons";

import { Button, Descriptions, Empty, Space, Spin, Table, Tabs, Tag, Typography } from "antd";

import type { ReactNode } from "react";

import type { PodDetail, PodEventItem, PodItem } from "../../services/pods";

import { formatDateTime } from "../../utils/format";

import { PhaseTag } from "../ops/phase-tag";



export type PodDetailPanelProps = {

  selected: PodItem | null;

  detail: PodDetail | null;

  events: PodEventItem[];

  loading?: boolean;

  activeTab?: string;

  onTabChange?: (key: string) => void;

  logsPanel?: ReactNode;

  onExec?: () => void;

  onDiagnose?: () => void;

  onFiles?: () => void;

  onRestart?: () => void;

  onDelete?: () => void;

  onEdit?: () => void;

  onExpand?: () => void;

};



export function PodDetailPanel({

  selected,

  detail,

  events,

  loading,

  activeTab = "overview",

  onTabChange,

  logsPanel,

  onExec,

  onDiagnose,

  onFiles,

  onRestart,

  onDelete,

  onEdit,

  onExpand,

}: PodDetailPanelProps) {

  if (!selected) {

    return (

      <div className="pod-detail-panel pod-detail-panel--empty">

        <Empty description="选择左侧 Pod 查看详情与快捷操作" />

      </div>

    );

  }



  const overviewContent = loading ? (

    <div className="pod-detail-panel__loading">

      <Spin tip="加载详情..." />

    </div>

  ) : !detail ? (

    <Typography.Text type="secondary">暂无详情</Typography.Text>

  ) : (

    <div className="pod-detail-panel__body">

      <Descriptions size="small" column={1} bordered>

        <Descriptions.Item label="节点">{detail.node_name || "-"}</Descriptions.Item>

        <Descriptions.Item label="Pod IP">{detail.pod_ip || "-"}</Descriptions.Item>

        <Descriptions.Item label="QoS">{detail.qos_class || "-"}</Descriptions.Item>

        <Descriptions.Item label="重启">{selected.restart_count ?? 0}</Descriptions.Item>

        <Descriptions.Item label="启动时间">{formatDateTime(detail.start_time)}</Descriptions.Item>

        <Descriptions.Item label="镜像">{detail.containers?.[0]?.image || "-"}</Descriptions.Item>

      </Descriptions>



      <Typography.Text strong style={{ display: "block", margin: "12px 0 8px" }}>

        容器

      </Typography.Text>

      <Table

        size="small"

        rowKey="name"

        pagination={false}

        dataSource={detail.containers}

        columns={[

          { title: "名称", dataIndex: "name", ellipsis: true },

          { title: "状态", dataIndex: "state", width: 72, render: (v: string) => <Tag>{v}</Tag> },

          { title: "重启", dataIndex: "restart_count", width: 56 },

        ]}

      />

    </div>

  );



  const eventsContent = (

    <Table

      size="small"

      rowKey={(r) => `${r.reason}-${r.last_timestamp}-${r.message}`}

      pagination={{ pageSize: 8, size: "small" }}

      dataSource={events}

      locale={{ emptyText: "无事件" }}

      columns={[

        {

          title: "类型",

          dataIndex: "type",

          width: 64,

          render: (v: string) => <Tag color={v === "Warning" ? "warning" : "success"}>{v}</Tag>,

        },

        { title: "原因", dataIndex: "reason", width: 88, ellipsis: true },

        { title: "消息", dataIndex: "message", ellipsis: true },

        {

          title: "时间",

          dataIndex: "last_timestamp",

          width: 140,

          render: (v: string) => formatDateTime(v),

        },

      ]}

    />

  );



  return (

    <div className="pod-detail-panel">

      <div className="pod-detail-panel__header">

        <div>

          <Typography.Text type="secondary" className="pod-detail-panel__ns">

            {selected.namespace}

          </Typography.Text>

          <Typography.Title level={5} className="pod-detail-panel__name">

            {selected.name}

          </Typography.Title>

          <PhaseTag phase={selected.phase} />

        </div>

        {onExpand ? (

          <Button type="text" size="small" icon={<ExpandOutlined />} onClick={onExpand} title="全屏详情" />

        ) : null}

      </div>



      <Space wrap size={[8, 8]} className="pod-detail-panel__actions">

        {onExec ? (

          <Button size="small" icon={<CodeOutlined />} onClick={onExec}>

            Exec

          </Button>

        ) : null}

        {onDiagnose ? (

          <Button size="small" icon={<MedicineBoxOutlined />} onClick={onDiagnose}>

            诊断

          </Button>

        ) : null}

        {onFiles ? (

          <Button size="small" icon={<FolderOpenOutlined />} onClick={onFiles}>

            文件

          </Button>

        ) : null}

        {onRestart ? (

          <Button size="small" icon={<UndoOutlined />} onClick={onRestart}>

            重启

          </Button>

        ) : null}

        {onEdit ? (

          <Button size="small" icon={<EditOutlined />} onClick={onEdit}>

            编辑

          </Button>

        ) : null}

        {onDelete ? (

          <Button size="small" danger icon={<DeleteOutlined />} onClick={onDelete}>

            删除

          </Button>

        ) : null}

      </Space>



      <Tabs

        className="pod-detail-panel__tabs"

        activeKey={activeTab}

        onChange={onTabChange}

        size="small"

        items={[

          { key: "overview", label: "概览", children: overviewContent },

          { key: "logs", label: "日志", children: logsPanel ?? <Typography.Text type="secondary">暂无日志面板</Typography.Text> },

          { key: "events", label: "事件", children: eventsContent },

        ]}

      />

    </div>

  );

}

