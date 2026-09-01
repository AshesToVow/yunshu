// @ts-nocheck
import {
  Alert,
  Button,
  Card,
  Collapse,
  Input,
  Popconfirm,
  Radio,
  Segmented,
  Select,
  Space,
  Table,
  Typography,
  message,
} from "antd";
import { DeleteOutlined, PlusOutlined, ReloadOutlined, StarOutlined } from "@ant-design/icons";
import { useCallback, useEffect, useState } from "react";
import { useAlertMonitor } from "../context";
import { tablePagination } from "../../../utils/table-pagination";
import {
  createPromqlSavedQuery,
  deletePromqlSavedQuery,
  listPromqlSavedQueries,
  type PromqlSavedQueryItem,
} from "../../../services/alerts";
import { extractApiErrorMessage } from "../../../services/http";

export function PromqlTab() {
  const ctx = useAlertMonitor();
  const [savedQueries, setSavedQueries] = useState<PromqlSavedQueryItem[]>([]);
  const [savedLoading, setSavedLoading] = useState(false);
  const [saveName, setSaveName] = useState("");

  const loadSaved = useCallback(async () => {
    setSavedLoading(true);
    try {
      setSavedQueries(await listPromqlSavedQueries());
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载收藏失败"));
    } finally {
      setSavedLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadSaved();
  }, [loadSaved]);

  async function handleSaveQuery() {
    const name = saveName.trim();
    const query = ctx.promQuery.trim();
    if (!name) {
      message.warning("请填写收藏名称");
      return;
    }
    if (!query) {
      message.warning("请先填写 PromQL");
      return;
    }
    try {
      await createPromqlSavedQuery({
        name,
        query,
        datasource_id: ctx.promDsId || undefined,
        kind: ctx.promMode,
        project_id: ctx.projectContextId || undefined,
      });
      message.success("已收藏");
      setSaveName("");
      await loadSaved();
    } catch (e) {
      message.error(extractApiErrorMessage(e, "收藏失败"));
    }
  }

  function applySaved(row: PromqlSavedQueryItem) {
    ctx.setPromQuery(row.query);
    if (row.datasource_id) ctx.setPromDsId(row.datasource_id);
    if (row.kind === "range" || row.kind === "instant") ctx.setPromMode(row.kind);
    message.success(`已应用「${row.name}」`);
  }

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Space wrap>
        <Select
          style={{ minWidth: 220 }}
          placeholder="数据源"
          value={ctx.promDsId}
          onChange={(v) => ctx.setPromDsId(v)}
          options={ctx.dsList.map((d) => ({ label: d.name, value: d.id }))}
        />
        <Radio.Group value={ctx.promMode} onChange={(e) => ctx.setPromMode(e.target.value)}>
          <Radio.Button value="instant">即时</Radio.Button>
          <Radio.Button value="range">范围</Radio.Button>
        </Radio.Group>
      </Space>
      <Input.TextArea rows={4} value={ctx.promQuery} onChange={(e) => ctx.setPromQuery(e.target.value)} placeholder="PromQL" />
      {ctx.promMode === "instant" ? (
        <Space wrap>
          <Input
            style={{ maxWidth: 420 }}
            placeholder="评估时间（可选，RFC3339，例如 2026-04-18T13:30:00+08:00）"
            value={ctx.promTime}
            onChange={(e) => ctx.setPromTime(e.target.value)}
          />
          <Button onClick={ctx.fillPromTimeNow}>当前时间</Button>
          <Button onClick={() => ctx.setPromTime("")}>清空</Button>
        </Space>
      ) : (
        <Space wrap>
          <Input
            style={{ width: 280 }}
            placeholder="start RFC3339，如 2026-04-18T12:00:00+08:00"
            value={ctx.promStart}
            onChange={(e) => ctx.setPromStart(e.target.value)}
          />
          <Input
            style={{ width: 280 }}
            placeholder="end RFC3339，如 2026-04-18T13:00:00+08:00"
            value={ctx.promEnd}
            onChange={(e) => ctx.setPromEnd(e.target.value)}
          />
          <Input style={{ width: 100 }} placeholder="step" value={ctx.promStep} onChange={(e) => ctx.setPromStep(e.target.value)} />
          <Button onClick={ctx.fillPromRangeLastHour}>最近1小时</Button>
        </Space>
      )}
      <Typography.Text type="secondary">
        说明：评估时间是“在哪个时刻执行这条 PromQL”。留空默认当前时间；范围查询需填写 ctx.start/ctx.end（RFC3339），step 可填 30s/1m/5m。
      </Typography.Text>
      <Space wrap>
        <Button type="primary" loading={ctx.promLoading} onClick={() => void ctx.runProm()}>
          执行
        </Button>
        <Input
          style={{ width: 200 }}
          placeholder="收藏名称"
          value={saveName}
          onChange={(e) => setSaveName(e.target.value)}
          prefix={<StarOutlined />}
        />
        <Button icon={<PlusOutlined />} onClick={() => void handleSaveQuery()}>
          收藏当前查询
        </Button>
      </Space>
      <Card
        size="small"
        title="我的 PromQL 收藏"
        extra={
          <Button size="small" icon={<ReloadOutlined />} loading={savedLoading} onClick={() => void loadSaved()}>
            刷新
          </Button>
        }
      >
        <Table
          rowKey="id"
          size="small"
          loading={savedLoading}
          dataSource={savedQueries}
          pagination={tablePagination({ pageSize: 5 })}
          columns={[
            { title: "名称", dataIndex: "name", width: 160, ellipsis: true },
            { title: "类型", dataIndex: "kind", width: 80 },
            {
              title: "查询",
              dataIndex: "query",
              ellipsis: true,
              render: (v: string) => <Typography.Text code>{v}</Typography.Text>,
            },
            {
              title: "操作",
              width: 140,
              render: (_: unknown, row: PromqlSavedQueryItem) => (
                <Space size={4}>
                  <Button type="link" size="small" onClick={() => applySaved(row)}>
                    应用
                  </Button>
                  <Popconfirm title="删除该收藏？" onConfirm={() => void deletePromqlSavedQuery(row.id).then(loadSaved)}>
                    <Button type="link" size="small" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Segmented
        value={ctx.promViewMode}
        onChange={(v) => ctx.setPromViewMode(v as "table" | "json")}
        options={[
          { label: "表格结果", value: "table" },
          { label: "JSON 原文", value: "json" },
        ]}
      />
      {ctx.promViewMode === "json" ? (
        <Input.TextArea rows={14} readOnly value={ctx.promResult} placeholder="查询结果 JSON" />
      ) : ctx.promTableView ? (
        <Table
          rowKey="key"
          size="small"
          bordered
          pagination={tablePagination()}
          scroll={{ x: "max-content", y: 420 }}
          columns={ctx.promTableView.columns}
          dataSource={ctx.promTableView.dataSource}
        />
      ) : ctx.promScalarText ? (
        <Typography.Paragraph>{ctx.promScalarText}</Typography.Paragraph>
      ) : (
        <Typography.Paragraph type="secondary">
          执行查询后在此展示与 Prometheus 页面类似的表格；当前返回类型可能为标量或空结果，可切换到「JSON 原文」查看。
        </Typography.Paragraph>
      )}
    </Space>
  );
}
