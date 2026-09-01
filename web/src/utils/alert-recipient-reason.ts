export type RecipientReasonKind = "assignee" | "assignee_and_cc" | "channel" | "unknown";

export type RecipientReason = {
  kind: RecipientReasonKind;
  short: string;
  detail: string;
};

function asStringList(raw: unknown): string[] {
  if (raw == null) return [];
  if (Array.isArray(raw)) {
    return raw.map((x) => String(x ?? "").trim()).filter(Boolean);
  }
  if (typeof raw === "string") {
    const s = raw.trim();
    if (!s) return [];
    if (s.startsWith("[")) {
      try {
        return asStringList(JSON.parse(s) as unknown);
      } catch {
        /* fallthrough */
      }
    }
    return s.split(/[,;]/).map((x) => x.trim()).filter(Boolean);
  }
  return [];
}

function parsePayload(raw?: string | null): Record<string, unknown> | null {
  const s = String(raw || "").trim();
  if (!s) return null;
  try {
    const v = JSON.parse(s) as unknown;
    if (v && typeof v === "object" && !Array.isArray(v)) {
      return v as Record<string, unknown>;
    }
  } catch {
    return null;
  }
  return null;
}

function emailsOf(payload: Record<string, unknown>, key: string): string[] {
  return asStringList(payload[key]).map((e) => e.toLowerCase());
}

/** 从投递入库 JSON 判断收件人来自规则处理人/值班，还是通道固定收件人。 */
export function explainAlertRecipients(requestPayload?: string | null): RecipientReason {
  const payload = parsePayload(requestPayload);
  if (!payload) {
    return { kind: "unknown", short: "-", detail: "无投递载荷，无法判断收件人来源。" };
  }
  const mode = String(payload.recipient_mode || "").trim().toLowerCase();
  const assignee = emailsOf(payload, "assignee_emails");
  const to = [
    ...emailsOf(payload, "to"),
    ...emailsOf(payload, "recipients"),
    ...emailsOf(payload, "emails"),
  ];
  if (mode === "channel_only") {
    return {
      kind: "channel",
      short: "仅通道收件人",
      detail: "规则配置为仅使用通道/接收组固定收件人（忽略处理人邮箱）。",
    };
  }
  if (mode === "assignee_only" && assignee.length > 0) {
    return {
      kind: "assignee",
      short: "仅处理人/值班",
      detail: "规则配置为仅发送处理人与值班邮箱，未使用通道固定收件人。",
    };
  }
  if (assignee.length > 0 && (mode === "assignee_and_cc" || !mode)) {
    return {
      kind: "assignee_and_cc",
      short: "处理人+抄送通道",
      detail: "处理人/值班邮箱与通道固定收件人一并发送（默认策略）。",
    };
  }
  if (assignee.length > 0) {
    return {
      kind: "assignee",
      short: "规则处理人/值班",
      detail: "规则处理人或当前值班邮箱参与投递。",
    };
  }
  if (to.length > 0) {
    return {
      kind: "channel",
      short: "通道默认收件人",
      detail: "未配置规则处理人/值班，使用通道或接收组上的固定收件人。",
    };
  }
  return { kind: "unknown", short: "-", detail: "载荷中没有邮箱列表（可能是 IM 通道或仅留痕）。" };
}

export function parseLabelMap(raw?: string | null): Record<string, string> {
  const s = String(raw || "").trim();
  if (!s) return {};
  try {
    const v = JSON.parse(s) as unknown;
    if (!v || typeof v !== "object" || Array.isArray(v)) return {};
    const out: Record<string, string> = {};
    for (const [k, val] of Object.entries(v as Record<string, unknown>)) {
      const vs = String(val ?? "").trim();
      if (k && vs && vs !== "<nil>") out[k] = vs;
    }
    return out;
  } catch {
    return {};
  }
}
