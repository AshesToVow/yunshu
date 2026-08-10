import { ClearOutlined, RobotOutlined, SendOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Input, InputNumber, Select, Space, Switch, Tag, Typography, message } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  chatAI,
  getAIStatus,
  pingAI,
  type AIChatMessage,
  type AIChatResult,
  type AIStatus,
} from "../services/ai";
import { extractApiErrorMessage } from "../services/http";

type Bubble = AIChatMessage & { id: string; meta?: string };

export function AiAssistantPage() {
  const [status, setStatus] = useState<AIStatus | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);
  const [provider, setProvider] = useState<string>("");
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [pinging, setPinging] = useState(false);
  const [messages, setMessages] = useState<Bubble[]>([]);
  const [enableTools, setEnableTools] = useState(true);
  const [enableWrite, setEnableWrite] = useState(false);
  const [clusterId, setClusterId] = useState<number>();
  const [namespace, setNamespace] = useState("");
  const [lastSteps, setLastSteps] = useState<AIChatResult["tool_steps"]>([]);
  const listRef = useRef<HTMLDivElement>(null);

  const providerOptions = useMemo(() => {
    const list = status?.providers || [];
    return list.map((p) => ({
      value: p.name,
      label: `${p.name}${p.configured ? "" : "（未配置 Key）"} · ${p.model || "-"}`,
      disabled: !p.configured,
    }));
  }, [status]);

  useEffect(() => {
    void loadStatus();
  }, []);

  useEffect(() => {
    const el = listRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages, sending]);

  async function loadStatus() {
    setStatusLoading(true);
    try {
      const res = await getAIStatus();
      setStatus(res);
      const def = res.default_provider || "";
      const configured = res.providers?.find((p) => p.name === def && p.configured);
      const firstConfigured = res.providers?.find((p) => p.configured);
      setProvider(configured?.name || firstConfigured?.name || def || "");
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载 AI 状态失败"));
    } finally {
      setStatusLoading(false);
    }
  }

  async function handlePing() {
    setPinging(true);
    try {
      const res = await pingAI(provider);
      if (res.ok) {
        message.success(`连通成功：${res.provider} / ${res.model}`);
      } else {
        message.warning(res.message || "连通失败");
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "连通测试失败"));
    } finally {
      setPinging(false);
    }
  }

  async function handleSend() {
    const text = input.trim();
    if (!text || sending) return;
    const userMsg: Bubble = { id: `u-${Date.now()}`, role: "user", content: text };
    const next = [...messages, userMsg];
    setMessages(next);
    setInput("");
    setSending(true);
    try {
      const res = await chatAI({
        provider: provider || undefined,
        messages: next.map(({ role, content }) => ({ role, content })),
        cluster_id: clusterId,
        namespace: namespace || undefined,
        enable_tools: enableTools,
        enable_write_tools: enableWrite,
      });
      setLastSteps(res.tool_steps || []);
      const meta =
        (res.tool_steps?.length ? `工具 ${res.tool_steps.length} 次` : "") +
        (res.rag_hits?.length ? ` · RAG ${res.rag_hits.length}` : "");
      setMessages((prev) => [
        ...prev,
        { id: `a-${Date.now()}`, role: "assistant", content: res.reply || "（空回复）", meta },
      ]);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "对话失败"));
    } finally {
      setSending(false);
    }
  }

  return (
    <div>
      <Card className="table-card">
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Space wrap style={{ width: "100%", justifyContent: "space-between" }}>
            <Space>
              <RobotOutlined />
              <Typography.Title level={4} style={{ margin: 0 }}>
                AI 运维助手
              </Typography.Title>
              {status ? (
                <Tag color={status.enabled ? "success" : "default"}>
                  {status.enabled ? "已启用" : "未启用"}
                </Tag>
              ) : null}
            </Space>
            <Space wrap>
              <Select
                style={{ minWidth: 260 }}
                placeholder="选择 Provider"
                value={provider || undefined}
                options={providerOptions}
                onChange={setProvider}
                loading={statusLoading}
              />
              <Button onClick={() => void loadStatus()} loading={statusLoading}>
                刷新状态
              </Button>
              <Button onClick={() => void handlePing()} loading={pinging} disabled={!status?.enabled}>
                连通测试
              </Button>
              <Button
                icon={<ClearOutlined />}
                onClick={() => setMessages([])}
                disabled={messages.length === 0 || sending}
              >
                清空
              </Button>
            </Space>
          </Space>

          {!status?.enabled ? (
            <Alert
              type="info"
              showIcon
              message="AI 未启用"
              description="请在数据字典将 ai_enabled 设为 true，并配置对应 Provider 的 api_key（status=1）。"
            />
          ) : (
            <Alert
              type="warning"
              showIcon
              message="工具增强对话"
              description="默认启用只读工具（list/diagnose/logs）。写操作需勾选并进入审批；请勿粘贴密钥。"
            />
          )}

          <Space wrap>
            <span>工具</span>
            <Switch checked={enableTools} onChange={setEnableTools} />
            <span>写工具(审批)</span>
            <Switch checked={enableWrite} onChange={setEnableWrite} disabled={!enableTools} />
            <InputNumber
              placeholder="cluster_id"
              value={clusterId}
              onChange={(v) => setClusterId(typeof v === "number" ? v : undefined)}
              style={{ width: 140 }}
            />
            <Input
              placeholder="namespace"
              value={namespace}
              onChange={(e) => setNamespace(e.target.value)}
              style={{ width: 160 }}
            />
          </Space>
          <div
            ref={listRef}
            style={{
              height: 420,
              overflow: "auto",
              padding: 12,
              background: "var(--ant-color-fill-quaternary, #fafafa)",
              borderRadius: 8,
              border: "1px solid var(--ant-color-border-secondary, #f0f0f0)",
            }}
          >
            {messages.length === 0 ? (
              <Typography.Text type="secondary">
                可询问：Pod 异常排查思路、发布回滚检查项、告警噪音治理建议等。
              </Typography.Text>
            ) : (
              <Space direction="vertical" style={{ width: "100%" }} size="middle">
                {messages.map((m) => (
                  <div
                    key={m.id}
                    style={{
                      alignSelf: m.role === "user" ? "flex-end" : "flex-start",
                      maxWidth: "88%",
                      marginLeft: m.role === "user" ? "auto" : 0,
                      padding: "10px 12px",
                      borderRadius: 8,
                      background: m.role === "user" ? "var(--ant-color-primary-bg, #e6f4ff)" : "#fff",
                      border: "1px solid var(--ant-color-border-secondary, #f0f0f0)",
                      whiteSpace: "pre-wrap",
                      wordBreak: "break-word",
                    }}
                  >
                    <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                      {m.role === "user" ? "我" : "助手"}
                      {m.meta ? ` · ${m.meta}` : ""}
                    </Typography.Text>
                    <div>{m.content}</div>
                  </div>
                ))}
                {sending ? <Typography.Text type="secondary">思考中...</Typography.Text> : null}
              </Space>
            )}
          </div>

          {lastSteps && lastSteps.length > 0 ? (
            <Card size="small" title="最近工具调用">
              <Space direction="vertical" style={{ width: "100%" }}>
                {lastSteps.map((s, i) => (
                  <Alert
                    key={`${s.name}-${i}`}
                    type={s.ok ? "success" : "error"}
                    showIcon
                    message={s.name}
                    description={
                      <pre style={{ margin: 0, maxHeight: 120, overflow: "auto", whiteSpace: "pre-wrap", fontSize: 12 }}>
                        {s.error || s.result || s.args}
                      </pre>
                    }
                  />
                ))}
              </Space>
            </Card>
          ) : null}
          <Space.Compact style={{ width: "100%" }}>
            <Input.TextArea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="输入运维问题，Enter 发送（Shift+Enter 换行）"
              autoSize={{ minRows: 2, maxRows: 6 }}
              disabled={!status?.enabled || sending}
              onPressEnter={(e) => {
                if (!e.shiftKey) {
                  e.preventDefault();
                  void handleSend();
                }
              }}
            />
            <Button
              type="primary"
              icon={<SendOutlined />}
              loading={sending}
              disabled={!status?.enabled || !input.trim()}
              onClick={() => void handleSend()}
              style={{ height: "auto" }}
            >
              发送
            </Button>
          </Space.Compact>
        </Space>
      </Card>
    </div>
  );
}

export default AiAssistantPage;
