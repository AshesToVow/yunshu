import { describe, expect, it } from "vitest";
import { extractApiErrorMessage } from "./http";

describe("extractApiErrorMessage", () => {
  it("falls back when non-axios error", () => {
    expect(extractApiErrorMessage(new Error("boom"), "请求失败")).toBe("boom");
    expect(extractApiErrorMessage("x", "请求失败")).toBe("请求失败");
  });
});
