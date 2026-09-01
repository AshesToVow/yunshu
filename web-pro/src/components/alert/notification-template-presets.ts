// @ts-nocheck
export type TemplatePreviewStatus = "firing" | "resolved";

export const DEFAULT_FIRING_TEMPLATE =
  "【{{.StatusText}}】**{{.Title}}**\n\n- 现象：{{.Summary}}（当前值 {{.Current}}）\n\n- 级别：`{{.Severity}}` · 项目：`{{.ProjectName}}` · 集群：`{{.Cluster}}`\n\n- 时间：{{.OccurredAt}}\n\n- 打开事件：{{.EventPath}}\n\n- 标签：{{.LabelsText}}";

export const DEFAULT_RESOLVED_TEMPLATE =
  "【{{.StatusText}}】**{{.Title}}**\n\n- 摘要：{{.Summary}}\n\n- 级别：`{{.Severity}}` · 项目：`{{.ProjectName}}`\n\n- 开始：{{.StartsAt}} · 恢复：{{.EndsAt}}\n\n- 打开事件：{{.EventPath}}";

export const SIMPLE_FIRING_TEMPLATE =
  "【告警】{{.Title}}\n现象：{{.Summary}}（{{.Current}}）\n级别：{{.Severity}} · 项目：{{.ProjectName}}\n打开：{{.EventPath}}";

export const SIMPLE_RESOLVED_TEMPLATE =
  "【恢复】{{.Title}}\n开始：{{.StartsAt}} · 结束：{{.EndsAt}}\n项目：{{.ProjectName}}\n打开：{{.EventPath}}";

export const DETAILED_FIRING_TEMPLATE =
  "【{{.StatusText}}】{{.Title}}\n级别：{{.Severity}}\n项目：{{.ProjectName}}\n集群：{{.Cluster}}\n现象：{{.Summary}}\n当前值：{{.Current}}\n描述：{{.Description}}\n时间：{{.OccurredAt}}\n标签：{{.LabelsText}}\n打开事件：{{.EventPath}}\nGenerator：{{.GeneratorURL}}";

export const DETAILED_RESOLVED_TEMPLATE =
  "【{{.StatusText}}】{{.Title}}\n级别：{{.Severity}}\n项目：{{.ProjectName}}\n集群：{{.Cluster}}\n开始：{{.StartsAt}}\n结束：{{.EndsAt}}\n摘要：{{.Summary}}\n打开事件：{{.EventPath}}\n标签：{{.LabelsText}}";

export const CHANNEL_PRESET_OPTIONS = [
  { label: "新手简版（推荐）", value: "simple" },
  { label: "标准版（默认）", value: "default" },
  { label: "详细排障版", value: "detailed" },
] as const;

export function presetTemplateByMode(mode?: string) {
  switch (String(mode || "").trim()) {
    case "simple":
      return { firing: SIMPLE_FIRING_TEMPLATE, resolved: SIMPLE_RESOLVED_TEMPLATE };
    case "detailed":
      return { firing: DETAILED_FIRING_TEMPLATE, resolved: DETAILED_RESOLVED_TEMPLATE };
    default:
      return { firing: DEFAULT_FIRING_TEMPLATE, resolved: DEFAULT_RESOLVED_TEMPLATE };
  }
}
