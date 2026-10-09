import type { MetadataRoute } from "next";
import { getContent } from "@/lib/content";
import { siteUrl, contentPath } from "@/lib/utils";
export const dynamic = "force-dynamic";
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const { projects, notes } = await getContent();
  return [
    ...[
      "",
      "/work",
      "/notes",
      "/photography",
      "/about",
      "/now",
      "/playground",
      "/guestbook",
      "/contact",
    ].map((path) => ({ url: `${siteUrl}${path}` })),
    ...[...projects, ...notes].map((d) => ({
      url: `${siteUrl}${contentPath(d)}`,
      lastModified: d._updatedAt ? new Date(d._updatedAt) : undefined,
    })),
  ];
}
