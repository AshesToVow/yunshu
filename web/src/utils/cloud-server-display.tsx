// 云服务器展示层辅助：厂商名映射、计费方式中文化、云标签解析与序列化。
// 由 web/src/pages/project-servers-page.tsx 原样搬迁（RF-08），未改动任何取值与判定顺序。
// 注意：renderCloudTags 返回 JSX，故本文件为 .tsx；其余导出均为纯函数，可直接补单测。
import { Space, Tag } from "antd";

export const CLOUD_PROVIDER_LABEL: Record<string, string> = {
  alibaba: "阿里云",
  tencent: "腾讯云",
  jd: "京东云",
  custom: "自定义",
};

export function mapChargeTypeZh(v: string): string {
  const x = String(v || "").trim().toUpperCase();
  if (!x) return "-";
  const m: Record<string, string> = {
    PREPAID: "包年包月",
    PREPAID_BY_DURATION: "包年包月",
    POSTPAID: "按量付费",
    POSTPAID_BY_USAGE: "按量付费",
    POSTPAID_BY_HOUR: "按小时后付费",
    POSTPAID_BY_DURATION: "按配置后付费",
    CDHPAID: "专有宿主机付费",
  };
  return m[x] || v;
}

export function mapNetworkChargeTypeZh(v: string): string {
  const x = String(v || "").trim().toUpperCase();
  if (!x) return "-";
  const m: Record<string, string> = {
    PAYBYTRAFFIC: "按流量计费",
    PAYBYBANDWIDTH: "按带宽计费",
    TRAFFIC_POSTPAID_BY_HOUR: "按流量后付费",
    BANDWIDTH_POSTPAID_BY_HOUR: "按带宽后付费",
    BANDWIDTH_PREPAID: "带宽预付费",
    BANDWIDTH_PACKAGE: "带宽包计费",
    NORMAL: "正常计费",
    OVERDUE: "已到期",
    ARREAR: "欠费",
  };
  return m[x] || v;
}

export function renderCloudTags(tagsJSON?: string) {
  const raw = String(tagsJSON || "").trim();
  if (!raw) return "-";
  try {
    const obj = JSON.parse(raw) as Record<string, unknown>;
    const entries = Object.entries(obj).filter(([k]) => String(k).trim() !== "");
    if (!entries.length) return "-";
    return (
      <Space size={[4, 4]} wrap>
        {entries.map(([k, v]) => (
          <Tag key={k} color="blue">{`${k}:${String(v ?? "")}`}</Tag>
        ))}
      </Space>
    );
  } catch {
    return raw;
  }
}

export type CloudTagKV = { key: string; value: string };

export function parseCloudTagRows(raw?: string): CloudTagKV[] {
  const text = String(raw || "").trim();
  if (!text) return [];
  try {
    const obj = JSON.parse(text) as Record<string, unknown>;
    return Object.entries(obj)
      .map(([key, value]) => ({ key: String(key || "").trim(), value: String(value ?? "").trim() }))
      .filter((it) => it.key);
  } catch {
    return [];
  }
}

export function buildCloudTagsJSON(rows: CloudTagKV[]): string {
  const obj: Record<string, string> = {};
  rows.forEach((it) => {
    const key = String(it.key || "").trim();
    if (!key) return;
    obj[key] = String(it.value ?? "").trim();
  });
  const keys = Object.keys(obj);
  if (keys.length === 0) return "";
  return JSON.stringify(obj);
}
