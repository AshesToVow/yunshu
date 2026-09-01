// @ts-nocheck
import { useEffect, useRef, type RefObject } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { getPodDetail, type PodDetail, type PodItem } from "../../services/pods";
import { openAuthenticatedWebSocket } from "../../services/ws-auth";

export type UsePodExecParams = {
  execOpen: boolean;
  clusterId?: number;
  selected: PodItem | null;
  detail: PodDetail | null;
  execCommand: string;
  execTermHostRef: RefObject<HTMLDivElement | null>;
};

/** Pod Exec：xterm + Cookie 会话换 ticket 的 WebSocket。 */
export function usePodExec({
  execOpen,
  clusterId,
  selected,
  detail,
  execCommand,
  execTermHostRef,
}: UsePodExecParams) {
  const execTermRef = useRef<Terminal | null>(null);
  const execFitRef = useRef<FitAddon | null>(null);
  const execWsRef = useRef<WebSocket | null>(null);
  const execCommandRef = useRef(execCommand);
  execCommandRef.current = execCommand;

  function closeExecSocket() {
    try {
      execWsRef.current?.close();
    } catch {
      // ignore
    }
    execWsRef.current = null;
  }

  useEffect(() => {
    if (!execOpen) return;
    if (!clusterId || !selected) return;
    const host = execTermHostRef.current;
    if (!host) return;

    let disposed = false;
    let removeWindowResize: (() => void) | undefined;
    let onDataDispose: { dispose: () => void } | undefined;
    let onResizeDispose: { dispose: () => void } | undefined;

    host.innerHTML = "";

    const term = new Terminal({
      cursorBlink: true,
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      fontSize: 12,
      theme: { background: "#141414" },
      scrollback: 5000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);
    fit.fit();
    term.focus();

    execTermRef.current = term;
    execFitRef.current = fit;

    void (async () => {
      let container: string | undefined;
      if (detail?.namespace === selected.namespace && detail.name === selected.name) {
        container = detail.containers?.[0]?.name;
      } else {
        try {
          const d = await getPodDetail({
            cluster_id: clusterId,
            namespace: selected.namespace,
            name: selected.name,
          });
          container = d.containers?.[0]?.name;
        } catch {
          // 后端会解析默认容器；此处失败时不阻塞 Exec 连接
        }
      }

      let ws: WebSocket;
      try {
        ws = await openAuthenticatedWebSocket(
          "/api/v1/pods/exec/ws",
          {
            cluster_id: clusterId,
            namespace: selected.namespace,
            name: selected.name,
            ...(container ? { container } : {}),
          },
          "pod-exec",
        );
      } catch {
        if (!disposed) term.writeln("\r\n[connection error: ticket failed]\r\n");
        return;
      }
      if (disposed) {
        ws.close();
        return;
      }
      execWsRef.current = ws;

      const sendResize = () => {
        try {
          const cols = term.cols;
          const rows = term.rows;
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "resize", cols, rows }));
          }
        } catch {
          // ignore
        }
      };

      onDataDispose = term.onData((data) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify({ type: "input", data }));
      });
      onResizeDispose = term.onResize(({ cols, rows }) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        ws.send(JSON.stringify({ type: "resize", cols, rows }));
      });

      ws.onopen = () => {
        term.writeln(`Connected: ${selected.namespace}/${selected.name}`);
        sendResize();
        ws.send(JSON.stringify({ type: "input", data: `${execCommandRef.current}\n` }));
      };
      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(String(ev.data));
          if (msg.type === "stdout" && typeof msg.data === "string") {
            term.write(msg.data);
          } else if (msg.type === "error") {
            term.writeln(`\r\n[error] ${msg.data || "unknown"}`);
          } else if (msg.type === "exit") {
            term.writeln("\r\n[disconnected]");
          }
        } catch {
          term.write(String(ev.data));
        }
      };
      ws.onclose = () => {
        term.writeln("\r\n[connection closed]");
      };
      ws.onerror = () => {
        term.writeln("\r\n[connection error]");
      };

      const onWindowResize = () => {
        try {
          fit.fit();
          sendResize();
        } catch {
          // ignore
        }
      };
      window.addEventListener("resize", onWindowResize);
      removeWindowResize = () => window.removeEventListener("resize", onWindowResize);
    })();

    return () => {
      disposed = true;
      removeWindowResize?.();
      try {
        onDataDispose?.dispose();
        onResizeDispose?.dispose();
      } catch {
        // ignore
      }
      closeExecSocket();
      try {
        term.dispose();
      } catch {
        // ignore
      }
      execTermRef.current = null;
      execFitRef.current = null;
    };
  }, [
    execOpen,
    clusterId,
    selected?.namespace,
    selected?.name,
    detail?.namespace,
    detail?.name,
    detail?.containers,
    execTermHostRef,
  ]);

  return { closeExecSocket, execTermRef };
}
