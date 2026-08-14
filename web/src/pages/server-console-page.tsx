import { PlayCircleOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Form, Input, Space, Table, Tabs, Tag, Typography, Upload, message } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  deleteProjectServerFile,
  downloadProjectServerFile,
  execProjectServerCommand,
  getProjectServerDetail,
  listProjectServerFiles,
  uploadProjectServerFile,
  type ServerDetailItem,
  type ServerRemoteFileItem,
} from "../services/projects";
import { getMyServerAccess } from "../services/project-resource-grants";
import { extractApiErrorMessage } from "../services/http";
import { openAuthenticatedWebSocket } from "../services/ws-auth";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";

type ExecForm = {
  command: string;
  timeout_sec?: number;
};

type ExecResult = {
  stdout: string;
  stderr: string;
  exit_code: number;
  duration_ms: number;
  truncated: boolean;
};

type TerminalFrame = {
  type?: string;
  data?: string;
};

export function ServerConsolePage() {
  const [searchParams] = useSearchParams();
  const projectId = Number(searchParams.get("project_id") || 0);
  const serverId = Number(searchParams.get("server_id") || 0);

  const [server, setServer] = useState<ServerDetailItem | null>(null);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<ExecResult | null>(null);
  const [terminalConnected, setTerminalConnected] = useState(false);
  const [terminalConnecting, setTerminalConnecting] = useState(false);
  const [canExec, setCanExec] = useState(false);
  const [accessLoaded, setAccessLoaded] = useState(false);
  const [remotePath, setRemotePath] = useState("/");
  const [fileList, setFileList] = useState<ServerRemoteFileItem[]>([]);
  const [filesLoading, setFilesLoading] = useState(false);
  const [maxTransferMB, setMaxTransferMB] = useState(50);
  const [uploading, setUploading] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const termBoxRef = useRef<HTMLDivElement | null>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const dataDisposableRef = useRef<{ dispose: () => void } | null>(null);
  const [form] = Form.useForm<ExecForm>();

  const validParams = useMemo(() => projectId > 0 && serverId > 0, [projectId, serverId]);

  useEffect(() => {
    if (!validParams) return;
    void loadDetail();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [validParams, projectId, serverId]);

  async function loadDetail() {
    setLoading(true);
    setAccessLoaded(false);
    setCanExec(false);
    try {
      const [data, access] = await Promise.all([
        getProjectServerDetail(projectId, serverId),
        getMyServerAccess(projectId, serverId).catch(() => ({ can_view: false, can_exec: false, can_manage: false })),
      ]);
      setServer(data);
      setCanExec(Boolean(access.can_exec || access.can_manage));
    } finally {
      setAccessLoaded(true);
      setLoading(false);
    }
  }

  async function runCommand() {
    if (!validParams) return;
    if (!accessLoaded || !canExec) {
      message.error("仅有查看权限，不能执行命令");
      return;
    }
    const values = await form.validateFields();
    setRunning(true);
    try {
      const res = await execProjectServerCommand(projectId, serverId, {
        command: values.command,
        timeout_sec: values.timeout_sec ?? 20,
      });
      setResult(res);
      if (res.exit_code === 0) {
        message.success("命令执行成功");
      } else {
        message.warning(`命令执行完成，退出码 ${res.exit_code}`);
      }
    } finally {
      setRunning(false);
    }
  }

  async function loadFiles(path = remotePath) {
    if (!validParams || !canExec) return;
    setFilesLoading(true);
    try {
      const res = await listProjectServerFiles(projectId, serverId, path || "/");
      setFileList(res?.list || []);
      setRemotePath(res?.path || path || "/");
      if (res?.max_transfer_mb) setMaxTransferMB(res.max_transfer_mb);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "列出远端文件失败"));
    } finally {
      setFilesLoading(false);
    }
  }

  function parentPath(p: string) {
    const clean = (p || "/").replace(/\\/g, "/").replace(/\/+$/, "") || "/";
    if (clean === "/") return "/";
    const idx = clean.lastIndexOf("/");
    return idx <= 0 ? "/" : clean.slice(0, idx) || "/";
  }

  async function handleDownload(row: ServerRemoteFileItem) {
    try {
      const blob = await downloadProjectServerFile(projectId, serverId, row.path);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = row.name || "download.bin";
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "下载失败"));
    }
  }

  async function handleDelete(row: ServerRemoteFileItem) {
    try {
      await deleteProjectServerFile(projectId, serverId, row.path);
      message.success("已删除");
      await loadFiles(remotePath);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "删除失败"));
    }
  }

  function appendTerminalText(text: string) {
    if (!xtermRef.current) return;
    xtermRef.current.write(text);
  }

  async function openTerminal() {
    if (!validParams) return;
    if (!accessLoaded || !canExec) {
      message.error("仅有查看权限，不能连接 SSH 终端");
      return;
    }
    if (!xtermRef.current) {
      message.warning("终端正在初始化，请稍后再试");
      return;
    }
    if (wsRef.current && (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING)) {
      return;
    }

    appendTerminalText("\r\n[connecting]\r\n");
    setTerminalConnecting(true);

    let ws: WebSocket;
    try {
      ws = await openAuthenticatedWebSocket(
        `/api/v1/projects/${projectId}/servers/${serverId}/terminal/ws`,
        {},
        "server-terminal",
      );
    } catch (error) {
      setTerminalConnecting(false);
      const reason = error instanceof Error ? error.message : "unknown error";
      appendTerminalText(`\r\n[websocket init failed] ${reason}\r\n`);
      message.error("终端连接初始化失败");
      return;
    }

    wsRef.current = ws;

    ws.onopen = () => {
      setTerminalConnecting(false);
      setTerminalConnected(true);
      appendTerminalText("\r\n[connected]\r\n");
      fitAddonRef.current?.fit();
      xtermRef.current?.focus();
      const cols = Math.max(80, xtermRef.current?.cols ?? 120);
      const rows = Math.max(24, xtermRef.current?.rows ?? 40);
      ws.send(JSON.stringify({ type: "resize", cols, rows }));
    };

    ws.onmessage = (ev) => {
      try {
        const payload = JSON.parse(String(ev.data)) as TerminalFrame;
        if (payload.type === "stdout" && typeof payload.data === "string") {
          appendTerminalText(payload.data);
          return;
        }
        if (payload.type === "error" && payload.data) {
          appendTerminalText(`\r\n[error] ${payload.data}\r\n`);
          return;
        }
        if (payload.type === "exit") {
          appendTerminalText("\r\n[session exited]\r\n");
          setTerminalConnecting(false);
          setTerminalConnected(false);
          return;
        }
      } catch {
        appendTerminalText(String(ev.data));
      }
    };

    ws.onerror = () => {
      setTerminalConnecting(false);
      appendTerminalText("\r\n[websocket error]\r\n");
      message.error("终端 WebSocket 连接失败");
    };

    ws.onclose = (ev) => {
      setTerminalConnecting(false);
      setTerminalConnected(false);
      wsRef.current = null;
      appendTerminalText(`\r\n[disconnected code=${ev.code}${ev.reason ? ` reason=${ev.reason}` : ""}]\r\n`);
    };
  }

  function closeTerminal() {
    const ws = wsRef.current;
    if (!ws) return;
    try {
      ws.send(JSON.stringify({ type: "close" }));
    } catch {
      // ignore
    }
    ws.close();
    wsRef.current = null;
    setTerminalConnecting(false);
    setTerminalConnected(false);
  }

  function sendTerminalInput(text: string) {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({ type: "input", data: text }));
  }

  useEffect(() => {
    if (xtermRef.current || !termBoxRef.current) return;

    const term = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontFamily: "Consolas, Menlo, Monaco, monospace",
      fontSize: 13,
      lineHeight: 1.25,
      theme: {
        background: "#0b1220",
        foreground: "#d7e3ff",
      },
      scrollback: 5000,
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(termBoxRef.current);
    fitAddon.fit();
    term.focus();
    term.writeln("Ready. Click '连接终端' to start.");

    dataDisposableRef.current = term.onData((data) => {
      sendTerminalInput(data);
    });

    term.attachCustomKeyEventHandler((ev) => {
      if ((ev.ctrlKey || ev.metaKey) && ev.shiftKey && ev.type === "keydown") {
        const key = ev.key.toLowerCase();
        if (key === "c") {
          const selected = term.getSelection();
          if (selected) {
            void navigator.clipboard?.writeText(selected);
          } else {
            sendTerminalInput("\u0003");
          }
          return false;
        }
        if (key === "v") {
          void navigator.clipboard?.readText().then((txt) => {
            if (txt) {
              sendTerminalInput(txt);
            }
          });
          return false;
        }
      }
      return true;
    });

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    const resizeObs = new ResizeObserver(() => {
      fitAddon.fit();
      const ws = wsRef.current;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
      }
    });
    resizeObs.observe(termBoxRef.current);

    return () => {
      resizeObs.disconnect();
      dataDisposableRef.current?.dispose();
      dataDisposableRef.current = null;
      xtermRef.current?.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
    };
  }, []);

  useEffect(() => {
    return () => {
      closeTerminal();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!validParams) {
    return (
      <Card className="table-card" title="服务器控制台">
        <Alert type="error" showIcon message="参数不完整" description="请从服务器管理页面点击“连接”进入。" />
      </Card>
    );
  }

  return (
    <Space direction="vertical" size={12} style={{ width: "100%" }}>
      <Card className="table-card" loading={loading} title="服务器控制台" extra={<Link to="/project-servers">返回服务器管理</Link>}>
        <Space wrap>
          <Tag color="blue">Project #{projectId}</Tag>
          <Tag>Server #{serverId}</Tag>
          <Tag>{server?.source_type === "cloud" ? `云 · ${server?.provider || "-"}` : "自建"}</Tag>
          <Tag>{server?.host || "-"}</Tag>
          <Tag>{server?.auth_type || "-"}</Tag>
        </Space>
      </Card>

      <Card className="table-card" bodyStyle={{ paddingTop: 8 }}>
        {!accessLoaded ? (
          <Alert type="info" showIcon style={{ marginBottom: 12 }} message="正在校验服务器访问权限…" />
        ) : !canExec ? (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 12 }}
            message="当前账号对该服务器仅有查看权限"
            description="交互式终端、单次命令与文件传输需要「SSH/执行」授权。请联系项目 owner/admin 在「项目成员 → 资源授权」中勾选 SSH/执行。"
          />
        ) : null}
        <Tabs
          onChange={(key) => {
            if (key === "files" && canExec) void loadFiles(remotePath || "/");
          }}
          items={[
            {
              key: "terminal",
              label: "交互式终端（WebSocket）",
              children: (
                <Space direction="vertical" size={12} style={{ width: "100%" }}>
                  <Space wrap>
                    <Button
                      type="primary"
                      onClick={openTerminal}
                      disabled={!accessLoaded || !canExec || terminalConnected || terminalConnecting}
                      loading={terminalConnecting}
                    >
                      连接终端
                    </Button>
                    <Button onClick={closeTerminal} disabled={!terminalConnected}>
                      断开
                    </Button>
                    <Button onClick={() => sendTerminalInput("\u0003")} disabled={!terminalConnected}>
                      发送 Ctrl+C
                    </Button>
                    <Button onClick={() => xtermRef.current?.clear()}>清屏</Button>
                    <Tag color={terminalConnected ? "success" : terminalConnecting ? "processing" : "default"}>
                      {terminalConnected ? "已连接" : terminalConnecting ? "连接中" : "未连接"}
                    </Tag>
                    <Tag>快捷键: Ctrl+Shift+C 复制/中断, Ctrl+Shift+V 粘贴</Tag>
                  </Space>
                  <div
                    ref={termBoxRef}
                    style={{ minHeight: 340, maxHeight: 520, overflow: "hidden", borderRadius: 10, padding: 8, background: "#0b1220" }}
                  />
                </Space>
              ),
            },
            {
              key: "oneshot",
              label: "单次命令执行",
              children: (
                <Space direction="vertical" size={12} style={{ width: "100%" }}>
                  <Form form={form} layout="vertical" initialValues={{ command: "uname -a", timeout_sec: 20 }} onFinish={() => void runCommand()}>
                    <Form.Item label="命令" name="command" rules={[{ required: true, message: "请输入要执行的命令" }]}>
                      <Input.TextArea rows={4} placeholder="例如: uname -a && whoami" />
                    </Form.Item>
                    <Form.Item label="超时时间（秒）" name="timeout_sec">
                      <Input type="number" min={1} max={120} style={{ width: 180 }} />
                    </Form.Item>
                    <Space>
                      <Button
                        type="primary"
                        icon={<PlayCircleOutlined />}
                        onClick={() => void runCommand()}
                        loading={running}
                        disabled={!accessLoaded || !canExec}
                      >
                        执行
                      </Button>
                      <Button icon={<ReloadOutlined />} onClick={() => setResult(null)}>
                        清空结果
                      </Button>
                    </Space>
                  </Form>

                  <Card size="small" title="执行结果">
                    {result ? (
                      <Space direction="vertical" size={10} style={{ width: "100%" }}>
                        <Space>
                          <Tag color={result.exit_code === 0 ? "success" : "error"}>退出码 {result.exit_code}</Tag>
                          <Tag>耗时 {result.duration_ms} ms</Tag>
                          {result.truncated ? <Tag color="warning">输出已截断</Tag> : null}
                        </Space>
                        <Typography.Text strong>STDOUT</Typography.Text>
                        <pre style={{ margin: 0, padding: 12, borderRadius: 10, background: "#0b1220", color: "#d7e3ff", maxHeight: 280, overflow: "auto" }}>{result.stdout || "(empty)"}</pre>
                        <Typography.Text strong>STDERR</Typography.Text>
                        <pre style={{ margin: 0, padding: 12, borderRadius: 10, background: "#0b1220", color: "#ffd5d5", maxHeight: 220, overflow: "auto" }}>{result.stderr || "(empty)"}</pre>
                      </Space>
                    ) : (
                      <Typography.Text type="secondary">暂无执行结果。</Typography.Text>
                    )}
                  </Card>
                </Space>
              ),
            },
            {
              key: "files",
              label: "文件传输",
              children: (
                <Space direction="vertical" size={12} style={{ width: "100%" }}>
                  <Alert
                    type="info"
                    showIcon
                    message={`单文件上限 ${maxTransferMB} MB（数据字典 cmdb_max_transfer_file_mb）`}
                    description="通过 SFTP 浏览/上传/下载/删除；需要服务器 SSH 执行权限。"
                  />
                  <Space wrap style={{ width: "100%" }}>
                    <Input
                      value={remotePath}
                      onChange={(e) => setRemotePath(e.target.value)}
                      onPressEnter={() => void loadFiles(remotePath)}
                      style={{ minWidth: 320, flex: 1 }}
                      placeholder="/tmp"
                      disabled={!canExec}
                    />
                    <Button icon={<ReloadOutlined />} onClick={() => void loadFiles(remotePath)} loading={filesLoading} disabled={!canExec}>
                      刷新
                    </Button>
                    <Button onClick={() => void loadFiles(parentPath(remotePath))} disabled={!canExec || remotePath === "/"}>
                      上级
                    </Button>
                    <Upload
                      showUploadList={false}
                      disabled={!canExec || uploading}
                      beforeUpload={async (file) => {
                        setUploading(true);
                        try {
                          await uploadProjectServerFile(projectId, serverId, remotePath || "/", file);
                          message.success("上传成功");
                          await loadFiles(remotePath);
                        } catch (e) {
                          message.error(extractApiErrorMessage(e, "上传失败"));
                        } finally {
                          setUploading(false);
                        }
                        return false;
                      }}
                    >
                      <Button type="primary" icon={<UploadOutlined />} loading={uploading} disabled={!canExec}>
                        上传到当前目录
                      </Button>
                    </Upload>
                  </Space>
                  <Table
                    rowKey="path"
                    size="small"
                    loading={filesLoading}
                    dataSource={fileList}
                    pagination={{ pageSize: 50 }}
                    columns={[
                      {
                        title: "名称",
                        dataIndex: "name",
                        render: (_, row) =>
                          row.is_dir ? (
                            <Button type="link" style={{ padding: 0 }} onClick={() => void loadFiles(row.path)}>
                              {row.name}/
                            </Button>
                          ) : (
                            row.name
                          ),
                      },
                      {
                        title: "大小",
                        dataIndex: "size",
                        width: 120,
                        render: (v, row) => (row.is_dir ? "-" : `${v}`),
                      },
                      { title: "权限", dataIndex: "mode", width: 140 },
                      { title: "修改时间", dataIndex: "mod_time", width: 200 },
                      {
                        title: "操作",
                        width: 160,
                        render: (_, row) =>
                          row.is_dir ? null : (
                            <Space size="small">
                              <Button type="link" size="small" onClick={() => void handleDownload(row)}>
                                下载
                              </Button>
                              <Button type="link" size="small" danger onClick={() => void handleDelete(row)}>
                                删除
                              </Button>
                            </Space>
                          ),
                      },
                    ]}
                  />
                </Space>
              ),
            },
          ]}
        />
      </Card>
    </Space>
  );
}
