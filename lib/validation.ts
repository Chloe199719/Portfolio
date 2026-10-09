import { z } from "zod";
const text = (max: number) => z.string().trim().max(max);
const link = z
  .string()
  .url()
  .refine((v) => /^https?:\/\//.test(v), "Use an http or https link.");
const image = z.object({
  _type: z.literal("image"),
  asset: z.object({
    _type: z.literal("reference"),
    _ref: z
      .string()
      .regex(/^(?:image-[a-zA-Z0-9]+-\d+x\d+-\w+|upload-[a-f0-9-]+)$/),
  }),
  alt: text(500).optional(),
  caption: text(1000).optional(),
});
const span = z.object({
  _type: z.literal("span"),
  _key: text(100),
  text: z.string().max(30000),
  marks: z.array(text(100)).max(20).default([]),
});
const block = z.object({
  _type: z.literal("block"),
  _key: text(100),
  style: z.enum(["normal", "h2", "h3", "blockquote"]).default("normal"),
  listItem: z.enum(["bullet", "number"]).optional(),
  level: z.number().int().min(1).max(6).optional(),
  markDefs: z
    .array(z.object({ _type: z.literal("link"), _key: text(100), href: link }))
    .max(100)
    .default([]),
  children: z.array(span).max(1000),
});
export const contentSchema = z.object({
  _type: z.enum([
    "pageInfo",
    "projects",
    "note",
    "photograph",
    "nowUpdate",
    "siteSettings",
  ]),
  title: text(160).optional(),
  slug: z
    .object({
      _type: z.literal("slug").optional(),
      current: z
        .string()
        .regex(/^[a-z0-9]+(?:-[a-z0-9]+)*$/)
        .max(160),
    })
    .optional(),
  summary: text(1500).optional(),
  body: z
    .array(z.union([block, image.extend({ _key: text(100) })]))
    .max(500)
    .optional(),
  image: image.optional(),
  heroImage: image.optional(),
  profilePic: image.optional(),
  featured: z.boolean().optional(),
  order: z.number().int().min(0).max(9999).optional(),
  date: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/)
    .refine((v) => !Number.isNaN(Date.parse(v)))
    .optional(),
  category: text(80).optional(),
  tags: z.array(text(40)).max(15).optional(),
  location: text(150).optional(),
  linkToBuild: z.union([link, z.literal("")]).optional(),
  projectRole: text(500).optional(),
  problem: text(5000).optional(),
  decisions: text(8000).optional(),
  outcome: text(5000).optional(),
  name: text(100).optional(),
  role: text(100).optional(),
  email: z.union([z.string().email(), z.literal("")]).optional(),
  intro: text(1500).optional(),
  bio: text(8000).optional(),
  interests: z.array(text(80)).max(15).optional(),
  homepageTitle: text(200).optional(),
  homepageSubtitle: text(1000).optional(),
});
export function validatePublish(doc: z.infer<typeof contentSchema>) {
  const issues: string[] = [];
  if (
    ["projects", "note", "photograph", "nowUpdate"].includes(doc._type) &&
    !doc.title
  )
    issues.push("Add a title.");
  if (["projects", "note"].includes(doc._type) && !doc.slug?.current)
    issues.push("Add a URL slug.");
  if (doc._type === "note" && (!doc.body?.length || !doc.date))
    issues.push("Add article content and a publication date.");
  if (doc._type === "photograph" && (!doc.image || !doc.image.alt))
    issues.push("Add a photograph and alternative text.");
  if (doc.image && !doc.image.alt)
    issues.push("Add alternative text for the image.");
  if (doc.body?.some((b) => b._type === "image" && !("alt" in b && b.alt)))
    issues.push("Add alternative text for every article image.");
  if (doc._type === "nowUpdate" && (!doc.summary || !doc.date))
    issues.push("Add an update and date.");
  return issues;
}
export const contactSchema = z.object({
  name: text(100).min(1, "Enter your name."),
  email: z.string().trim().email().max(254),
  subject: text(200).min(1, "Add a subject."),
  message: text(5000).min(10, "Write at least 10 characters."),
  website: z.string().max(0).optional(),
});
export const guestSchema = z.object({
  name: text(60).min(1, "Add a public name."),
  message: text(500).min(2, "Write a short message."),
});
export function canDeleteEntry(
  user: { uid: string; owner: boolean },
  authorUid: string,
) {
  return user.owner || user.uid === authorUid;
}
