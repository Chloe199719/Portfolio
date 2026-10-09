import { notFound } from "next/navigation";
import { getContent } from "@/lib/content";
import { metadata } from "@/lib/seo";
import { safeJson, siteUrl } from "@/lib/utils";
import { ContentDetail } from "@/components/site/content-detail";
type Props = { params: Promise<{ slug: string }> };
export async function generateMetadata({ params }: Props) {
  const { slug } = await params;
  const n = (await getContent()).notes.find((n) => n.slug?.current === slug);
  return n ? metadata(n.title!, n.summary || "", `/notes/${slug}`) : {};
}
export default async function Note({ params }: Props) {
  const { slug } = await params;
  const { notes, profile } = await getContent();
  const note = notes.find((n) => n.slug?.current === slug);
  if (!note) notFound();
  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: safeJson({
            "@context": "https://schema.org",
            "@type": "BlogPosting",
            headline: note.title,
            datePublished: note.date,
            dateModified: note._updatedAt,
            author: {
              "@type": "Person",
              name: profile.name,
              url: `${siteUrl}/about`,
            },
            mainEntityOfPage: `${siteUrl}/notes/${slug}`,
          }),
        }}
      />
      <ContentDetail doc={note} />
    </>
  );
}
