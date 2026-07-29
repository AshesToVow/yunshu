import { ArrowLeftOutlined, LinkOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Col, Descriptions, Empty, List, Row, Select, Space, Table, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { getProjects, type ProjectItem } from "../services/projects";
import { listServiceCatalog, type ServiceCatalogItem } from "../services/service-catalog";
import { getServicePortrait, type ServicePortrait } from "../services/service-portrait";
import { formatDateTime } from "../utils/format";

export function ServicePortraitPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [services, setServices] = useState<ServiceCatalogItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [catalogId, setCatalogId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [portrait, setPortrait] = useState<ServicePortrait | null>(null);

  const projectOptions = useMemo(
    () => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })),
    [projects],
  );
  const serviceOptions = useMemo(
    () => services.map((s) => ({ value: s.id, label: `${s.name} (${s.identifier})` })),
    [services],
  );

  useEffect(() => {
    void (async () => {
      const p = await getProjects({ page: 1, page_size: 1000 });
      setProjects(p.list);
      const qPid = Number(searchParams.get("project_id") || 0);
      const initial = qPid || p.list[0]?.id;
      if (initial) setProjectId(initial);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void (async () => {
      const res = await listServiceCatalog(projectId, { page: 1, page_size: 500 });
      setServices(res.list);
      const qCid = Number(searchParams.get("catalog_id") || 0);
      const next = qCid && res.list.some((s) => s.id === qCid) ? qCid : res.list[0]?.id;
      setCatalogId(next);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  useEffect(() => {
    if (!projectId || !catalogId) {
      setPortrait(null);
      return;
    }
    setSearchParams({ project_id: String(projectId), catalog_id: String(catalogId) }, { replace: true });
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, catalogId]);

  async function load() {
    if (!projectId || !catalogId) return;
    setLoading(true);
    try {
      setPortrait(await getServicePortrait(projectId, catalogId));
    } finally {
      setLoading(false);
    }
  }

  const svc = portrait?.service;

  return (
    <Space direction="vertical" size="middle" style={{ width: "100%" }}>
      <Card
        title="服务画像"
        extra={
          <Space wrap>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate("/service-catalog")}>
              服务目录
            </Button>
            <Select style={{ width: 240 }} value={projectId} onChange={setProjectId} options={projectOptions} />
            <Select style={{ width: 260 }} value={catalogId} onChange={setCatalogId} options={serviceOptions} placeholder="选择服务" />
            <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
              刷新
            </Button>
          </Space>
        }
      >
        {!svc ? (
          <Empty description="请先在服务目录创建并绑定服务" />
        ) : (
          <Descriptions bordered size="small" column={2}>
            <Descriptions.Item label="标识">{svc.identifier}</Descriptions.Item>
            <Descriptions.Item label="名称">{svc.name}</Descriptions.Item>
            <Descriptions.Item label="负责人">{svc.owner || "-"}</Descriptions.Item>
            <Descriptions.Item label="产品线">{svc.product_line || "-"}</Descriptions.Item>
            <Descriptions.Item label="关键等级">
              <Tag>{svc.criticality || "normal"}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">{svc.status === 1 ? "启用" : "停用"}</Descriptions.Item>
            {portrait?.health ? (
              <Descriptions.Item label="健康分" span={2}>
                <Space>
                  <Tag color={portrait.health.grade === "A" ? "green" : portrait.health.grade === "B" ? "blue" : "orange"}>
                    {portrait.health.score} / {portrait.health.grade}
                  </Tag>
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {portrait.health.factors?.map((f) => `${f.label}${f.score}`).join(" · ")}
                  </Typography.Text>
                </Space>
              </Descriptions.Item>
            ) : null}
          </Descriptions>
          {portrait?.health?.factors?.length ? (
            <Table
              style={{ marginTop: 12 }}
              size="small"
              pagination={false}
              rowKey="key"
              dataSource={portrait.health.factors}
              columns={[
                { title: "因子", dataIndex: "label", width: 100 },
                { title: "得分", width: 100, render: (_: unknown, r: { score: number; max: number }) => `${r.score}/${r.max}` },
                { title: "扣分", dataIndex: "deduct", width: 80 },
                { title: "说明", dataIndex: "detail" },
              ]}
            />
          ) : null}
        )}
      </Card>

      <Row gutter={16}>
        <Col xs={24} lg={10}>
          <Card title="入口" loading={loading} size="small">
            <List
              dataSource={portrait?.entry_points || []}
              locale={{ emptyText: "暂无绑定入口" }}
              renderItem={(it) => (
                <List.Item>
                  <Space>
                    <Tag>{it.kind}</Tag>
                    <Link to={it.path}>
                      <LinkOutlined /> {it.label}
                    </Link>
                    {it.hint ? (
                      <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                        {it.hint}
                      </Typography.Text>
                    ) : null}
                  </Space>
                </List.Item>
              )}
            />
          </Card>
          {portrait?.cicd_summary ? (
            <Card title="CI/CD 摘要" size="small" style={{ marginTop: 16 }}>
              <Descriptions size="small" column={1}>
                <Descriptions.Item label="服务">{portrait.cicd_summary.name}</Descriptions.Item>
                <Descriptions.Item label="最近发布">{portrait.cicd_summary.last_title || "-"}</Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Tag>{portrait.cicd_summary.last_status || "-"}</Tag>
                </Descriptions.Item>
                <Descriptions.Item label="时间">
                  {portrait.cicd_summary.last_at ? formatDateTime(portrait.cicd_summary.last_at) : "-"}
                </Descriptions.Item>
              </Descriptions>
            </Card>
          ) : null}
        </Col>
        <Col xs={24} lg={14}>
          <Card title="最近变更" loading={loading} size="small">
            <Table
              rowKey="id"
              size="small"
              pagination={false}
              dataSource={portrait?.recent_changes || []}
              columns={[
                { title: "时间", dataIndex: "started_at", width: 160, render: (v: string) => formatDateTime(v) },
                { title: "来源", dataIndex: "source", width: 80, render: (v: string) => <Tag>{v}</Tag> },
                { title: "动作", dataIndex: "action", width: 120 },
                { title: "摘要", dataIndex: "summary" },
              ]}
            />
          </Card>
        </Col>
      </Row>
    </Space>
  );
}
