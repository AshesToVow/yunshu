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

export type SilenceDisplayRow = AlertSilenceItem & { source: "platform"; rowKey: string };

export type RuleComparator = ">" | ">=" | "<" | "<=" | "==" | "!=";
export type RuleBuilderLogic = "and" | "or";
export type RuleBuilderCondition = { metric: string; comparator: RuleComparator; threshold: number | null };

export type PromTableView = {
  columns: ColumnsType<Record<string, string>>;
  dataSource: Record<string, string>[];
};
