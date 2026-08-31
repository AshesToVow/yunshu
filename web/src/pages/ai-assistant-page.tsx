import { Link } from "react-router-dom";
import {
  ClearOutlined,
  DeleteOutlined,
  PlusOutlined,
  RobotOutlined,
  SendOutlined,
} from "@ant-design/icons";
import { Alert, Button, Card, Input, List, Select, Space, Switch, Tag, Typography, message } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  chatAIStream,
  clearAISession,
  createAISession,
  deleteAISession,
  getAISession,
  getAIStatus,
  listAISessions,
  pingAI,
  type AIChatMessage,
  type AIChatResult,
  type AIChatSession,
  type AIChatStreamEvent,
  type AIStatus,
} from "../services/ai";
import { getClusters, type ClusterItem } from "../services/clusters";
import { getProjects, type ProjectItem } from "../services/projects";
import { extractApiErrorMessage } from "../services/http";

type Bubble = AIChatMessage & { id: string; meta?: string };

const PREFS_KEY = "yunshu.ai.assistant.prefs.v1";

type Prefs = {
  clusterId?: number;
  projectId?: number;
  enableTools: boolean;
  enableWrite: boolean;
  provider?: string;
};

function loadPrefs(): Prefs {
  try {
    const raw = localStorage.getItem(PREFS_KEY);
    if (!raw) return { enableTools: true, enableWrite: false };
    const parsed = JSON.parse(raw) as Prefs;
    return {
      enableTools: parsed.enableTools ?? true,
      enableWrite: parsed.enableWrite ?? false,
      clusterId: parsed.clusterId,
      projectId: parsed.projectId,
      provider: parsed.provider,
    };
  } catch {
    return { enableTools: true, enableWrite: false };
  }
}

function savePrefs(prefs: Prefs) {
  try {
    localStorage.setItem(PREFS_KEY, JSON.stringify(prefs));
  } catch {
    /* ignore */
  }
}

function parseAssistantMeta(metaJSON?: string): string {
  if (!metaJSON) return "";
  try {
    const meta = JSON.parse(metaJSON) as {
      tool_steps?: unknown[];
      rag_hits?: unknown[];
    };
    const parts: string[] = [];
    if (meta.tool_steps?.length) parts.push(`工具 ${meta.tool_steps.length} 次`);
    if (meta.rag_hits?.length) parts.push(`RAG ${meta.rag_hits.length}`);
    return parts.join(" · ");
  } catch {
    return "";
  }
}

