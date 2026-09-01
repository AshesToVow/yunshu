// @ts-nocheck
/** Shared types extracted for alert-monitor platform / modals (avoid circular imports). */
import type { Dayjs } from "dayjs";
import type { ColumnsType } from "antd/es/table";
import type { AlertSilenceItem } from "../../services/alert-platform";

export type MetricLabelFilter = { key: string; op: "=" | "!=" | "=~" | "!~"; value: string };

export type SilenceMatcherForm = { name: string; value: string; is_regex: boolean };

export type PromNativeAlertRow = {
  key: string;
  alertname: string;
  state: string;
  labelsShort: string;
  activeAt?: string;
  labels: Record<string, string>;
};

export type QuickSilenceTarget = {
  key: string;
  name: string;
  labels: Record<string, string>;
  startsAt: Dayjs;
  endsAt: Dayjs;
};

export type AlertmanagerSilenceRow = {
  rowKey: string;
  source: "alertmanager";
  amId: string;
  name: string;
  comment?: string;
  matchers?: Array<{ name: string; value: string; is_regex?: boolean }>;
  starts_at: string;
  ends_at: string;
  state: string;
  enabled: boolean;
};

export type SilenceDisplayRow =
  | (AlertSilenceItem & { source: "platform"; rowKey: string })
  | AlertmanagerSilenceRow;

export type RuleComparator = ">" | ">=" | "<" | "<=" | "==" | "!=";
export type RuleBuilderLogic = "and" | "or";
export type RuleBuilderCondition = { metric: string; comparator: RuleComparator; threshold: number | null };

export type PromTableView = {
  columns: ColumnsType<Record<string, string>>;
  dataSource: Record<string, string>[];
};
