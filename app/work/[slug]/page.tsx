import { notFound } from "next/navigation";
import { getContent } from "@/lib/content";
import { metadata } from "@/lib/seo";
import { ContentDetail } from "@/components/site/content-detail";
type Props = { params: Promise<{ slug: string }> };
export async function generateMetadata({ params }: Props) {
  const { slug } = await params;
  const p = (await getContent()).projects.find((p) => p.slug.current === slug);
  return p ? metadata(p.title, p.summary, `/work/${slug}`) : {};
}
export default async function ProjectPage({ params }: Props) {
  const { slug } = await params;
  const project = (await getContent()).projects.find(
    (p) => p.slug.current === slug,
  );
  if (!project) notFound();
  return <ContentDetail doc={project} />;
}
