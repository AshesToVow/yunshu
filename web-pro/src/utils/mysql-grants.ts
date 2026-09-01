// @ts-nocheck
import { MYSQL_PRIV_GROUPS } from "../components/dbmgmt/dbmgmt-ui-shared";

/** 库级权限不可直接授予，需走 *.* 的 MySQL 权限 */
export const MYSQL_GLOBAL_ONLY_PRIVS = [
  "GRANT",
  "SUPER",
  "PROCESS",
  "RELOAD",
  "SHUTDOWN",
  "SHOW DATABASES",
  "REPLICATION CLIENT",
  "REPLICATION SLAVE",
  "CREATE USER",
] as const;

export const ALL_MYSQL_PRIVS = [
  ...MYSQL_PRIV_GROUPS.data,
  ...MYSQL_PRIV_GROUPS.structure,
  ...MYSQL_PRIV_GROUPS.management,
] as const;

const KNOWN_PRIVS_SORTED = [...ALL_MYSQL_PRIVS].sort((a, b) => b.length - a.length);

export type MysqlPrivLevel = "global" | "database";

function parseGrantObject(onPart: string): { scope: "global" | "database" | "table"; database?: string } {
  const normalized = onPart.trim().replace(/`/g, "");
  if (normalized === "*.*" || normalized === "*") {
    return { scope: "global" };
  }
  const dbWild = normalized.match(/^([^.]+)\.\*$/);
  if (dbWild) {
    return { scope: "database", database: dbWild[1].toLowerCase() };
  }
  const dbTable = normalized.match(/^([^.]+)\./);
  if (dbTable) {
    return { scope: "table", database: dbTable[1].toLowerCase() };
  }
  return { scope: "global" };
}

function parsePrivPart(part: string): string[] {
  const text = part.trim();
  if (!text) return [];
  const upper = text.toUpperCase();
  if (upper.includes("ALL PRIVILEGES") || /\bALL\b/.test(upper)) {
    return [...ALL_MYSQL_PRIVS];
  }
  let remaining = upper;
  const found: string[] = [];
  for (const priv of KNOWN_PRIVS_SORTED) {
    if (remaining.includes(priv)) {
      found.push(priv);
      remaining = remaining.split(priv).join(" ");
    }
  }
  return found;
}

function grantApplies(
  scope: ReturnType<typeof parseGrantObject>["scope"],
  database: string | undefined,
  options: { level: MysqlPrivLevel; database?: string },
): boolean {
  if (options.level === "global") {
    return scope === "global";
  }
  const targetDb = options.database?.trim().toLowerCase();
  if (!targetDb) return false;
  if (scope === "global") return true;
  return (scope === "database" || scope === "table") && database === targetDb;
}

/** 从 SHOW GRANTS 结果解析用户在指定级别的已有 MySQL 权限。 */
export function parseMysqlGrantPrivileges(
  grantLines: string[],
  options: { level: MysqlPrivLevel; database?: string },
): Set<string> {
  const privs = new Set<string>();
  for (const raw of grantLines) {
    const line = raw.trim();
    if (!line || /^GRANT\s+USAGE\b/i.test(line) || /^GRANT\s+PROXY\b/i.test(line)) {
      continue;
    }
    const m = line.match(/^GRANT\s+(.+?)\s+ON\s+(.+?)\s+TO[\s`]/i);
    if (!m) continue;
    const obj = parseGrantObject(m[2]);
    if (!grantApplies(obj.scope, obj.database, options)) continue;
    for (const p of parsePrivPart(m[1])) {
      privs.add(p);
    }
  }
  return privs;
}
