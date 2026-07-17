/** 将查询结果导出为 CSV 并触发下载 */
export function downloadQueryResultCsv(filename: string, columns: string[], rows: unknown[][]) {
  const escape = (v: unknown) => {
    const s = v == null ? "" : String(v);
    if (/[",\n\r]/.test(s)) return `"${s.replace(/"/g, '""')}"`;
    return s;
  };
  const lines = [columns.map(escape).join(",")];
  for (const row of rows) {
    lines.push(row.map(escape).join(","));
  }
  const blob = new Blob(["\uFEFF" + lines.join("\n")], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

/** 去掉 USE 语句，保留实际查询（目标库由请求参数指定） */
export function sanitizeQuerySql(sql: string): string {
  const parts = sql
    .split(";")
    .map((s) => s.trim())
    .filter(Boolean)
    .filter((s) => !/^\s*USE\s+/i.test(s));
  return parts.length === 1 ? parts[0] : parts.join(";\n");
}

/** 控制台写操作风险粗判（与后端 SQL Guard 口径对齐的简化版） */
export function guessSqlRiskLevel(sql: string): "low" | "medium" | "high" | "blocked" {
  const text = sanitizeQuerySql(sql).trim() || sql.trim();
  if (!text) return "low";
  // 多语句是否允许由后端 goInception 预检决定，前端不做本地阻断
  if (/\b(DROP\s+DATABASE|GRANT|REVOKE|TRUNCATE)\b/i.test(text)) return "blocked";
  if (/\b(DROP\s+TABLE|ALTER\s+TABLE|CREATE\s+INDEX|DROP\s+INDEX)\b/i.test(text)) return "high";
  if (/\b(CREATE|ALTER|DROP|TRUNCATE|RENAME)\b/i.test(text)) return "high";
  if (/\b(INSERT|UPDATE|DELETE)\b/i.test(text)) return "medium";
  if (/^\s*(SELECT|SHOW|DESCRIBE|DESC|EXPLAIN|WITH)\b/i.test(text)) return "low";
  return "high";
}

export function formatSqlCheckSummary(res: {
  goinception?: boolean;
  syntax_type?: number;
  error_count?: number;
  warning_count?: number;
  risk_level?: string;
  error?: string;
}) {
  const syntax = res.syntax_type === 1 ? "DDL" : res.syntax_type === 2 ? "DML" : "其他";
  if (res.error) {
    return res.error;
  }
  if (res.goinception) {
    return `goInception 预检：${syntax}，错误 ${res.error_count ?? 0}，警告 ${res.warning_count ?? 0}`;
  }
  return `本地规则评估：风险 ${riskLevelLabel(res.risk_level ?? "")}`;
}

export function formatSqlCheckFailure(res: { error?: string; error_count?: number }) {
  if (res.error?.trim()) return res.error.trim();
  if ((res.error_count ?? 0) > 0) return `预检发现 ${res.error_count} 个错误，请修正后提交`;
  return "预检未通过，请修正后提交";
}

export function riskLevelLabel(level: string) {
  const map: Record<string, string> = {
    low: "低",
    medium: "中",
    high: "高",
    blocked: "阻断",
  };
  return map[level] ?? level;
}

export function riskLevelColor(level: string) {
  if (level === "blocked") return "red";
  if (level === "high") return "red";
  if (level === "medium") return "orange";
  return "green";
}

/** 按驱动类型引用 SQL 标识符（表/列名） */
export function quoteSqlIdent(name: string, driver?: string) {
  const d = (driver || "mysql").toLowerCase();
  if (d.includes("postgres")) return `"${name.replace(/"/g, '""')}"`;
  return `\`${name.replace(/`/g, "``")}\``;
}
