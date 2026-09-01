// @ts-nocheck
import { CheckOutlined, DownOutlined } from "@ant-design/icons";
import { Button, Dropdown } from "antd";
import { useMemo } from "react";
import { useDictOptions } from "../../hooks/use-dict-options";

export const ALERT_ACK_TTL_DICT = "alert_ack_ttl_minutes";

export type AckTTLOption = { label: string; minutes: number };

export function useAckTTLOptions(): AckTTLOption[] {
  const raw = useDictOptions(ALERT_ACK_TTL_DICT);
  return useMemo(() => {
    const out: AckTTLOption[] = [];
    const seen = new Set<number>();
    for (const item of raw) {
      const minutes = Number(item.value);
      if (!Number.isFinite(minutes) || minutes <= 0) continue;
      if (seen.has(minutes)) continue;
      seen.add(minutes);
      out.push({ label: String(item.label || `${minutes} 分钟`), minutes });
    }
    return out;
  }, [raw]);
}

type Props = {
  acked: boolean;
  loading?: boolean;
  disabled?: boolean;
  variant?: "link" | "default";
  onAck: (minutes?: number) => void;
  onClear: () => void;
};

export function AlertAckActionButton({ acked, loading, disabled, variant = "link", onAck, onClear }: Props) {
  const options = useAckTTLOptions();
  const btnType = variant === "link" ? "link" : "default";
  const size = variant === "link" ? "small" : undefined;
  if (acked) {
    return (
      <Button type={btnType} size={size} icon={<CheckOutlined />} loading={loading} disabled={disabled} onClick={onClear}>
        取消认领
      </Button>
    );
  }
  const items = options.map((o) => ({
    key: String(o.minutes),
    label: `认领 ${o.label}`,
    onClick: () => onAck(o.minutes),
  }));
  if (items.length <= 1) {
    return (
      <Button type={btnType} size={size} icon={<CheckOutlined />} loading={loading} disabled={disabled} onClick={() => onAck(options[0]?.minutes)}>
        认领
      </Button>
    );
  }
  return (
    <Dropdown menu={{ items }} disabled={disabled}>
      <Button type={btnType} size={size} icon={<CheckOutlined />} loading={loading}>
        认领 <DownOutlined />
      </Button>
    </Dropdown>
  );
}
