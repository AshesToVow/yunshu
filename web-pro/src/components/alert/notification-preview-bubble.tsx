// @ts-nocheck
import { Card, Segmented, Space, Tag, Typography } from "antd";
import type { ReactNode } from "react";

function renderInlineSegment(text: string): ReactNode[] {
  const parts: ReactNode[] = [];
  const re = /(\*\*[^*]+\*\*|`[^`]+`)/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let idx = 0;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) parts.push(text.slice(last, m.index));
    const token = m[0];
    if (token.startsWith("**")) {
      parts.push(<strong key={idx++}>{token.slice(2, -2)}</strong>);
    } else if (token.startsWith("`")) {
      parts.push(
        <Typography.Text code key={idx++} style={{ fontSize: 12 }}>
          {token.slice(1, -1)}
        </Typography.Text>,
      );
    }
    last = m.index + token.length;
  }
  if (last < text.length) parts.push(text.slice(last));
  return parts.length ? parts : [text];
}

export function NotificationPreviewBubble({
  title,
  rendered,
  status,
  loading,
  error,
  channelType,
}: {
  title?: string;
  rendered: string;
  status: "firing" | "resolved";
  loading?: boolean;
  error?: string;
  channelType?: string;
}) {
  const statusColor = status === "firing" ? "#ff4d4f" : "#52c41a";
  const statusLabel = status === "firing" ? "触发告警" : "已恢复";

  return (
    <Card
      size="small"
      title={title ?? "通知预览"}
      extra={
        <Space size={6}>
          {channelType ? <Tag>{channelType}</Tag> : null}
          <Tag color={status === "firing" ? "error" : "success"}>{statusLabel}</Tag>
        </Space>
      }
      styles={{ body: { padding: 12, background: "#f5f7fb" } }}
    >
      <div
        style={{
          maxWidth: 420,
          margin: "0 auto",
          background: "#fff",
          borderRadius: 10,
          border: "1px solid #e8edf5",
          boxShadow: "0 6px 20px rgba(15,23,42,0.06)",
          overflow: "hidden",
        }}
      >
        <div style={{ padding: "10px 14px", background: statusColor, color: "#fff", fontSize: 13, fontWeight: 600 }}>
          {loading ? "渲染中…" : error ? "渲染失败" : `模拟 ${channelType || "Webhook"} 通知`}
        </div>
        <div style={{ padding: 14, fontSize: 14, lineHeight: 1.65, color: "#1f2937", minHeight: 160 }}>
          {error ? (
            <Typography.Text type="danger">{error}</Typography.Text>
          ) : rendered ? (
            rendered.split("\n").map((line, i) => (
              <div key={i} style={{ marginBottom: line.trim() === "" ? 8 : 2 }}>
                {line.trim() === "" ? <br /> : renderInlineSegment(line)}
              </div>
            ))
          ) : (
            <Typography.Text type="secondary">编辑左侧模板后此处实时预览</Typography.Text>
          )}
        </div>
      </div>
    </Card>
  );
}

export function NotificationPreviewModeSwitch({
  value,
  onChange,
}: {
  value: "firing" | "resolved";
  onChange: (v: "firing" | "resolved") => void;
}) {
  return (
    <Segmented
      value={value}
      onChange={(v) => onChange(v as "firing" | "resolved")}
      options={[
        { label: "触发预览", value: "firing" },
        { label: "恢复预览", value: "resolved" },
      ]}
    />
  );
}
