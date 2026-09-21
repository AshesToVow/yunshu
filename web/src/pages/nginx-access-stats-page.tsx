import { BarChartOutlined, ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Col,
  DatePicker,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs, { type Dayjs } from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  getNginxAccessStats,
  getProjectLogSources,
  getProjectServers,
  getProjectServices,
  getProjects,
  type LogHistogramBucket,
  type NginxAccessStatsResult,
  type NginxAccessURIStat,
  type ProjectItem,
  type ServerItem,
  type ServiceItem,
  type LogSourceItem,
} from "../services/projects";
import { formatDateTime } from "../utils/format";
import { buildProjectLogsUrl } from "../utils/log-navigation";

const { RangePicker } = DatePicker;
const { Text, Link, Paragraph } = Typography;

type RangeKey = "today" | "7d" | "30d" | "custom";

function rangeForPreset(key: RangeKey): [Dayjs, Dayjs] {
  const end = dayjs();
  switch (key) {
    case "today":
      return [end.startOf("day"), end];
    case "7d":
      return [end.subtract(7, "day"), end];
    case "30d":
      return [end.subtract(30, "day"), end];
    default:
      return [end.subtract(1, "day"), end];
  }
}

function formatRate(v?: number) {
  if (v == null || Number.isNaN(v)) return "-";
  return `${v.toFixed(2)}%`;
}

function formatQps(v?: number) {
  if (v == null || Number.isNaN(v)) return "-";
  return v.toFixed(3);
}

function formatMs(v?: number) {
  if (v == null || Number.isNaN(v)) return "-";
  return `${v.toFixed(2)} ms`;
}

