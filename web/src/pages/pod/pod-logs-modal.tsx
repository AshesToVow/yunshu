/**
 * Pod 日志高级筛选对话框（RF-04 拆分产物）
 * 支持标题栏最大化/还原，以及右下角拖拽调整尺寸（运维控制台常见交互）。
 */
import { useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent } from "react";
import {
  CompressOutlined,
  DownloadOutlined,
  ExpandOutlined,
  FileSearchOutlined,
} from "@ant-design/icons";
import { Button, Checkbox, Input, Modal, Select, Space, Switch, Tooltip, Typography } from "antd";
import type { PodItem } from "../../services/pods";

const DEFAULT_WIDTH = 980;
const DEFAULT_BODY_HEIGHT = 480;
const MIN_WIDTH = 640;
const MIN_BODY_HEIGHT = 240;

export type PodLogsModalProps = {
  logsOpen: boolean;
  logsTitle: string;
  logsLoading: boolean;
  logsText: string;
  logsKeyword: string;
  logsStartTime: string;
  logsEndTime: string;
  logsPrevious: boolean;
  logsTimestamps: boolean;
  logsSinceSeconds: number | undefined;
  logsSinceTime: string;
  logsContainer: string | undefined;
  logContainerOptions: string[];
  streaming: boolean;
  selected: PodItem | null;
  setLogsOpen: (v: boolean) => void;
  setLogsKeyword: (v: string) => void;
  setLogsStartTime: (v: string) => void;
  setLogsEndTime: (v: string) => void;
  setLogsPrevious: (v: boolean) => void;
  setLogsTimestamps: (v: boolean) => void;
  setLogsSinceSeconds: (v: number | undefined) => void;
  setLogsSinceTime: (v: string) => void;
  setLogsContainer: (v: string | undefined) => void;
  stopLogStream: () => void;
  startLogStream: () => void | Promise<void>;
  handleFilterLogs: () => void | Promise<void>;
  handleDownloadLogs: () => void | Promise<void>;
  handleViewLogs: (pod: PodItem, mode?: "inline" | "modal") => void | Promise<void>;
};

