import { describe, it, expect } from "vitest";
import { berlinToUTC } from "../lib/scheduling";
describe("Berlin scheduling", () => {
  it("uses the summer and winter offsets", () => {
    expect(berlinToUTC("2026-07-12T14:30")).toBe("2026-07-12T12:30:00.000Z");
    expect(berlinToUTC("2026-12-12T14:30")).toBe("2026-12-12T13:30:00.000Z");
  });
  it("rejects the missing spring daylight-saving hour", () => {
    expect(() => berlinToUTC("2026-03-29T02:30")).toThrow("does not exist");
  });
  it("uses the later occurrence of the repeated autumn hour", () => {
    expect(berlinToUTC("2026-10-25T02:30")).toBe("2026-10-25T01:30:00.000Z");
  });
});
