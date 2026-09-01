// @ts-nocheck
import { Tag } from "antd";

const PHASE_COLORS: Record<string, string> = {
  running: "success",
  succeeded: "processing",
  pending: "warning",
  failed: "error",
  unknown: "default",
  terminating: "warning",
  degraded: "warning",
};

type PhaseTagProps = {
  phase?: string | null;
};

export function PhaseTag({ phase }: PhaseTagProps) {
  const text = (phase || "-").trim();
  const key = text.toLowerCase();
  return <Tag color={PHASE_COLORS[key] ?? "default"}>{text}</Tag>;
}
