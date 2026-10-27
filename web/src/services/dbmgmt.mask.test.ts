import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ getData: vi.fn() }));

vi.mock("./http", () => ({
  http: { get: vi.fn() },
  getData: mocks.getData,
  idempotencyConfig: () => ({}),
}));

import { listDbColumnMaskRules } from "./dbmgmt";

describe("listDbColumnMaskRules", () => {
  beforeEach(() => {
    mocks.getData.mockReset();
  });

  it("解包后端 { list } 包装", async () => {
    mocks.getData.mockResolvedValue({ list: [{ id: 1, table_name: "t_user" }] });
    await expect(listDbColumnMaskRules(1, 2)).resolves.toEqual([{ id: 1, table_name: "t_user" }]);
  });

  it("后端返回空值时回退空数组", async () => {
    mocks.getData.mockResolvedValue(null);
    const rules = await listDbColumnMaskRules(1, 2);
    expect(Array.isArray(rules)).toBe(true);
    expect(rules).toEqual([]);
  });
});
