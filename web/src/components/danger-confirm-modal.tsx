import { ExclamationCircleOutlined } from "@ant-design/icons";
import { Alert, Input, Modal, Typography } from "antd";
import { useEffect, useState, type ReactNode } from "react";

export interface DangerConfirmPayload {
  confirm: string;
  reason: string;
}

interface DangerConfirmModalProps {
  open: boolean;
  title: string;
  /** 需要用户完整输入的目标类型，如“索引名”“Topic” */
  targetLabel: string;
  target: string;
  description?: ReactNode;
  confirmText?: string;
  loading?: boolean;
  /** 是否要求完整输入目标名（默认 true）；批量/非具名操作可只填原因。 */
  requireTyping?: boolean;
  /** 原因输入框标题，默认“删除原因（必填）”。 */
  reasonLabel?: string;
  onCancel: () => void;
  onConfirm: (payload: DangerConfirmPayload) => void | Promise<void>;
}

/** 不可恢复操作二次确认：必须输入完整目标名并填写删除原因。 */
export function DangerConfirmModal({
  open,
  title,
  targetLabel,
  target,
  description,
  confirmText = "确认删除",
  loading = false,
  requireTyping = true,
  reasonLabel = "删除原因（必填）",
  onCancel,
  onConfirm,
}: DangerConfirmModalProps) {
  const [confirmValue, setConfirmValue] = useState("");
  const [reason, setReason] = useState("");

  useEffect(() => {
    if (open) {
      setConfirmValue("");
      setReason("");
    }
  }, [open, target]);

  const typingOK = !requireTyping || confirmValue.trim() === target.trim();
  const canSubmit = typingOK && reason.trim().length >= 2;

  return (
    <Modal
      open={open}
      title={title}
      okText={confirmText}
      cancelText="取消"
      okButtonProps={{ danger: true, disabled: !canSubmit }}
      confirmLoading={loading}
      maskClosable={!loading}
      closable={!loading}
      destroyOnClose
      onCancel={() => {
        if (!loading) onCancel();
      }}
      onOk={() => {
        if (!canSubmit || loading) return;
        void onConfirm({ confirm: requireTyping ? confirmValue.trim() : target.trim(), reason: reason.trim() });
      }}
    >
      <Alert
        type="warning"
        showIcon
        icon={<ExclamationCircleOutlined />}
        message="该操作不可恢复，请谨慎确认。"
        description={description}
        style={{ marginBottom: 16 }}
      />
      {requireTyping ? (
        <>
          <Typography.Paragraph>
            请输入 <Typography.Text code>{target}</Typography.Text> 以确认删除：
          </Typography.Paragraph>
          <Input
            value={confirmValue}
            onChange={(e) => setConfirmValue(e.target.value)}
            placeholder={`请输入完整${targetLabel}`}
            autoComplete="off"
            disabled={loading}
          />
        </>
      ) : null}
      <Typography.Paragraph style={{ marginTop: 16, marginBottom: 8 }}>{reasonLabel}</Typography.Paragraph>
      <Input.TextArea
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        placeholder="请说明删除原因，便于审计追溯"
        rows={3}
        maxLength={200}
        showCount
        disabled={loading}
      />
    </Modal>
  );
}
