// @ts-nocheck
/** 路由/策略名称人话化：去掉 global:、[policy_n]，根节点默认不展示。 */

const TOKEN_ZH: Record<string, string> = {
  prod: "生产",
  production: "生产",
  staging: "预发",
  test: "测试",
  dev: "开发",
  warning: "Warning",
  critical: "致命",
  info: "提示",
  email: "邮件",
  mail: "邮件",
  wecom: "企微",
  wechat: "微信",
  ding: "钉钉",
  dingding: "钉钉",
  root: "根路由",
};

export function splitMatchedPolicyNames(raw?: string | string[] | null): string[] {
  if (Array.isArray(raw)) {
    return raw.map((s) => String(s || "").trim()).filter(Boolean);
  }
  const s = String(raw || "").trim();
  if (!s) return [];
  return s
    .split(",")
    .map((p) => p.trim())
    .filter(Boolean);
}

export function humanizeMatchedPolicyName(raw: string): string {
  let s = String(raw || "").trim();
  if (!s) return "";
  s = s.replace(/\[[^\]]+\]$/, "").trim();
  if (s.startsWith("global:")) s = s.slice("global:".length);
  if (s.startsWith("migrated:")) s = s.slice("migrated:".length);
  if (!s || s === "root") return "全局根路由";
  const parts = s.split(/[-_./]/).filter(Boolean);
  if (!parts.length) return s;
  return parts.map((p) => TOKEN_ZH[p.toLowerCase()] || p).join(" · ");
}

/** 列表展示：去掉根路由噪音，多条用顿号。原始名放 tooltip。 */
export function formatMatchedPolicyNamesDisplay(raw?: string | string[] | null): { text: string; title: string } {
  const rawList = splitMatchedPolicyNames(raw);
  const human = rawList
    .map(humanizeMatchedPolicyName)
    .filter((n) => n && n !== "全局根路由");
  const text = human.length ? human.join("、") : rawList.length ? "全局默认路由" : "-";
  return { text, title: rawList.join("，") || text };
}
