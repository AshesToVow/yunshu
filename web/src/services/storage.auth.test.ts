import { beforeEach, describe, expect, it } from "vitest";
import { clearAuthStorage, clearToken } from "./storage";

describe("auth token storage (cookie migration)", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("clearToken removes legacy token key", () => {
    window.localStorage.setItem("permission-system-token", "legacy");
    clearToken();
    expect(window.localStorage.getItem("permission-system-token")).toBeNull();
  });

  it("clearAuthStorage removes legacy token key", () => {
    window.localStorage.setItem("permission-system-token", "legacy");
    clearAuthStorage();
    expect(window.localStorage.getItem("permission-system-token")).toBeNull();
  });
});