export function PodLogsModal({
  logsOpen,
  logsTitle,
  logsLoading,
  logsText,
  logsKeyword,
  logsStartTime,
  logsEndTime,
  logsPrevious,
  logsTimestamps,
  logsSinceSeconds,
  logsSinceTime,
  logsContainer,
  logContainerOptions,
  streaming,
  selected,
  setLogsOpen,
  setLogsKeyword,
  setLogsStartTime,
  setLogsEndTime,
  setLogsPrevious,
  setLogsTimestamps,
  setLogsSinceSeconds,
  setLogsSinceTime,
  setLogsContainer,
  stopLogStream,
  startLogStream,
  handleFilterLogs,
  handleDownloadLogs,
  handleViewLogs,
}: PodLogsModalProps) {
  const [maximized, setMaximized] = useState(false);
  const [width, setWidth] = useState(DEFAULT_WIDTH);
  const [bodyHeight, setBodyHeight] = useState(DEFAULT_BODY_HEIGHT);
  const resizeRef = useRef<{
    startX: number;
    startY: number;
    startW: number;
    startH: number;
  } | null>(null);

  useEffect(() => {
    if (!logsOpen) {
      setMaximized(false);
    }
  }, [logsOpen]);

  const toggleMaximize = useCallback(() => {
    setMaximized((v) => !v);
  }, []);

  const onResizePointerDown = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (maximized) return;
    e.preventDefault();
    e.stopPropagation();
    const el = e.currentTarget;
    el.setPointerCapture(e.pointerId);
    resizeRef.current = {
      startX: e.clientX,
      startY: e.clientY,
      startW: width,
      startH: bodyHeight,
    };
  };

  const onResizePointerMove = (e: ReactPointerEvent<HTMLDivElement>) => {
    const state = resizeRef.current;
    if (!state) return;
    const maxW = Math.max(MIN_WIDTH, window.innerWidth - 48);
    const maxH = Math.max(MIN_BODY_HEIGHT, window.innerHeight - 200);
    const nextW = Math.min(maxW, Math.max(MIN_WIDTH, state.startW + (e.clientX - state.startX)));
    const nextH = Math.min(maxH, Math.max(MIN_BODY_HEIGHT, state.startH + (e.clientY - state.startY)));
    setWidth(nextW);
    setBodyHeight(nextH);
  };

  const onResizePointerUp = (e: ReactPointerEvent<HTMLDivElement>) => {
    if (!resizeRef.current) return;
    resizeRef.current = null;
    try {
      e.currentTarget.releasePointerCapture(e.pointerId);
    } catch {
      // ignore
    }
  };

  const modalWidth = maximized ? "100vw" : width;
  const contentHeight = maximized ? "calc(100vh - 168px)" : bodyHeight;

  return (
    <Modal
      title={
        <div className="pod-logs-modal__title">
          <span className="pod-logs-modal__title-text" title={logsTitle}>
            Pod 日志 - {logsTitle}
          </span>
          <Tooltip title={maximized ? "还原" : "最大化"}>
            <Button
              type="text"
              size="small"
              className="pod-logs-modal__max-btn"
              aria-label={maximized ? "还原" : "最大化"}
              icon={maximized ? <CompressOutlined /> : <ExpandOutlined />}
              onClick={toggleMaximize}
            />
          </Tooltip>
        </div>
      }
      open={logsOpen}
      onCancel={() => {
        stopLogStream();
        setLogsOpen(false);
      }}
      footer={null}
      width={modalWidth}
      centered={!maximized}
      destroyOnClose={false}
      wrapClassName={`pod-logs-modal-wrap${maximized ? " pod-logs-modal-wrap--maximized" : ""}`}
      styles={{
        body: {
          height: contentHeight,
          maxHeight: maximized ? "none" : undefined,
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
          position: "relative",
          paddingBottom: maximized ? 16 : 28,
        },
      }}
    >
      <Space wrap className="pod-logs-modal__toolbar">
        {logContainerOptions.length > 1 ? (
          <Select
            allowClear
            placeholder="容器"
            style={{ width: 140 }}
            value={logsContainer}
            options={logContainerOptions.map((n) => ({ label: n, value: n }))}
            onChange={(v) => setLogsContainer(v)}
          />
        ) : null}
        <Select
          allowClear
          placeholder="最近时间"
          style={{ width: 130 }}
          value={logsSinceSeconds}
          options={[
            { label: "5 分钟", value: 300 },
            { label: "1 小时", value: 3600 },
            { label: "6 小时", value: 21600 },
            { label: "24 小时", value: 86400 },
          ]}
          onChange={(v) => setLogsSinceSeconds(v)}
        />
        <Input
          placeholder="since-time RFC3339"
          value={logsSinceTime}
          onChange={(e) => setLogsSinceTime(e.target.value)}
          style={{ width: 200 }}
        />
        <Checkbox checked={logsPrevious} onChange={(e) => setLogsPrevious(e.target.checked)}>
          上一实例
        </Checkbox>
        <Switch
          size="small"
          checked={logsTimestamps}
          onChange={setLogsTimestamps}
          checkedChildren="时间戳"
          unCheckedChildren="时间戳"
        />
        <Input
          placeholder="关键字过滤"
          value={logsKeyword}
          onChange={(e) => setLogsKeyword(e.target.value)}
          style={{ width: 160 }}
        />
        <Input
          placeholder="开始时间 2026-01-02 15:04:05"
          value={logsStartTime}
          onChange={(e) => setLogsStartTime(e.target.value)}
          style={{ width: 210 }}
        />
        <Input
          placeholder="结束时间 2026-01-02 15:04:05"
          value={logsEndTime}
          onChange={(e) => setLogsEndTime(e.target.value)}
          style={{ width: 210 }}
        />
        <Button icon={<FileSearchOutlined />} onClick={() => void handleFilterLogs()}>
          拉取/过滤
        </Button>
        <Button icon={<DownloadOutlined />} onClick={() => void handleDownloadLogs()}>
          下载
        </Button>
        <Button type={streaming ? "default" : "primary"} onClick={() => void startLogStream()} disabled={streaming}>
          开始实时流
        </Button>
        <Button danger={streaming} onClick={stopLogStream} disabled={!streaming}>
          停止流
        </Button>
        <Button onClick={() => selected && void handleViewLogs(selected)}>获取当前日志</Button>
      </Space>

      <div className="pod-logs-modal__body">
        {logsLoading ? (
          <Typography.Text>日志加载中...</Typography.Text>
        ) : (
          <pre className="code-block-panel pod-logs-modal__pre">{logsText || "暂无日志"}</pre>
        )}
      </div>

      {!maximized ? (
        <div
          className="pod-logs-modal__resize-handle"
          title="拖拽调整大小"
          onPointerDown={onResizePointerDown}
          onPointerMove={onResizePointerMove}
          onPointerUp={onResizePointerUp}
          onPointerCancel={onResizePointerUp}
        />
      ) : null}
    </Modal>
  );
}
