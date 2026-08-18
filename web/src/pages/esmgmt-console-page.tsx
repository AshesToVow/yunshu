import { PlayCircleOutlined } from "@ant-design/icons";
import { Button, Card, Input, Select, Space, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { OpsPageHeader } from "../components/ops/ops-page-header";
import { listEsmgmtConnections, proxyEsmgmtREST, type EsmgmtConnection } from "../services/esmgmt";
import { extractApiErrorMessage } from "../services/http";

const METHODS = ["GET", "POST", "PUT", "DELETE", "HEAD"] as const;

type QuickCommand = {
  label: string;
  method: (typeof METHODS)[number];
  path: string;
  body?: string;
};

const QUICK_COMMANDS: QuickCommand[] = [
  { label: "集群健康", method: "GET", path: "/_cluster/health" },
  { label: "集群设置", method: "GET", path: "/_cluster/settings?include_defaults=true" },
	{ label: "节点列表", method: "GET", path: "/_cat/nodes?v&format=json" },
  { label: "索引列表", method: "GET", path: "/_cat/indices?v&format=json" },
  { label: "分片列表", method: "GET", path: "/_cat/shards?v&format=json" },
  { label: "别名列表", method: "GET", path: "/_cat/aliases?v&format=json" },
  { label: "节点统计", method: "GET", path: "/_nodes/stats" },
  { label: "索引 Mapping", method: "GET", path: "/my-index/_mapping" },
  { label: "索引 Settings", method: "GET", path: "/my-index/_settings" },
  { label: "搜索文档", method: "POST", path: "/my-index/_search", body: '{\n  "query": { "match_all": {} },\n  "size": 10\n}' },
  { label: "创建/更新索引", method: "PUT", path: "/my-index", body: '{\n  "settings": { "number_of_shards": 1, "number_of_replicas": 0 },\n  "mappings": { "properties": { "title": { "type": "text" } } }\n}' },
  { label: "更新 Settings", method: "PUT", path: "/my-index/_settings", body: '{\n  "index": { "number_of_replicas": 0 }\n}' },
  { label: "写入文档", method: "PUT", path: "/my-index/_doc/1", body: '{\n  "title": "hello"\n}' },
  { label: "删除文档", method: "DELETE", path: "/my-index/_doc/1" },
  { label: "删除索引", method: "DELETE", path: "/my-index" },
  { label: "刷新索引", method: "POST", path: "/my-index/_refresh" },
  { label: "打开索引", method: "POST", path: "/my-index/_open" },
  { label: "关闭索引", method: "POST", path: "/my-index/_close" },
  { label: "管理别名", method: "POST", path: "/_aliases", body: '{\n  "actions": [\n    { "add": { "index": "my-index", "alias": "my-alias" } }\n  ]\n}' },
];

export function EsmgmtConsolePage() {
  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [method, setMethod] = useState<(typeof METHODS)[number]>("GET");
  const [path, setPath] = useState("/_cluster/health");
  const [body, setBody] = useState("");
  const [result, setResult] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void listEsmgmtConnections({ include_log_platform: true })
      .then((list) => {
        setConnections(list || []);
        const logPlat = list?.find((c) => c.id === 0);
        const def = logPlat || list?.find((c) => c.is_default) || list?.[0];
        if (def) setConnectionId(def.id);
      })
      .catch((e) => message.error(extractApiErrorMessage(e, "加载连接失败")));
  }, []);

  function applyQuickCommand(key: string) {
    const cmd = QUICK_COMMANDS.find((c) => `${c.method} ${c.path}` === key);
    if (!cmd) return;
    setMethod(cmd.method);
    setPath(cmd.path);
    setBody(cmd.body || "");
  }

  async function run() {
    setLoading(true);
    try {
      const res = await proxyEsmgmtREST({
        connection_id: connectionId,
        method,
        path,
        body: body.trim() || undefined,
      });
      setResult(JSON.stringify({ status: res.status, body: res.body }, null, 2));
    } catch (e) {
      message.error(extractApiErrorMessage(e, "请求失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="page-stack">
      <OpsPageHeader
        title="ES REST 控制台"
        description="管理员受限代理：支持 GET/POST/PUT/DELETE/HEAD，禁止脚本执行与节点关机。"
        breadcrumbs={[{ title: "ES 管理" }, { title: "REST 控制台" }]}
        extra={
          <Space>
            <Link to="/esmgmt/overview">集群概览</Link>
            <Link to="/esmgmt/connections">连接管理</Link>
          </Space>
        }
      />
      <Card className="table-card">
      <Space direction="vertical" style={{ width: "100%" }} size="middle">
        <Typography.Text type="secondary">
          支持 GET / POST / PUT / DELETE / HEAD。允许集群探查、索引与文档读写、别名与模板等管理操作；禁止脚本执行与节点关机。
        </Typography.Text>
        <Select
          style={{ width: "100%", maxWidth: 560 }}
          placeholder="常用命令（选择后可再改路径/Body）"
          allowClear
          options={QUICK_COMMANDS.map((c) => ({
            value: `${c.method} ${c.path}`,
            label: `${c.label} · ${c.method} ${c.path}`,
          }))}
          onChange={(v) => {
            if (typeof v === "string") applyQuickCommand(v);
          }}
        />
        <Space wrap style={{ width: "100%" }}>
          <Select
            style={{ minWidth: 200 }}
            placeholder="连接"
            value={connectionId}
            options={connections.map((c) => ({ value: c.id, label: c.name }))}
            onChange={setConnectionId}
            allowClear
          />
          <Select
            style={{ width: 110 }}
            value={method}
            options={METHODS.map((m) => ({ value: m, label: m }))}
            onChange={setMethod}
          />
          <Input style={{ minWidth: 320, flex: 1 }} value={path} onChange={(e) => setPath(e.target.value)} />
          <Button type="primary" icon={<PlayCircleOutlined />} loading={loading} onClick={() => void run()}>
            执行
          </Button>
        </Space>
        <Input.TextArea
          rows={6}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="可选 JSON Body（POST / PUT 常用；DELETE 一般为空）"
        />
        <pre className="code-block-panel" style={{ minHeight: 280, margin: 0 }}>
          {result || "响应将显示在这里"}
        </pre>
      </Space>
    </Card>
    </div>
  );
}

export default EsmgmtConsolePage;
