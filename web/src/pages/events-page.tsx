import { Card, Radio, Space, Table, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { K8sPageToolbar } from "../components/ops/k8s-page-toolbar";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import { useK8sContext } from "../hooks/use-k8s-context";
import { useK8sWatch } from "../hooks/use-k8s-watch";
import { listEvents, listEventsGrouped, type EventGroupItem, type EventItem } from "../services/events";
import { buildK8sResourceLink } from "../utils/k8s-resource-links";

type ViewMode = "list" | "grouped";

function InvolvedObjectLink({
  clusterId,
  namespace,
  kind,
  name,
}: {
  clusterId?: number;
  namespace?: string;
  kind?: string;
  name?: string;
}) {
  const href = buildK8sResourceLink({ kind, name, clusterId, namespace });
  const label = `${kind ?? "-"} / ${name ?? "-"}`;
  if (!href) return <span>{label}</span>;
  return (
    <Link to={href}>
      <Tag color="blue">{kind || "?"}</Tag> {name || "-"}
    </Link>
  );
}

export function EventsPage() {
  const {
    clusterId,
    namespace = "default",
    setClusterId,
    setNamespace,
    clusterOptions,
    namespaceOptions,
  } = useK8sContext({ needNamespace: true, syncUrl: true });
  const [keyword, setKeyword] = useState("");
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [watchLive, setWatchLive] = useState(false);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<EventItem[]>([]);
  const [grouped, setGrouped] = useState<EventGroupItem[]>([]);

  const filterRef = useRef({ clusterId, namespace, keyword, viewMode });
  filterRef.current = { clusterId, namespace, keyword, viewMode };

  const reload = useCallback(async (overrideKeyword?: string, opts?: { silent?: boolean }) => {
    const { clusterId: cid, namespace: ns, keyword: kw, viewMode: mode } = filterRef.current;
    if (!cid) return;
    const effectiveKeyword = (overrideKeyword ?? kw).trim();
    const silent = Boolean(opts?.silent);
    if (!silent) setLoading(true);
    try {
      const params = {
        cluster_id: cid,
        namespace: ns,
        keyword: effectiveKeyword || undefined,
        limit: 500,
      };
      if (mode === "grouped") {
        setGrouped(await listEventsGrouped(params));
      } else {
        setData(await listEvents(params));
      }
    } catch {
      // http 拦截器已 toast
    } finally {
      if (!silent) setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [clusterId, namespace, viewMode, reload]);

  useK8sWatch({
    enabled: watchLive,
    clusterId,
    namespace,
    resource: "events",
    onRefresh: () => void reload(undefined, { silent: true }),
    onDisabled: () => setWatchLive(false),
  });

  const listColumns: ColumnsType<EventItem> = [
    { title: "时间", dataIndex: "last_time", width: 180, render: (v: string, r) => v || r.creation_time || "-" },
    { title: "命名空间", dataIndex: "namespace", width: 140 },
    { title: "类型", dataIndex: "type", width: 90, render: (v: string) => <Tag color={v === "Warning" ? "error" : "success"}>{v || "-"}</Tag> },
    { title: "原因", dataIndex: "reason", width: 160 },
    {
      title: "对象",
      key: "obj",
      width: 240,
      render: (_, r) => (
        <InvolvedObjectLink clusterId={clusterId} namespace={r.namespace} kind={r.involved_kind} name={r.involved_name} />
      ),
    },
    { title: "次数", dataIndex: "count", width: 80 },
    { title: "消息", dataIndex: "message", ellipsis: true },
  ];

  const groupedColumns: ColumnsType<EventGroupItem> = [
    { title: "最近时间", dataIndex: "last_time", width: 180 },
    { title: "命名空间", dataIndex: "namespace", width: 120 },
    {
      title: "对象",
      key: "obj",
      width: 240,
      render: (_, r) => (
        <InvolvedObjectLink clusterId={clusterId} namespace={r.namespace} kind={r.involved_kind} name={r.involved_name} />
      ),
    },
    { title: "原因", dataIndex: "reason", width: 140 },
    { title: "类型", dataIndex: "type", width: 90, render: (v) => <Tag color={v === "Warning" ? "error" : "success"}>{v}</Tag> },
    { title: "事件条数", dataIndex: "event_count", width: 90 },
    { title: "累计次数", dataIndex: "total_count", width: 90 },
    { title: "最新消息", dataIndex: "message", ellipsis: true },
  ];

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="Event 事件"
        description="集群与命名空间内 Kubernetes 事件检索与聚合"
        breadcrumbs={[{ title: "Kubernetes" }, { title: "Events" }]}
      />
      <Card className="table-card yaml-crud-card" bordered={false}>
        <K8sPageToolbar
          clusterId={clusterId}
          namespace={namespace}
          clusterOptions={clusterOptions}
          namespaceOptions={namespaceOptions}
          searchPlaceholder="搜索 reason / message / 对象"
          onClusterChange={setClusterId}
          onNamespaceChange={setNamespace}
          onSearch={(v) => {
            setKeyword(v);
            void reload(v);
          }}
          onRefresh={() => void reload()}
          watchLive={watchLive}
          onWatchChange={setWatchLive}
          extraRight={
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value as ViewMode)}>
              <Radio.Button value="list">列表</Radio.Button>
              <Radio.Button value="grouped">按对象聚合</Radio.Button>
            </Radio.Group>
          }
        />
        {viewMode === "list" ? (
          <Table
            size="small"
            rowKey={(r) => `${r.namespace}/${r.involved_kind}/${r.involved_name}/${r.last_time}/${r.reason}`}
            loading={loading}
            dataSource={data}
            columns={listColumns}
            pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
          />
        ) : (
          <Table
            size="small"
            rowKey={(r) => `${r.namespace}/${r.involved_kind}/${r.involved_name}/${r.reason}`}
            loading={loading}
            dataSource={grouped}
            columns={groupedColumns}
            expandable={{
              expandedRowRender: (r) => (
                <Table
                  size="small"
                  pagination={false}
                  dataSource={r.events ?? []}
                  rowKey={(e) => `${e.last_time}/${e.message}`}
                  columns={[
                    { title: "时间", dataIndex: "last_time", width: 160 },
                    { title: "次数", dataIndex: "count", width: 70 },
                    { title: "消息", dataIndex: "message" },
                  ]}
                />
              ),
            }}
            pagination={{ pageSize: 10, showSizeChanger: true }}
          />
        )}
      </Card>
    </div>
  );
}
