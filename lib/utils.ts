import { apiBase } from "./api-config";
import type { ContentDoc, ImageAsset } from "./types";
export const siteUrl = (
  process.env.NEXT_PUBLIC_SITE_URL || "https://www.chloepratas.com"
).replace(/\/$/, "");
export const slugify = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
export function imageUrl(image?: ImageAsset, width = 1200) {
  const ref = image?.asset?._ref;
  void width;
  if (!ref) return undefined;
  if (/^upload-[a-f0-9-]+$/.test(ref)) return `${apiBase}/media/${ref}`;
  if (/^image-[a-zA-Z0-9]+-\d+x\d+-\w+$/.test(ref))
    return `/images/${ref}.webp`;
  return undefined;
}
export function contentPath(doc: ContentDoc) {
  if (doc._type === "projects")
    return `/work/${doc.slug?.current || slugify(doc.title || doc._id)}`;
  if (doc._type === "note")
    return `/notes/${doc.slug?.current || slugify(doc.title || doc._id)}`;
  return (
    {
      pageInfo: "/about",
      photograph: "/photography",
      nowUpdate: "/now",
      siteSettings: "/",
    } as Record<string, string>
  )[doc._type];
}
export function formatDate(value?: string) {
  return value
    ? new Intl.DateTimeFormat("en", {
        month: "long",
        day: "numeric",
        year: "numeric",
        timeZone: "UTC",
      }).format(new Date(value))
    : "";
}
export function safeJson(value: unknown) {
  return JSON.stringify(value).replace(/</g, "\\u003c");
}
export function escapeXml(value: string) {
  return value.replace(
    /[<>&"']/g,
    (c) =>
      ({
        "<": "&lt;",
        ">": "&gt;",
        "&": "&amp;",
        '"': "&quot;",
        "'": "&apos;",
      })[c]!,
  );
}
