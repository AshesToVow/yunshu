/** Shared types extracted for alert-monitor modals (avoid circular imports). */
export type MetricLabelFilter = { key: string; op: "=" | "!=" | "=~" | "!~"; value: string };
