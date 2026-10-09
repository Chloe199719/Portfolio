import { afterEach, beforeEach, expect, it, vi } from "vitest";
beforeEach(() => {
  vi.resetModules();
  vi.stubEnv("NEXT_PUBLIC_API_URL", "https://api.example.test");
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});
it.each(["not-json", "{}", '{"ok":false}', "[]", "null"])(
  "rejects an unconfirmed successful mutation response: %s",
  async (body) => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(body, { status: 200 })),
    );
    const { request } = await import("../lib/client-api");
    await expect(request("/v1/contact", { method: "POST" })).rejects.toThrow(
      "invalid response",
    );
  },
);
it("rejects a malformed saved document", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response('{"document":{}}', { status: 200 })),
  );
  const { request } = await import("../lib/client-api");
  await expect(
    request("/v1/admin/content/example", { method: "POST" }),
  ).rejects.toThrow("invalid document");
});
it("accepts an explicit successful submission", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response('{"ok":true}', { status: 201 })),
  );
  const { request } = await import("../lib/client-api");
  await expect(request("/v1/contact", { method: "POST" })).resolves.toEqual({
    ok: true,
  });
});