export function NginxAccessStatsPage() {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [sources, setSources] = useState<LogSourceItem[]>([]);
  const [serverId, setServerId] = useState<number>();
  const [serviceId, setServiceId] = useState<number>();
  const [logSourceId, setLogSourceId] = useState<number>();
  const [rangeKey, setRangeKey] = useState<RangeKey>("today");
  const [range, setRange] = useState<[Dayjs, Dayjs]>(() => rangeForPreset("today"));
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState<NginxAccessStatsResult | null>(null);

  const projectOptions = useMemo(
    () => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })),
    [projects],
  );
  const serverOptions = useMemo(
    () => servers.map((s) => ({ value: s.id, label: `${s.name || s.host} (${s.host})` })),
    [servers],
  );
  const serviceOptions = useMemo(
    () => services.map((s) => ({ value: s.id, label: s.name })),
    [services],
  );
  const sourceOptions = useMemo(
    () =>
      sources.map((s) => ({
        value: s.id,
        label: `${s.path}${s.multiline_rule ? ` (${s.multiline_rule})` : ""}`,
      })),
    [sources],
  );

  useEffect(() => {
    void (async () => {
      try {
        const res = await getProjects({ page: 1, page_size: 200 });
        setProjects(res.list || []);
        if (!projectId && res.list?.length) {
          setProjectId(res.list[0].id);
        }
      } catch (e: unknown) {
        message.error(String((e as Error)?.message ?? e));
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!projectId) return;
    setServerId(undefined);
    setServiceId(undefined);
    setLogSourceId(undefined);
    void (async () => {
      try {
        const [srv, svc, src] = await Promise.all([
          getProjectServers(projectId, { page: 1, page_size: 200 }),
          getProjectServices(projectId, { page: 1, page_size: 1000 }),
          getProjectLogSources(projectId, { page: 1, page_size: 1000 }),
        ]);
        setServers(srv.list || []);
        setServices(svc.list || []);
        setSources(src.list || []);
      } catch (e: unknown) {
        message.error(String((e as Error)?.message ?? e));
      }
    })();
  }, [projectId]);

  const load = useCallback(async () => {
    if (!projectId) {
      message.warning("请先选择项目");
      return;
    }
    const [from, to] = range;
    if (!from || !to) {
      message.warning("请选择时间范围");
      return;
    }
    setLoading(true);
    try {
      const res = await getNginxAccessStats(projectId, {
        from: from.toISOString(),
        to: to.toISOString(),
        server_id: serverId,
        service_id: serviceId,
        log_source_id: logSourceId,
        top_n: 5,
      });
      setStats(res);
    } catch (e: unknown) {
      setStats(null);
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setLoading(false);
    }
  }, [projectId, range, serverId, serviceId, logSourceId]);

  useEffect(() => {
    if (projectId) {
      void load();
    }
  }, [projectId, load]);

  const onPreset = (key: RangeKey) => {
    setRangeKey(key);
    if (key !== "custom") {
      setRange(rangeForPreset(key));
    }
  };

  const uriColumns: ColumnsType<NginxAccessURIStat> = [
    {
      title: "接口 URI",
      dataIndex: "uri",
      render: (uri: string) => (
        <Link
          onClick={() => {
            if (!projectId) return;
            navigate(
              buildProjectLogsUrl({
                project_id: projectId,
                server_id: serverId,
                service_id: serviceId,
                log_source_id: logSourceId,
                keyword: uri,
                from: range[0]?.toISOString(),
                to: range[1]?.toISOString(),
                tab: "logs",
              }),
            );
          }}
        >
          {uri}
        </Link>
      ),
    },
    { title: "次数", dataIndex: "count", width: 100 },
    {
      title: "占比",
      dataIndex: "percent",
      width: 100,
      render: (p: number) => formatRate(p),
    },
  ];

  return (
    <div className="page-card">
      <Space direction="vertical" size="middle" style={{ width: "100%" }}>
        <Space wrap align="center">
          <BarChartOutlined style={{ fontSize: 18 }} />
          <Typography.Title level={4} style={{ margin: 0 }}>
            Nginx 访问统计
          </Typography.Title>
          <Text type="secondary">按天 / 周 / 月 / 自定义时间段聚合 access 日志</Text>
        </Space>

        <Card size="small">
          <Space wrap>
            <Select
              style={{ minWidth: 220 }}
              placeholder="项目"
              options={projectOptions}
              value={projectId}
              onChange={setProjectId}
              showSearch
              optionFilterProp="label"
            />
            <Select
              allowClear
              style={{ minWidth: 200 }}
              placeholder="服务器"
              options={serverOptions}
              value={serverId}
              onChange={setServerId}
              showSearch
              optionFilterProp="label"
            />
            <Select
              allowClear
              style={{ minWidth: 180 }}
              placeholder="服务"
              options={serviceOptions}
              value={serviceId}
              onChange={setServiceId}
              showSearch
              optionFilterProp="label"
            />
            <Select
              allowClear
              style={{ minWidth: 240 }}
              placeholder="日志源"
              options={sourceOptions}
              value={logSourceId}
              onChange={setLogSourceId}
              showSearch
              optionFilterProp="label"
            />
            <Select
              style={{ width: 120 }}
              value={rangeKey}
              onChange={onPreset}
              options={[
                { value: "today", label: "今天" },
                { value: "7d", label: "近 7 天" },
                { value: "30d", label: "近 30 天" },
                { value: "custom", label: "自定义" },
              ]}
            />
            <RangePicker
              showTime
              value={range}
              onChange={(v) => {
                if (v?.[0] && v?.[1]) {
                  setRangeKey("custom");
                  setRange([v[0], v[1]]);
                }
              }}
            />
            <Button type="primary" icon={<ReloadOutlined />} loading={loading} onClick={() => void load()}>
              查询
            </Button>
            <Button
              icon={<SearchOutlined />}
              disabled={!projectId}
              onClick={() => {
                if (!projectId) return;
                navigate(
                  buildProjectLogsUrl({
                    project_id: projectId,
                    server_id: serverId,
                    service_id: serviceId,
                    log_source_id: logSourceId,
                    from: range[0]?.toISOString(),
                    to: range[1]?.toISOString(),
                  }),
                );
              }}
            >
              打开日志检索
            </Button>
          </Space>
        </Card>

        <Alert
          type="info"
          showIcon
          message="数据要求"
          description={
            <Paragraph style={{ marginBottom: 0 }}>
              日志源请使用 <Text code>nginx_access</Text> 解析模板。延迟指标需要 log_format 末尾包含{" "}
              <Text code>$request_time</Text>
              ，例如：
              <Text code>
                {"$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" $request_time"}
              </Text>
              。修改后需重新下发 Agent。
            </Paragraph>
          }
        />

        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} md={6} lg={4}>
            <Card size="small">
              <Statistic title="访问次数 (PV)" value={stats?.pv ?? 0} loading={loading} />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6} lg={4}>
            <Card size="small">
              <Statistic title="独立 IP (UV)" value={stats?.uv ?? 0} loading={loading} />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6} lg={4}>
            <Card size="small">
              <Statistic title="平均 QPS" value={formatQps(stats?.avg_qps)} loading={loading} />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6} lg={4}>
            <Card size="small">
              <Statistic title="峰值 QPS" value={formatQps(stats?.peak_qps)} loading={loading} />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6} lg={4}>
            <Card size="small">
              <Statistic title="2xx 成功率" value={formatRate(stats?.status_2xx_rate)} loading={loading} />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6} lg={4}>
            <Card size="small">
              <Statistic title="3xx 比例" value={formatRate(stats?.status_3xx_rate)} loading={loading} />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6} lg={4}>
            <Card size="small">
              <Statistic title="4xx 比例" value={formatRate(stats?.status_4xx_rate)} loading={loading} />
            </Card>
          </Col>
          <Col xs={12} sm={8} md={6} lg={4}>
            <Card size="small">
              <Statistic title="5xx 比例" value={formatRate(stats?.status_5xx_rate)} loading={loading} />
            </Card>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card size="small" title="响应时间">
              {stats?.latency_available ? (
                <Row gutter={16}>
                  <Col span={8}>
                    <Statistic title="平均" value={formatMs(stats.avg_latency_ms)} />
                  </Col>
                  <Col span={8}>
                    <Statistic title="P95" value={formatMs(stats.p95_latency_ms)} />
                  </Col>
                  <Col span={8}>
                    <Statistic title="P99" value={formatMs(stats.p99_latency_ms)} />
                  </Col>
                </Row>
              ) : (
                <Alert type="warning" showIcon message={stats?.latency_hint || "暂无延迟数据"} />
              )}
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card size="small" title="访问量趋势">
              <AccessHistogram buckets={stats?.histogram || []} />
            </Card>
          </Col>
        </Row>

        <Card size="small" title="TOP 5 接口">
          <Table
            rowKey="uri"
            size="small"
            loading={loading}
            pagination={false}
            columns={uriColumns}
            dataSource={stats?.top_uris || []}
            locale={{ emptyText: "暂无 URI 排行（确认已用 nginx_access 解析并产生 status/request 字段）" }}
          />
        </Card>
      </Space>
    </div>
  );
}

