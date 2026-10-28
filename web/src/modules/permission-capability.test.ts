import { describe, expect, it } from "vitest";
import {
  buildCapabilities,
  capabilityKey,
  EMPTY_CAPABILITIES,
  hasCapability,
} from "./permission-capability";

describe("capabilityKey", () => {
  it("动作大写、两端空白归一", () => {
    expect(capabilityKey(" post ", " /api/v1/users ")).toBe("POST /api/v1/users");
  });
});

describe("buildCapabilities", () => {
  it("忽略缺字段的条目并去重", () => {
    const caps = buildCapabilities({
      superAdmin: false,
      permissions: [
        { action: "GET", resource: "/api/v1/users" },
        { action: "get", resource: "/api/v1/users" },
        { action: "", resource: "/api/v1/x" },
        { action: "POST", resource: "" },
      ],
    });
    expect(caps.ready).toBe(true);
    expect(caps.keys.size).toBe(1);
    expect(caps.keys.has("GET /api/v1/users")).toBe(true);
  });

  it("permissions 为空时得到空集合而不是未就绪", () => {
    const caps = buildCapabilities({ superAdmin: false, permissions: null });
    expect(caps.ready).toBe(true);
    expect(caps.keys.size).toBe(0);
  });
});

describe("hasCapability", () => {
  it("超管一律放行", () => {
    expect(hasCapability({ superAdmin: true, ready: true, keys: new Set() }, "DELETE", "/api/v1/x")).toBe(true);
  });

  it("能力未加载时放行（后端仍会拦截）", () => {
    expect(hasCapability(EMPTY_CAPABILITIES, "POST", "/api/v1/users")).toBe(true);
  });

  it("已加载时按精确匹配判定", () => {
    const caps = buildCapabilities({
      superAdmin: false,
      permissions: [{ action: "POST", resource: "/api/v1/users" }],
    });
    expect(hasCapability(caps, "POST", "/api/v1/users")).toBe(true);
    expect(hasCapability(caps, "post", "/api/v1/users")).toBe(true);
    expect(hasCapability(caps, "DELETE", "/api/v1/users")).toBe(false);
    expect(hasCapability(caps, "POST", "/api/v1/users/1")).toBe(false);
  });
});
