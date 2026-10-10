import { getContent } from "@/lib/content";
import { escapeXml, siteUrl } from "@/lib/utils";
export const dynamic = "force-dynamic";
export async function GET() {
  const { notes } = await getContent();
  const xml = `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>Chloe’s notes</title><link>${siteUrl}/notes</link><description>Discoveries, ideas, and occasional tangents.</description><language>en</language>${notes.map((n) => `<item><title>${escapeXml(n.title || "")}</title><link>${siteUrl}/notes/${escapeXml(n.slug?.current || "")}</link><guid>${siteUrl}/notes/${escapeXml(n.slug?.current || "")}</guid><description>${escapeXml(n.summary || "")}</description>${n.date ? `<pubDate>${new Date(n.date).toUTCString()}</pubDate>` : ""}</item>`).join("")}</channel></rss>`;
  return new Response(xml, {
    headers: {
      "Content-Type": "application/rss+xml; charset=utf-8",
      "Cache-Control": "public, max-age=60",
    },
  });
}
