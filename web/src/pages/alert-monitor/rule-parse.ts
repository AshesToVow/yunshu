import type { RuleBuilderCondition, RuleBuilderLogic, RuleComparator } from "./platform-provider-types";

export function parseTemplatePresetPair(raw: string): { summary: string; description: string } | null {
  const s = String(raw || "").trim();
  if (!s) return null;
  try {
    const parsed = JSON.parse(s) as { summary?: string; description?: string };
    const summary = String(parsed.summary || "").trim();
    const description = String(parsed.description || "").trim();
    if (!summary || !description) return null;
    return { summary, description };
  } catch {
    return null;
  }
}

export function parseRuleBuilderExpr(exprRaw: string): { conditions: RuleBuilderCondition[]; logic: RuleBuilderLogic } | null {
  const expr = String(exprRaw || "").trim();
  if (!expr) return null;
  const hasOr = /\s+or\s+/i.test(expr);
  const hasAnd = /\s+and\s+/i.test(expr);
  if (hasOr && hasAnd) return null;
  const logic: RuleBuilderLogic = hasOr ? "or" : "and";
  const parts = (hasOr ? expr.split(/\s+or\s+/i) : expr.split(/\s+and\s+/i)).map((p) => p.trim()).filter(Boolean);
  const parsed: RuleBuilderCondition[] = [];
  for (const p0 of parts) {
    const p = p0.replace(/^\((.*)\)$/, "$1").trim();
    const m = p.match(/^(.+?)\s*(>=|<=|==|!=|>|<)\s*(-?\d+(?:\.\d+)?)\s*$/);
    if (!m) return null;
    parsed.push({
      metric: String(m[1] || "").trim(),
      comparator: (m[2] as RuleComparator) || ">",
      threshold: Number(m[3]),
    });
  }
  if (!parsed.length) return null;
  return { conditions: parsed, logic };
}

export function buildRuleExprByConditions(conditions: RuleBuilderCondition[], logic: RuleBuilderLogic): string {
  const valid = conditions
    .map((c) => ({
      metric: String(c.metric || "").trim(),
      comparator: c.comparator,
      threshold: c.threshold,
    }))
    .filter((c) => c.metric && c.threshold !== null && !Number.isNaN(c.threshold));
  if (!valid.length) return "";
  if (valid.length === 1) {
    return `${valid[0].metric} ${valid[0].comparator} ${valid[0].threshold}`;
  }
  const joiner = logic === "or" ? " or " : " and ";
  return valid.map((c) => `(${c.metric} ${c.comparator} ${c.threshold})`).join(joiner);
}