function AccessHistogram({ buckets }: { buckets: LogHistogramBucket[] }) {
  const sorted = [...buckets].sort((a, b) => a.time.localeCompare(b.time));
  const sampled =
    sorted.length > 60 ? sorted.filter((_, i) => i % Math.ceil(sorted.length / 60) === 0) : sorted;
  const max = Math.max(1, ...sampled.map((b) => b.count));
  if (!sampled.length) {
    return <Text type="secondary">无时间分布数据</Text>;
  }
  const chartW = Math.max(480, sampled.length * 12);
  const chartH = 110;
  const padL = 36;
  const padB = 20;
  const padT = 6;
  const innerH = chartH - padB - padT;
  const barW = Math.max(3, (chartW - padL - 8) / sampled.length - 2);

  return (
    <div style={{ overflowX: "auto" }}>
      <svg width="100%" height={chartH} viewBox={`0 0 ${chartW} ${chartH}`} preserveAspectRatio="xMinYMid meet">
        {[0, 0.5, 1].map((r) => {
          const y = padT + innerH * (1 - r);
          return (
            <g key={r}>
              <line x1={padL} y1={y} x2={chartW - 4} y2={y} stroke="rgba(0,0,0,0.06)" />
              <text x={padL - 4} y={y + 3} textAnchor="end" fontSize={9} fill="#8c8c8c">
                {Math.round(max * r)}
              </text>
            </g>
          );
        })}
        {sampled.map((b, i) => {
          const x = padL + i * (barW + 2);
          const h = Math.max(1, (b.count / max) * innerH);
          const y = padT + innerH - h;
          return (
            <g key={`${b.time}-${i}`}>
              <title>{`${formatDateTime(b.time)}: ${b.count}`}</title>
              <rect x={x} y={y} width={barW} height={h} rx={1} fill="#1677ff" opacity={0.88} />
            </g>
          );
        })}
      </svg>
    </div>
  );
}
