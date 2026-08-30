/**
 * Pod 日志高级筛选对话框（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX；state/handler 仍由页面持有，经同名 props 传入。
 */
import { DownloadOutlined, FileSearchOutlined } from "@ant-design/icons";
import { Button, Checkbox, Input, Modal, Select, Space, Switch, Typography } from "antd";
import type { PodItem } from "../../services/pods";

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
  return (
        <Modal
          title={`Pod 日志 - ${logsTitle}`}
          open={logsOpen}
          onCancel={() => {
            stopLogStream();
            setLogsOpen(false);
          }}
          footer={null}
          width={980}
        >
          <Space wrap style={{ marginBottom: 12 }}>
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
            <Input placeholder="since-time RFC3339" value={logsSinceTime} onChange={(e) => setLogsSinceTime(e.target.value)} style={{ width: 200 }} />
            <Checkbox checked={logsPrevious} onChange={(e) => setLogsPrevious(e.target.checked)}>上一实例</Checkbox>
            <Switch size="small" checked={logsTimestamps} onChange={setLogsTimestamps} checkedChildren="时间戳" unCheckedChildren="时间戳" />
            <Input placeholder="关键字过滤" value={logsKeyword} onChange={(e) => setLogsKeyword(e.target.value)} style={{ width: 160 }} />
            <Input placeholder="开始时间 2026-01-02 15:04:05" value={logsStartTime} onChange={(e) => setLogsStartTime(e.target.value)} style={{ width: 210 }} />
            <Input placeholder="结束时间 2026-01-02 15:04:05" value={logsEndTime} onChange={(e) => setLogsEndTime(e.target.value)} style={{ width: 210 }} />
            <Button icon={<FileSearchOutlined />} onClick={() => void handleFilterLogs()}>拉取/过滤</Button>
            <Button icon={<DownloadOutlined />} onClick={() => void handleDownloadLogs()}>下载</Button>
            <Button type={streaming ? "default" : "primary"} onClick={() => void startLogStream()} disabled={streaming}>开始实时流</Button>
            <Button danger={streaming} onClick={stopLogStream} disabled={!streaming}>停止流</Button>
            <Button onClick={() => selected && void handleViewLogs(selected)}>获取当前日志</Button>
          </Space>
          {logsLoading ? (
            <Typography.Text>日志加载中...</Typography.Text>
          ) : (
            <pre className="code-block-panel">{logsText || "暂无日志"}</pre>
          )}
        </Modal>
  );
}