export function AiAssistantPage() {
  const prefs = useMemo(() => loadPrefs(), []);
  const [status, setStatus] = useState<AIStatus | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);
  const [provider, setProvider] = useState<string>(prefs.provider || "");
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  const [pinging, setPinging] = useState(false);
  const [messages, setMessages] = useState<Bubble[]>([]);
  const [enableTools, setEnableTools] = useState(prefs.enableTools);
  const [enableWrite, setEnableWrite] = useState(prefs.enableWrite);
  const [clusterId, setClusterId] = useState<number | undefined>(prefs.clusterId);
  const [projectId, setProjectId] = useState<number | undefined>(prefs.projectId);
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [lastSteps, setLastSteps] = useState<AIChatResult["tool_steps"]>([]);
  const [lastRag, setLastRag] = useState<AIChatResult["rag_hits"]>([]);
  const [liveTimeline, setLiveTimeline] = useState<string[]>([]);
  const [sessions, setSessions] = useState<AIChatSession[]>([]);
  const [sessionId, setSessionId] = useState<number | undefined>();
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const listRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  const providerOptions = useMemo(() => {
    const list = status?.providers || [];
    return list.map((p) => ({
      value: p.name,
      label: `${p.source === "db" ? "[库] " : ""}${p.name}${p.configured ? "" : "（未配置 Key）"} · ${p.model || "-"}`,
      disabled: !p.configured,
    }));
  }, [status]);

  const clusterOptions = useMemo(
    () => clusters.map((c) => ({ value: c.id, label: c.name || `集群 #${c.id}` })),
    [clusters],
  );

  const projectOptions = useMemo(
    () => projects.map((p) => ({ value: p.id, label: p.name || `项目 #${p.id}` })),
    [projects],
  );

  useEffect(() => {
    void loadStatus();
    void getClusters({ page: 1, page_size: 1000 })
      .then((res) => setClusters(res?.list || []))
      .catch((e) => message.error(extractApiErrorMessage(e, "加载集群失败")));
    void getProjects({ page: 1, page_size: 1000 })
      .then((res) => setProjects(res?.list || []))
      .catch((e) => message.error(extractApiErrorMessage(e, "加载项目失败")));
    void refreshSessions(true);
  }, []);

  useEffect(() => {
    savePrefs({ clusterId, projectId, enableTools, enableWrite, provider });
  }, [clusterId, projectId, enableTools, enableWrite, provider]);

  useEffect(() => {
    const el = listRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages, sending]);

  async function loadStatus() {
    setStatusLoading(true);
    try {
      const res = await getAIStatus();
      setStatus(res);
      if (!provider) {
        const def = res.default_provider || "";
        const configured = res.providers?.find((p) => p.name === def && p.configured);
        const firstConfigured = res.providers?.find((p) => p.configured);
        setProvider(configured?.name || firstConfigured?.name || def || "");
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载 AI 状态失败"));
    } finally {
      setStatusLoading(false);
    }
  }

  async function refreshSessions(selectLatest = false) {
    setSessionsLoading(true);
    try {
      const res = await listAISessions({ page: 1, page_size: 50 });
      const list = res?.list || [];
      setSessions(list);
      if (selectLatest) {
        if (list.length > 0) {
          await openSession(list[0].id);
        } else {
          await handleNewSession();
        }
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "加载会话失败"));
    } finally {
      setSessionsLoading(false);
    }
  }

  async function openSession(id: number) {
    try {
      const detail = await getAISession(id);
      setSessionId(detail.session.id);
      if (detail.session.cluster_id) setClusterId(detail.session.cluster_id);
      if (detail.session.project_id) setProjectId(detail.session.project_id);
      if (detail.session.provider) setProvider(detail.session.provider);
      setEnableTools(detail.session.enable_tools);
      setEnableWrite(detail.session.enable_write);
      const bubbles: Bubble[] = (detail.messages || []).map((m) => ({
        id: `m-${m.id}`,
        role: m.role === "assistant" ? "assistant" : "user",
        content: m.content,
        meta: m.role === "assistant" ? parseAssistantMeta(m.meta_json) : undefined,
      }));
      setMessages(bubbles);
      const lastAssistant = [...(detail.messages || [])].reverse().find((m) => m.role === "assistant");
      if (lastAssistant?.meta_json) {
        try {
          const meta = JSON.parse(lastAssistant.meta_json) as { tool_steps?: AIChatResult["tool_steps"] };
          setLastSteps(meta.tool_steps || []);
        } catch {
          setLastSteps([]);
        }
      } else {
        setLastSteps([]);
      }
    } catch (e) {
      message.error(extractApiErrorMessage(e, "打开会话失败"));
    }
  }

  async function handleNewSession() {
    try {
      const sess = await createAISession({
        provider: provider || undefined,
        cluster_id: clusterId,
        project_id: projectId,
        enable_tools: enableTools,
        enable_write: enableWrite,
      });
      setSessionId(sess.id);
      setMessages([]);
      setLastSteps([]);
      setSessions((prev) => [sess, ...prev.filter((s) => s.id !== sess.id)]);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "创建会话失败"));
    }
  }

  async function handleDeleteSession(id: number) {
    try {
      await deleteAISession(id);
      const next = sessions.filter((s) => s.id !== id);
      setSessions(next);
      if (sessionId === id) {
        if (next.length > 0) {
          await openSession(next[0].id);
        } else {
          await handleNewSession();
        }
      }
      message.success("会话已删除");
    } catch (e) {
      message.error(extractApiErrorMessage(e, "删除会话失败"));
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
    let sid = sessionId;
    if (!sid) {
      try {
        const sess = await createAISession({
          provider: provider || undefined,
          cluster_id: clusterId,
          project_id: projectId,
          enable_tools: enableTools,
          enable_write: enableWrite,
        });
        sid = sess.id;
        setSessionId(sid);
        setSessions((prev) => [sess, ...prev]);
      } catch (e) {
        message.error(extractApiErrorMessage(e, "创建会话失败"));
        return;
      }
    }
    const userMsg: Bubble = { id: `u-${Date.now()}`, role: "user", content: text };
    const next = [...messages, userMsg];
    setMessages(next);
    setInput("");
    setSending(true);
    setLiveTimeline(["开始对话…"]);
    setLastSteps([]);
    setLastRag([]);
    abortRef.current?.abort();
    const ac = new AbortController();
    abortRef.current = ac;

    const steps: NonNullable<AIChatResult["tool_steps"]> = [];
    let replyText = "";
    let finalSid = sid;
    let ragHits: AIChatResult["rag_hits"] = [];

    const pushTimeline = (line: string) => {
      setLiveTimeline((prev) => [...prev.slice(-40), line]);
    };

    try {
      await chatAIStream(
        {
          provider: provider || undefined,
          session_id: sid,
          messages: next.map(({ role, content }) => ({ role, content })),
          cluster_id: clusterId,
          project_id: projectId,
          enable_tools: enableTools,
          enable_write_tools: enableWrite,
        },
        (ev: AIChatStreamEvent) => {
          if (ev.session_id) finalSid = ev.session_id;
          switch (ev.type) {
            case "progress":
              if (ev.message) pushTimeline(ev.message);
              break;
            case "rag":
              ragHits = ev.rag_hits || [];
              setLastRag(ragHits);
              pushTimeline(`RAG 命中 ${(ragHits || []).length} 条`);
              break;
            case "tool":
              if (ev.tool_step) {
                steps.push(ev.tool_step);
                setLastSteps([...steps]);
                pushTimeline(
                  `工具 ${ev.tool_step.name} ${ev.tool_step.ok === false ? "失败" : "完成"}`,
                );
              }
              break;
            case "reply":
              replyText = ev.reply || replyText;
              break;
            case "error":
              pushTimeline(`错误: ${ev.error || ev.message || "unknown"}`);
              message.error(ev.error || ev.message || "对话失败");
              break;
            case "done":
              if (ev.reply) replyText = ev.reply;
              if (ev.session_id) finalSid = ev.session_id;
              break;
            default:
              break;
          }
        },
        ac.signal,
      );

      if (finalSid && finalSid !== sid) {
        setSessionId(finalSid);
        sid = finalSid;
      }
      const metaParts: string[] = [];
      if (steps.length) metaParts.push(`工具 ${steps.length} 次`);
      if (ragHits?.length) metaParts.push(`RAG ${ragHits.length}`);
      setMessages((prev) => [
        ...prev,
        {
          id: `a-${Date.now()}`,
          role: "assistant",
          content: replyText || "（空回复）",
          meta: metaParts.length ? metaParts.join(" · ") : undefined,
        },
      ]);
      setSessions((prev) => {
        const title = text.length > 40 ? `${text.slice(0, 40)}…` : text;
        const mapped = prev.map((s) =>
          s.id === sid
            ? { ...s, title: s.title === "新对话" ? title : s.title, message_count: (s.message_count || 0) + 2 }
            : s,
        );
        return mapped.sort((a, b) => (a.id === sid ? -1 : b.id === sid ? 1 : 0));
      });
      pushTimeline("完成");
    } catch (e) {
      if ((e as Error)?.name === "AbortError") return;
      message.error(e instanceof Error ? e.message : extractApiErrorMessage(e, "对话失败"));
    } finally {
      setSending(false);
    }
  }

  async function clearChat() {
    if (!sessionId) {
      setMessages([]);
      setLastSteps([]);
      setLastRag([]);
      return;
    }
    try {
      await clearAISession(sessionId);
      setMessages([]);
      setLastSteps([]);
      setLastRag([]);
      setSessions((prev) => prev.map((s) => (s.id === sessionId ? { ...s, title: "新对话", message_count: 0 } : s)));
    } catch (e) {
      message.error(extractApiErrorMessage(e, "清空失败"));
    }
  }

  return (
    <div style={{ display: "flex", gap: 12, alignItems: "stretch" }}>
      <Card
        className="table-card"
        size="small"
        title="会话"
        extra={
          <Button size="small" type="link" icon={<PlusOutlined />} onClick={() => void handleNewSession()}>
            新建
          </Button>
        }
        style={{ width: 260, flexShrink: 0 }}
        styles={{ body: { padding: 0, maxHeight: 720, overflow: "auto" } }}
      >
        <List
          loading={sessionsLoading}
          dataSource={sessions}
          locale={{ emptyText: "暂无会话" }}
          renderItem={(item) => (
            <List.Item
              style={{
                padding: "8px 12px",
                cursor: "pointer",
                background: item.id === sessionId ? "var(--ant-color-primary-bg, #e6f4ff)" : undefined,
              }}
              onClick={() => void openSession(item.id)}
              actions={[
                <Button
                  key="del"
                  type="text"
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  onClick={(e) => {
                    e.stopPropagation();
                    void handleDeleteSession(item.id);
                  }}
                />,
              ]}
            >
              <List.Item.Meta
                title={
                  <Typography.Text ellipsis style={{ maxWidth: 160 }}>
                    {item.title || `会话 #${item.id}`}
                  </Typography.Text>
                }
                description={
                  <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                    {item.message_count ? `${item.message_count} 条` : "空"}
                  </Typography.Text>
                }
              />
            </List.Item>
          )}
        />
      </Card>

      <Card className="table-card" style={{ flex: 1, minWidth: 0 }}>
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
              {sessionId ? <Tag>会话 #{sessionId}</Tag> : null}
              <Link to="/ai/investigations">调查记录</Link>
              <Link to="/ai/approvals">操作审批</Link>
            </Space>
            <Space wrap>
              <Select
                style={{ minWidth: 260 }}
                placeholder="选择模型 Provider"
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
                onClick={() => void clearChat()}
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
              message="先选上下文，再问事实问题"
              description="查 Pod/事件请选集群；查日志/构建/告警请选项目。助手会优先调工具取证，知识库只作排查思路参考。写操作需开启「写工具」并走审批。"
            />
          )}

          <Space wrap>
            <span>工具</span>
            <Switch checked={enableTools} onChange={setEnableTools} />
            <span>写工具(审批)</span>
            <Switch checked={enableWrite} onChange={setEnableWrite} disabled={!enableTools} />
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder="选择项目（日志/CI/告警）"
              style={{ minWidth: 200 }}
              value={projectId}
              options={projectOptions}
              onChange={(v) => setProjectId(typeof v === "number" ? v : undefined)}
            />
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder="选择集群（K8s）"
              style={{ minWidth: 200 }}
              value={clusterId}
              options={clusterOptions}
              onChange={(v) => setClusterId(typeof v === "number" ? v : undefined)}
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
                示例：某命名空间 Pod CrashLoop；某次构建失败原因；告警为什么没收到。请先选好项目/集群，并在问题中给出 namespace 与资源名。
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
                {sending ? (
                  <div>
                    <Typography.Text type="secondary">思考中…</Typography.Text>
                    {liveTimeline.length > 0 ? (
                      <pre
                        style={{
                          margin: "8px 0 0",
                          padding: 8,
                          fontSize: 12,
                          maxHeight: 120,
                          overflow: "auto",
                          background: "rgba(0,0,0,0.04)",
                          borderRadius: 6,
                          whiteSpace: "pre-wrap",
                        }}
                      >
                        {liveTimeline.join("\n")}
                      </pre>
                    ) : null}
                  </div>
                ) : null}
              </Space>
            )}
          </div>

          {!sending && liveTimeline.length > 0 ? (
            <Card size="small" title="执行轨迹" extra={<Button type="link" size="small" onClick={() => setLiveTimeline([])}>清除</Button>}>
              <pre style={{ margin: 0, fontSize: 12, maxHeight: 140, overflow: "auto", whiteSpace: "pre-wrap" }}>
                {liveTimeline.join("\n")}
              </pre>
            </Card>
          ) : null}

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
          {lastRag && lastRag.length > 0 ? (
            <Card size="small" title="本次知识库命中">
              <Space direction="vertical" style={{ width: "100%" }}>
                {lastRag.map((h, i) => (
                  <Typography.Paragraph key={`${h.source}-${i}`} style={{ marginBottom: 4, fontSize: 12 }}>
                    <Tag>{h.module || "doc"}</Tag>
                    {h.source}
                  </Typography.Paragraph>
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
