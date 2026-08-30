/**
 * 告警配置中心：入库体与接收组字段的解析纯函数（RF-07 第一步拆分产物）
 *
 * 从 `alert-config-center-panel.tsx` 原地搬迁，逐字保留语义。四个函数都在处理
 * 「后端可能给结构化字段、也可能只给 JSON 字符串」的双形态兼容，任何一处
 * 顺序调整都会改变展示结果，改动前请先确认后端 `alert_receiver_group` /
 * `alert_event.request_payload` 的实际下发形态。
 */
import { type AlertReceiverGroup } from "../../services/alert-subscriptions";

import { stringifyPrettyJSON } from "../../services/alert-mappers";

/** 从告警历史入库体中解析顶层 `labels`（与后端 hydrate 逻辑一致）。 */
export function parseLabelsFromAlertEventRequestPayload(raw?: string): Record<string, string> {
  const s = String(raw || "").trim();
  if (!s) return {};
  try {
    const payload = JSON.parse(s) as Record<string, unknown>;
    const labels = payload?.labels;
    if (labels && typeof labels === "object" && !Array.isArray(labels)) {
      const out: Record<string, string> = {};
      for (const [k, v] of Object.entries(labels as Record<string, unknown>)) {
        const vs = String(v ?? "").trim();
        if (vs && vs !== "<nil>") {
          out[String(k).trim()] = vs;
        }
      }
      return out;
    }
  } catch {
    /* ignore */
  }
  return {};
}

/**
 * 接收组绑定的通道 ID：优先用结构化 `channel_ids`，为空时回落解析
 * `channel_ids_json`。两条链路都过滤掉 <= 0 的脏 ID。
 */
export function parseReceiverGroupChannelIds(g: AlertReceiverGroup): number[] {
  if (Array.isArray(g.channel_ids) && g.channel_ids.length) {
    return g.channel_ids.map((id) => Number(id)).filter((id) => id > 0);
  }
  try {
    const parsed = JSON.parse(String(g.channel_ids_json || "[]")) as unknown;
    if (Array.isArray(parsed)) {
      return parsed.map((id) => Number(id)).filter((id) => id > 0);
    }
  } catch {
    /* ignore */
  }
  return [];
}

/** 接收组静态抄送邮箱：与 `parseReceiverGroupChannelIds` 同一套回落策略。 */
export function parseReceiverGroupEmails(g: AlertReceiverGroup): string[] {
  if (Array.isArray(g.email_recipients) && g.email_recipients.length) {
    return g.email_recipients.map((e) => String(e).trim()).filter(Boolean);
  }
  try {
    const parsed = JSON.parse(String(g.email_recipients_json || "[]")) as unknown;
    if (Array.isArray(parsed)) {
      return parsed.map((e) => String(e).trim()).filter(Boolean);
    }
  } catch {
    /* ignore */
  }
  return [];
}

/** 入库体美化展示：非法 JSON 时原样返回，避免历史脏数据无法查看。 */
export function prettifyAlertRequestPayload(raw?: string): string {
  const s = String(raw || "").trim();
  if (!s) return "";
  try {
    return stringifyPrettyJSON(JSON.parse(s) as unknown, s);
  } catch {
    return s;
  }
}
