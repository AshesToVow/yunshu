import dayjs from "dayjs";
import { describe, expect, it } from "vitest";
import {
  LOG_ALERT_WINDOW_MAX_MINUTES,
  buildLogAlertRulePayload,
  clampLogAlertWindow,
  inferLogAlertWindow,
} from "./log-alert-rule";

describe("clampLogAlertWindow", () => {
  it("缺省/非法值回落到 5 分钟", () => {
    expect(clampLogAlertWindow(undefined)).toBe(5);
    expect(clampLogAlertWindow(0)).toBe(5);
    expect(clampLogAlertWindow(-3)).toBe(5);
  });

  it("不超过后端上限", () => {
    expect(clampLogAlertWindow(24 * 60)).toBe(LOG_ALERT_WINDOW_MAX_MINUTES);
    expect(clampLogAlertWindow(30)).toBe(30);
  });
});

describe("inferLogAlertWindow", () => {
  it("按检索时间范围折算窗口并截断", () => {
    const now = dayjs();
    expect(inferLogAlertWindow([now.subtract(15, "minute"), now])).toBe(15);
    expect(inferLogAlertWindow([now.subtract(7, "day"), now])).toBe(LOG_ALERT_WINDOW_MAX_MINUTES);
    expect(inferLogAlertWindow(null)).toBe(5);
  });
});

describe("buildLogAlertRulePayload", () => {
  it("把检索条件写入日志规则 Expr 与标签", () => {
    const payload = buildLogAlertRulePayload({
      projectId: 7,
      name: " 订单 ERROR 告警 ",
      mode: "error_count",
      level: "error",
      keyword: "timeout",
      serviceName: "order-api",
      namespace: "prod",
      windowMinutes: 30,
      threshold: 20,
      severity: "critical",
      evalIntervalSeconds: 30,
    });

    expect(payload).toMatchObject({
      datasource_id: 0,
      project_id: 7,
      name: "订单 ERROR 告警",
      rule_kind: "log",
      threshold_unit: "count",
      enabled: true,
      eval_interval_seconds: 30,
      severity: "critical",
    });
    expect(JSON.parse(String(payload.expr))).toEqual({
      mode: "error_count",
      level: "ERROR",
      keyword: "timeout",
      service_name: "order-api",
      namespace: "prod",
      window_minutes: 30,
      threshold: 20,
    });
    expect(JSON.parse(String(payload.labels_json))).toEqual({ category: "log", rule_kind: "log" });
    const annotations = JSON.parse(String(payload.annotations_json)) as Record<string, string>;
    expect(annotations.summary).toContain("30 分钟");
    expect(annotations.summary).toContain("20 条");
  });

  it("突增模式补齐基线与倍率", () => {
    const payload = buildLogAlertRulePayload({
      projectId: 1,
      name: "突增",
      mode: "error_spike",
      windowMinutes: 10,
      threshold: 5,
    });
    const expr = JSON.parse(String(payload.expr)) as Record<string, number | string>;
    expect(expr.mode).toBe("error_spike");
    expect(expr.window_minutes).toBe(10);
    expect(expr.baseline_minutes).toBe(60);
    expect(expr.spike_ratio).toBe(3);
  });

  it("窗口超限时按后端上限写入", () => {
    const payload = buildLogAlertRulePayload({
      projectId: 1,
      name: "长窗口",
      mode: "error_count",
      windowMinutes: 1440,
    });
    expect((JSON.parse(String(payload.expr)) as Record<string, number>).window_minutes).toBe(
      LOG_ALERT_WINDOW_MAX_MINUTES,
    );
  });

  it("缺少项目或名称时拒绝构建", () => {
    expect(() => buildLogAlertRulePayload({ name: "x", mode: "error_count" })).toThrow("项目");
    expect(() => buildLogAlertRulePayload({ projectId: 2, name: "  ", mode: "error_count" })).toThrow("规则名称");
  });
});
