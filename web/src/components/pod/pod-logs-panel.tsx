import { DownloadOutlined, FileSearchOutlined } from "@ant-design/icons";
import { Button, Select, Space, Spin, Typography } from "antd";

export type PodLogsPanelProps = {
  loading?: boolean;
  streaming?: boolean;
  logsText: string;
  containerOptions: string[];
  container?: string;
  onContainerChange: (value?: string) => void;
  onFetch: () => void;
  onDownload?: () => void;
  onStartStream?: () => void;
  onStopStream?: () => void;
};

export function PodLogsPanel({
  loading,
  streaming,
  logsText,
  containerOptions,
  container,
  onContainerChange,
  onFetch,
  onDownload,
  onStartStream,
  onStopStream,
}: PodLogsPanelProps) {
  return (
    <div className="pod-logs-panel">
      <Space wrap size={[8, 8]} style={{ marginBottom: 8 }}>
        {containerOptions.length > 1 ? (
          <Select
            allowClear
            placeholder="容器"
            size="small"
            style={{ width: 140 }}
            value={container}
            options={containerOptions.map((n) => ({ label: n, value: n }))}
            onChange={onContainerChange}
          />
        ) : null}
        <Button size="small" icon={<FileSearchOutlined />} loading={loading && !streaming} onClick={onFetch}>
          拉取日志
        </Button>
        {onDownload ? (
          <Button size="small" icon={<DownloadOutlined />} onClick={onDownload}>
            下载
          </Button>
        ) : null}
        {onStartStream ? (
          <Button size="small" type={streaming ? "default" : "primary"} disabled={streaming} onClick={onStartStream}>
            实时跟随
          </Button>
        ) : null}
        {onStopStream ? (
          <Button size="small" danger disabled={!streaming} onClick={onStopStream}>
            停止
          </Button>
        ) : null}
      </Space>
      {loading && !logsText ? (
        <div className="pod-logs-panel__loading">
          <Spin tip="加载日志..." />
        </div>
      ) : (
        <pre className="pod-logs-panel__content">{logsText || "暂无日志，点击「拉取日志」"}</pre>
      )}
      {streaming ? (
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          实时跟随中…
        </Typography.Text>
      ) : null}
    </div>
  );
}
