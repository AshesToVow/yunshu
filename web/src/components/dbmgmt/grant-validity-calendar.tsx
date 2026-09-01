import { Calendar, Space, Tag, Typography } from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import { useMemo, useState } from "react";

export type GrantValidityPeriod = { start: Dayjs; end: Dayjs };

const PRESETS: { label: string; days?: number; permanent?: boolean }[] = [
  { label: "7 天", days: 7 },
  { label: "30 天", days: 30 },
  { label: "90 天", days: 90 },
  { label: "1 年", days: 365 },
  { label: "永久有效", permanent: true },
];

function normalizeRange(start: Dayjs, end: Dayjs): GrantValidityPeriod {
  const s = start.startOf("day");
  const e = end.startOf("day");
  if (e.isBefore(s)) return { start: e, end: s.endOf("day") };
  return { start: s, end: e.endOf("day") };
}

export function grantPeriodToExpiresAt(period?: GrantValidityPeriod | null): string | undefined {
  if (!period?.end) return undefined;
  return period.end.endOf("day").toISOString();
}

export function expiresAtToGrantPeriod(expiresAt?: string): GrantValidityPeriod | null {
  if (!expiresAt) return null;
  const end = dayjs(expiresAt);
  if (!end.isValid()) return null;
  const d = expiresAt.slice(0, 10);
  if (!d || d >= "9999") return null;
  const start = end.isAfter(dayjs()) ? dayjs().startOf("day") : end.startOf("day");
  return { start, end: end.endOf("day") };
}

export function formatGrantPeriodSummary(period?: GrantValidityPeriod | null): string {
  if (!period) return "永久有效";
  const days = period.end.startOf("day").diff(period.start.startOf("day"), "day") + 1;
  return `${period.start.format("YYYY-MM-DD")} 至 ${period.end.format("YYYY-MM-DD")}（共 ${days} 天）`;
}

export function GrantValidityCalendarPicker({
  value,
  onChange,
}: {
  value?: GrantValidityPeriod | null;
  onChange?: (v: GrantValidityPeriod | null) => void;
}) {
  const today = dayjs().startOf("day");
  const [draftStart, setDraftStart] = useState<Dayjs | null>(null);
  const [hoverDay, setHoverDay] = useState<Dayjs | null>(null);

  const previewEnd = useMemo(() => {
    if (value?.end) return value.end.startOf("day");
    if (draftStart && hoverDay) return hoverDay.startOf("day");
    return null;
  }, [value, draftStart, hoverDay]);

  const previewStart = useMemo(() => {
    if (value?.start) return value.start.startOf("day");
    return draftStart?.startOf("day") ?? null;
  }, [value, draftStart]);

  const activePreset = useMemo(() => {
    if (!value) return "永久有效";
    const days = value.end.startOf("day").diff(value.start.startOf("day"), "day") + 1;
    const hit = PRESETS.find((p) => p.days === days && value.start.isSame(today, "day"));
    return hit?.label;
  }, [value, today]);

  function applyPreset(p: (typeof PRESETS)[number]) {
    if (p.permanent) {
      setDraftStart(null);
      onChange?.(null);
      return;
    }
    if (!p.days) return;
    onChange?.(normalizeRange(today, today.add(p.days - 1, "day")));
    setDraftStart(null);
  }

  function onSelect(date: Dayjs) {
    const d = date.startOf("day");
    if (d.isBefore(today)) return;
    if (!draftStart) {
      setDraftStart(d);
      return;
    }
    onChange?.(normalizeRange(draftStart, d));
    setDraftStart(null);
  }

  function isInRange(d: Dayjs) {
    const start = previewStart;
    const end = previewEnd;
    if (!start || !end) return false;
    const cur = d.startOf("day");
    const s = start.isBefore(end) ? start : end;
    const e = start.isBefore(end) ? end : start;
    return !cur.isBefore(s) && !cur.isAfter(e);
  }

  function isEdge(d: Dayjs, edge: "start" | "end") {
    const start = previewStart;
    const end = previewEnd;
    if (!start || !end) return edge === "start" && draftStart?.isSame(d, "day");
    const s = start.isBefore(end) ? start : end;
    const e = start.isBefore(end) ? end : start;
    return edge === "start" ? d.isSame(s, "day") : d.isSame(e, "day");
  }

  const fullCellRender = (date: Dayjs) => {
    const disabled = date.startOf("day").isBefore(today);
    const inRange = !disabled && isInRange(date);
    const isStart = isEdge(date, "start");
    const isEnd = isEdge(date, "end");
    const bg = disabled
      ? "transparent"
      : inRange
        ? isStart || isEnd
          ? "linear-gradient(135deg, #1677ff 0%, #4096ff 100%)"
          : "rgba(22, 119, 255, 0.12)"
        : "transparent";
    const color = isStart || isEnd ? "#fff" : disabled ? "rgba(0,0,0,0.25)" : undefined;
    const fontWeight = isStart || isEnd ? 600 : undefined;

    return (
      <div
        onMouseEnter={() => {
          if (draftStart) setHoverDay(date);
        }}
        style={{
          margin: "0 auto",
          width: 28,
          height: 28,
          lineHeight: "28px",
          borderRadius: isStart || isEnd ? 6 : 4,
          background: bg,
          color,
          fontWeight,
          textAlign: "center",
        }}
      >
        {date.date()}
      </div>
    );
  };

  return (
    <div className="grant-validity-calendar">
      <Space wrap style={{ marginBottom: 10 }}>
        {PRESETS.map((p) => (
          <Tag.CheckableTag key={p.label} checked={activePreset === p.label} onChange={(checked) => checked && applyPreset(p)}>
            {p.label}
          </Tag.CheckableTag>
        ))}
      </Space>
      <div
        style={{
          border: "1px solid #f0f0f0",
          borderRadius: 8,
          padding: 8,
          background: "#fafafa",
        }}
      >
        <Calendar
          fullscreen={false}
          onSelect={onSelect}
          fullCellRender={fullCellRender}
          disabledDate={(d) => d.startOf("day").isBefore(today)}
        />
      </div>
      <Typography.Paragraph type="secondary" style={{ marginTop: 10, marginBottom: 0 }}>
        {draftStart ? (
          <>
            已选开始：<Typography.Text strong>{draftStart.format("YYYY-MM-DD")}</Typography.Text>
            ，请点击结束日期完成选择
          </>
        ) : (
          <>
            当前授权：<Typography.Text strong>{formatGrantPeriodSummary(value)}</Typography.Text>
            <br />
            在日历上依次点击<strong>开始</strong>与<strong>结束</strong>日期；审批通过后在该区间内有效
          </>
        )}
      </Typography.Paragraph>
    </div>
  );
}
