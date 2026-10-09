import Link from "next/link";
import { ArrowTopRightIcon } from "@radix-ui/react-icons";
import { getContent } from "@/lib/content";
import { formatDate } from "@/lib/utils";
import { metadata as makeMetadata } from "@/lib/seo";
import { PageHeading, EmptyState, TextLink } from "@/components/site/ui";
export const metadata = makeMetadata(
  "Notes",
  "Discoveries, ideas, and occasional tangents from Chloe Pratas.",
  "/notes",
);
export default async function Notes({
  searchParams,
}: {
  searchParams: Promise<{ topic?: string }>;
}) {
  const { notes } = await getContent();
  const { topic } = await searchParams;
  const topics = [...new Set(notes.flatMap((n) => n.tags || []))];
  const filtered = topic ? notes.filter((n) => n.tags?.includes(topic)) : notes;
  return (
    <div className="shell page-bottom">
      <PageHeading eyebrow="From the margins" title="A notebook, of sorts.">
        <p>Things learned. Ideas still forming. The occasional tangent.</p>
        <TextLink href="/feed.xml">Follow via RSS</TextLink>
      </PageHeading>
      {topics.length > 0 && (
        <nav className="topic-filter" aria-label="Filter notes by topic">
          <Link aria-current={!topic ? "page" : undefined} href="/notes">
            All notes
          </Link>
          {topics.map((t) => (
            <Link
              key={t}
              aria-current={topic === t ? "page" : undefined}
              href={`/notes?topic=${encodeURIComponent(t)}`}
            >
              {t}
            </Link>
          ))}
        </nav>
      )}
      {filtered.length ? (
        <div className="note-list">
          {filtered.map((n) => (
            <Link
              href={`/notes/${n.slug?.current}`}
              key={n._id}
              className="note-row"
            >
              <span>{formatDate(n.date)}</span>
              <div>
                <h2>{n.title}</h2>
                <p>{n.summary}</p>
              </div>
              <ArrowTopRightIcon />
            </Link>
          ))}
        </div>
      ) : (
        <EmptyState
          title={
            topic
              ? "No notes on this topic yet."
              : "The first page is still unwritten."
          }
        >
          <p>
            {topic
              ? "Try another topic or explore all notes."
              : "This is where I’ll share ideas, discoveries, and things I want to remember."}
          </p>
        </EmptyState>
      )}
    </div>
  );
}
