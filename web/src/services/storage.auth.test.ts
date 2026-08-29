import { beforeEach, describe, expect, it } from "vitest";
import { clearAuthStorage, clearToken, getToken, setToken } from "./storage";

describe("auth token storage (cookie migration)", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("does not persist JWT in localStorage", () => {
    setToken("abc.def.ghi");
    expect(getToken()).toBe("");
    expect(window.localStorage.getItem("permission-system-token")).toBeNull();
  });

  it("clearAuthStorage removes legacy token key", () => {
    window.localStorage.setItem("permission-system-token", "legacy");
    clearToken();
    clearAuthStorage();
    expect(window.localStorage.getItem("permission-system-token")).toBeNull();
    expect(getToken()).toBe("");
  });
});
