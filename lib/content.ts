import "server-only";
import { cache } from "react";
import { publicAPI } from "./public-api";
import { slugify } from "./utils";
import type { ContentDoc, Project, SiteContent } from "./types";

export const contentTypes = [
  "pageInfo",
  "projects",
  "note",
  "photograph",
  "nowUpdate",
  "siteSettings",
] as const;
export function normalizeContent(documents: ContentDoc[]): SiteContent {
  const profile = documents.find((d) => d._type === "pageInfo") || {
    _id: "profile",
    _type: "pageInfo" as const,
  };
  const settings = documents.find((d) => d._type === "siteSettings") || {
    _id: "siteSettings",
    _type: "siteSettings" as const,
    homepageTitle: "Software, side quests, and everything in between.",
  };
  const all = documents as (ContentDoc & { url?: string })[];
  const projects = documents
    .filter((d) => d._type === "projects")
    .map((d) => ({
      ...d,
      title: d.title?.trim() || "Untitled project",
      summary: d.summary || "",
      slug: d.slug || { current: slugify(d.title || d._id) },
      technologyNames: (d.technologies || [])
        .map((t) => t.title || all.find((v) => v._id === t._ref)?.title || "")
        .filter(Boolean),
    }))
    .sort((a, b) => (a.order ?? 99) - (b.order ?? 99)) as Project[];
  return {
    profile: {
      ...profile,
      name: profile.name || "Chloe Pratas",
      role: profile.role?.trim() || "Software engineer",
      intro:
        profile.intro ||
        "I build things for the web. Away from the keyboard, you’ll find me with a camera, getting a workout in, or getting lost in a game.",
      bio:
        profile.bio ||
        "I’m Chloe, a software engineer with an interest in the whole picture — how something works, how it feels, and the people using it. Outside of software, photography, fitness, and gaming are part of my world.",
    },
    settings,
    projects,
    notes: documents
      .filter((d) => d._type === "note")
      .sort((a, b) => (b.date || "").localeCompare(a.date || "")),
    photos: documents
      .filter((d) => d._type === "photograph")
      .sort((a, b) => (a.order ?? 99) - (b.order ?? 99)),
    updates: documents
      .filter((d) => d._type === "nowUpdate")
      .sort((a, b) => (a.order ?? 99) - (b.order ?? 99)),
    socials: all
      .filter(
        (d) =>
          (d._type as string) === "social" && /^https?:\/\//.test(d.url || ""),
      )
      .map((d) => ({ title: d.title || "", url: d.url! })),
  };
}
export const getContent = cache(async (): Promise<SiteContent> =>
  normalizeContent(
    (await publicAPI<{ documents: ContentDoc[] }>("/v1/content")).documents,
  ),
);
