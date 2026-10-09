import { describe, it, expect } from "vitest";
import {
  contentSchema,
  validatePublish,
  contactSchema,
  guestSchema,
  canDeleteEntry,
} from "@/lib/validation";
import { safeJson, escapeXml, imageUrl } from "@/lib/utils";
import { shuffledDeck } from "@/lib/game";
describe("untrusted inputs", () => {
  it("rejects unsafe links and oversized guestbook messages", () => {
    expect(
      contentSchema.safeParse({
        _type: "projects",
        linkToBuild: "javascript:alert(1)",
      }).success,
    ).toBe(false);
    expect(
      guestSchema.safeParse({ name: "Visitor", message: "x".repeat(501) })
        .success,
    ).toBe(false);
  });
  it("requires valid contact details and rejects the spam field", () => {
    expect(
      contactSchema.safeParse({
        name: "A",
        email: "invalid",
        subject: "Hello",
        message: "A normal message.",
      }).success,
    ).toBe(false);
    expect(
      contactSchema.safeParse({
        name: "A",
        email: "a@example.org",
        subject: "Hello",
        message: "A normal message.",
        website: "spam",
      }).success,
    ).toBe(false);
  });
  it("lets incomplete drafts save but requires accessible publishable photos", () => {
    const draft = contentSchema.parse({
      _type: "photograph",
      title: "A photograph",
    });
    expect(validatePublish(draft)).toContain(
      "Add a photograph and alternative text.",
    );
  });
  it("strips client supplied owner and revision fields", () => {
    expect(
      contentSchema.parse({ _type: "note", owner: true, _rev: "forged" }),
    ).toEqual({ _type: "note" });
  });
  it("fails closed without an owner id and limits author deletion", () => {
    expect(canDeleteEntry({ uid: "a", owner: false }, "b")).toBe(false);
    expect(canDeleteEntry({ uid: "a", owner: true }, "b")).toBe(true);
  });
  it("escapes content embedded in structured data and feeds", () => {
    expect(safeJson({ text: "</script>" })).not.toContain("<");
    expect(escapeXml("<title>&")).toBe("&lt;title&gt;&amp;");
  });
  it("only produces local image paths from supported references", () => {
    expect(
      imageUrl({
        _type: "image",
        asset: { _type: "reference", _ref: "../../secret" },
      }),
    ).toBeUndefined();
    expect(
      imageUrl({
        _type: "image",
        asset: { _type: "reference", _ref: "upload-abcdef" },
      }),
    ).toBe("/media/upload-abcdef");
  });
});
describe("memory game", () => {
  it("always makes sixteen tiles with eight complete pairs", () => {
    for (let i = 0; i < 25; i++) {
      const deck = shuffledDeck();
      expect(deck).toHaveLength(16);
      for (let shape = 0; shape < 8; shape++)
        expect(deck.filter((v) => v === shape)).toHaveLength(2);
    }
  });
});
