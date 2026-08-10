import { PlayCircleOutlined } from "@ant-design/icons";
import { Button, Card, Input, Select, Space, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { listEsmgmtConnections, proxyEsmgmtREST, type EsmgmtConnection } from "../services/esmgmt";
import { extractApiErrorMessage } from "../services/http";

export function EsmgmtConsolePage() {
  const [connections, setConnections] = useState<EsmgmtConnection[]>([]);
  const [connectionId, setConnectionId] = useState<number>();
  const [method, setMethod] = useState("GET");
  const [path, setPath] = useState("/_cluster/health");
  const [body, setBody] = useState("");
  const [result, setResult] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void listEsmgmtConnections()
      .then((list) => {
        setConnections(list || []);
        const def = list?.find((c) => c.is_default) || list?.[0];
        if (def) setConnectionId(def.id);
      })
      .catch(() => undefined);
  }, []);

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
    <Card className="table-card" title="ES REST 控制台（受限代理）">
      <Space direction="vertical" style={{ width: "100%" }} size="middle">
        <Typography.Text type="secondary">
          仅允许 _cluster / _cat / _nodes / _search 等只读探查路径，禁止脚本执行。
        </Typography.Text>
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
            options={["GET", "POST", "HEAD"].map((m) => ({ value: m, label: m }))}
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
          placeholder="可选 JSON Body（POST）"
        />
        <pre className="code-block-panel" style={{ minHeight: 280, margin: 0 }}>
          {result || "响应将显示在这里"}
        </pre>
      </Space>
    </Card>
  );
}

export default EsmgmtConsolePage;
