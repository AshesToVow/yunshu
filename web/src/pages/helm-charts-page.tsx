import { CloudDownloadOutlined, ReloadOutlined, RocketOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Input, message, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import { K8sSummaryRow } from "../components/ops/k8s-summary-row";
import {
  getHarborInfo,
  listHarborChartVersions,
  listHarborCharts,
  type HarborChartSummary,
  type HarborChartVersion,
  type HarborConfigInfo,
} from "../services/helm";

const SEARCH_DEBOUNCE_MS = 400;

export function HelmChartsPage() {
  const [info, setInfo] = useState<HarborConfigInfo | null>(null);
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [charts, setCharts] = useState<HarborChartSummary[]>([]);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [expandedVersions, setExpandedVersions] = useState<Record<string, HarborChartVersion[]>>({});
  const [versionsLoading, setVersionsLoading] = useState<Record<string, boolean>>({});

  const loadInfo = useCallback(async () => {
    try {
      setInfo(await getHarborInfo());
    } catch {
      setInfo(null);
    }
  }, []);

  const fetchCharts = useCallback(async (kw: string) => {
    setLoading(true);
    try {
      setCharts(await listHarborCharts(kw || undefined));
      setFetchError(null);
    } catch (err) {
      setCharts([]);
      const msg = err instanceof Error ? err.message : "拉取 Harbor Chart 列表失败";
      setFetchError(msg);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadInfo();
  }, [loadInfo]);

  useEffect(() => {
    const delay = keyword === "" ? 0 : SEARCH_DEBOUNCE_MS;
    const id = window.setTimeout(() => void fetchCharts(keyword.trim()), delay);
    return () => window.clearTimeout(id);
  }, [keyword, fetchCharts]);

  const summary = useMemo(() => {
    const deprecated = charts.filter((c) => c.deprecated).length;
    const versions = charts.reduce((sum, c) => sum + (c.total_versions ?? 0), 0);
    return { total: charts.length, deprecated, versions };
  }, [charts]);

  const loadVersions = async (chartName: string) => {
    if (expandedVersions[chartName]) return;
    setVersionsLoading((p) => ({ ...p, [chartName]: true }));
    try {
      const vers = await listHarborChartVersions(chartName);
      setExpandedVersions((p) => ({ ...p, [chartName]: vers ?? [] }));
    } finally {
      setVersionsLoading((p) => ({ ...p, [chartName]: false }));
    }
  };

  const columns: ColumnsType<HarborChartSummary> = [
    { title: "Chart 名称", dataIndex: "name", width: 200 },
    { title: "最新版本", dataIndex: "latest_version", width: 120 },
    { title: "版本数", dataIndex: "total_versions", width: 90 },
    {
      title: "状态",
      key: "deprecated",
      width: 100,
      render: (_, r) => (r.deprecated ? <Tag color="warning">已废弃</Tag> : <Tag color="success">可用</Tag>),
    },
    {
      title: "OCI 引用",
      key: "oci",
      ellipsis: true,
      render: (_, r) => (
        <Typography.Text copyable={{ text: `${info?.oci_prefix ?? ""}/${r.name}` }} style={{ fontSize: 12 }}>
          {info?.oci_prefix}/{r.name}
        </Typography.Text>
      ),
    },
  ];

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Harbor Helm Chart"
        description="Jenkins 推送的 Chart 包列表与版本"
        extra={
          <Link to="/helm/releases">
            <Button type="primary" icon={<RocketOutlined />}>
              安装 Release
            </Button>
          </Link>
        }
        meta={
          <Typography.Text type="secondary">
            {info ? `${info.project} @ ${info.url}` : "加载 Harbor 配置…"}
          </Typography.Text>
        }
      />
      <Card className="table-card yaml-crud-card" bordered={false}>
        {info && (
          <Alert
            type={info.auth_configured ? "success" : "warning"}
            showIcon
            style={{ marginBottom: 12 }}
            message={
              <Space wrap size={[12, 4]}>
                <span>Harbor: {info.url}</span>
                <span>项目: {info.project}</span>
                <span>OCI: {info.oci_prefix}</span>
                {!info.auth_configured && <span>请在数据字典配置 cicd_harbor_username / cicd_harbor_password</span>}
              </Space>
            }
          />
        )}
        {fetchError ? <Alert type="error" showIcon style={{ marginBottom: 12 }} message={fetchError} /> : null}

        <div className="k8s-page-toolbar" style={{ marginBottom: 12 }}>
          <Space wrap className="k8s-page-toolbar__left">
            <Input.Search
              allowClear
              placeholder="搜索 Chart"
              className="k8s-page-toolbar__search"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              onSearch={(v) => {
                setKeyword(v);
                void fetchCharts(v.trim());
              }}
            />
          </Space>
          <Space wrap className="k8s-page-toolbar__right">
            <Button icon={<ReloadOutlined />} onClick={() => void fetchCharts(keyword.trim())}>
              刷新
            </Button>
          </Space>
        </div>

        <K8sSummaryRow
          items={[
            { label: "Chart 数", value: summary.total },
            { label: "版本总数", value: summary.versions },
            { label: "已废弃", value: summary.deprecated, accent: summary.deprecated > 0 ? "#fbbf24" : undefined },
          ]}
        />

        <Typography.Paragraph type="secondary" style={{ margin: "12px 0" }}>
          列表来自 Harbor Chart Museum API。安装请前往「Helm Release」页。
        </Typography.Paragraph>
        <Table
          rowKey="name"
          loading={loading}
          columns={columns}
          dataSource={charts}
          expandable={{
            expandedRowRender: (record) => {
              const vers = expandedVersions[record.name];
              if (!vers) {
                return (
                  <Button
                    size="small"
                    loading={versionsLoading[record.name]}
                    icon={<CloudDownloadOutlined />}
                    onClick={() => void loadVersions(record.name)}
                  >
                    加载版本列表
                  </Button>
                );
              }
              return (
                <Table
                  size="small"
                  rowKey="version"
                  pagination={false}
                  dataSource={vers}
                  columns={[
                    { title: "版本", dataIndex: "version", width: 120 },
                    { title: "App 版本", dataIndex: "app_version", width: 120 },
                    { title: "创建时间", dataIndex: "created", width: 180 },
                    {
                      title: "状态",
                      width: 90,
                      render: (_, v) => (v.deprecated ? <Tag color="warning">废弃</Tag> : <Tag>正常</Tag>),
                    },
                  ]}
                />
              );
            },
          }}
          pagination={{ pageSize: 20, showSizeChanger: true }}
        />
      </Card>
    </div>
  );
}
